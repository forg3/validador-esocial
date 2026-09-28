package crypto

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/asn1"
	"errors"
	"fmt"
	"hash"
	"math/big"
	"unicode/utf16"
)

// Leitura de .pfx em BER (tamanho indefinido, OCTET STRING em pedaços) — o formato em que muitos A1 do
// ICP-Brasil saem da AC ou do Windows. As bibliotecas de PKCS#12 do Go só aceitam DER. Aqui o arquivo é
// reescrito em DER, em memória: (1) a casca vira DER sem mexer nos bytes protegidos; (2) a senha é
// conferida pelo MAC original, sobre os bytes exatos do arquivo; (3) o conteúdo interno vira DER; (4) o
// MAC é recalculado com a mesma senha, sal e iterações. O resultado segue para a decodificação normal.

type pfxBER struct {
	Version  int
	AuthSafe contentInfoBER
	MacData  macDataBER `asn1:"optional"`
}

type contentInfoBER struct {
	ContentType asn1.ObjectIdentifier
	Content     asn1.RawValue `asn1:"tag:0,explicit,optional"`
}

type macDataBER struct {
	Mac        digestInfoBER
	MacSalt    []byte
	Iterations int `asn1:"optional,default:1"`
}

type digestInfoBER struct {
	Algorithm struct {
		Algorithm  asn1.ObjectIdentifier
		Parameters asn1.RawValue `asn1:"optional"`
	}
	Digest []byte
}

var (
	oidDataPKCS7 = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 7, 1}
	oidSHA1MAC   = asn1.ObjectIdentifier{1, 3, 14, 3, 2, 26}
	oidSHA256MAC = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 1}

	errSenhaPFX = errors.New("senha do certificado incorreta")
)

// pfxBERParaDER devolve o mesmo .pfx em DER, com o MAC refeito. Erro errSenhaPFX se a senha não confere.
func pfxBERParaDER(ber []byte, senha string) ([]byte, error) {
	casca, err := berParaDER(ber)
	if err != nil {
		return nil, err
	}
	var pfx pfxBER
	if _, err := asn1.Unmarshal(casca, &pfx); err != nil {
		return nil, fmt.Errorf("PKCS#12: estrutura inválida: %w", err)
	}
	if !pfx.AuthSafe.ContentType.Equal(oidDataPKCS7) || len(pfx.MacData.Mac.Digest) == 0 {
		return nil, errors.New("PKCS#12 sem MAC de senha (formato não suportado)")
	}
	var interno []byte // os bytes protegidos pelo MAC (conteúdo da OCTET STRING)
	if _, err := asn1.Unmarshal(pfx.AuthSafe.Content.Bytes, &interno); err != nil {
		return nil, fmt.Errorf("PKCS#12: conteúdo inválido: %w", err)
	}

	novoHash, tamanho, err := hashDoMAC(pfx.MacData.Mac.Algorithm.Algorithm)
	if err != nil {
		return nil, err
	}
	senhaBMP := senhaBMPString(senha)
	chave := pbkdfPKCS12(novoHash, tamanho, 64, pfx.MacData.MacSalt, senhaBMP, pfx.MacData.Iterations, 3, tamanho)
	if subtle.ConstantTimeCompare(macPKCS12(novoHash, chave, interno), pfx.MacData.Mac.Digest) != 1 {
		return nil, errSenhaPFX
	}

	internoDER, err := berParaDER(interno)
	if err != nil {
		return nil, err
	}
	if internoDER, err = normalizarAuthSafe(internoDER); err != nil {
		return nil, err
	}
	pfx.MacData.Mac.Digest = macPKCS12(novoHash, chave, internoDER)
	octet, err := asn1.Marshal(internoDER)
	if err != nil {
		return nil, err
	}
	pfx.AuthSafe.Content = asn1.RawValue{Class: asn1.ClassContextSpecific, Tag: 0, IsCompound: true, Bytes: octet}
	return asn1.Marshal(pfx)
}

var oidEncryptedDataPKCS7 = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 7, 6}

type encryptedDataBER struct {
	Version int
	Info    struct {
		ContentType asn1.ObjectIdentifier
		Algoritmo   asn1.RawValue
		Conteudo    asn1.RawValue `asn1:"tag:0,optional"`
	}
}

// normalizarAuthSafe ajusta o que o conversor genérico não pode decidir sozinho: (a) SafeContents dentro
// de ContentInfo "data" (bytes opacos numa OCTET STRING) também em BER; (b) encryptedContent [0] IMPLICIT
// OCTET STRING em pedaços (construída) — em DER ela é primitiva com os pedaços juntos.
func normalizarAuthSafe(der []byte) ([]byte, error) {
	var infos []contentInfoBER
	if _, err := asn1.Unmarshal(der, &infos); err != nil {
		return nil, fmt.Errorf("PKCS#12: AuthenticatedSafe inválido: %w", err)
	}
	for i, ci := range infos {
		switch {
		case ci.ContentType.Equal(oidDataPKCS7):
			var dados []byte
			if _, err := asn1.Unmarshal(ci.Content.Bytes, &dados); err != nil {
				return nil, fmt.Errorf("PKCS#12: SafeContents inválido: %w", err)
			}
			if convertido, err := berParaDER(dados); err == nil {
				dados = convertido
			}
			octet, err := asn1.Marshal(dados)
			if err != nil {
				return nil, err
			}
			infos[i].Content = asn1.RawValue{Class: asn1.ClassContextSpecific, Tag: 0, IsCompound: true, Bytes: octet}
		case ci.ContentType.Equal(oidEncryptedDataPKCS7):
			var ed encryptedDataBER
			if _, err := asn1.Unmarshal(ci.Content.Bytes, &ed); err != nil {
				return nil, fmt.Errorf("PKCS#12: EncryptedData inválido: %w", err)
			}
			if ed.Info.Conteudo.IsCompound {
				var junto []byte
				for resto := ed.Info.Conteudo.Bytes; len(resto) > 0; {
					var pedaco []byte
					var err error
					if resto, err = asn1.Unmarshal(resto, &pedaco); err != nil {
						return nil, fmt.Errorf("PKCS#12: conteúdo cifrado inválido: %w", err)
					}
					junto = append(junto, pedaco...)
				}
				ed.Info.Conteudo = asn1.RawValue{Class: asn1.ClassContextSpecific, Tag: 0, Bytes: junto}
			}
			seq, err := asn1.Marshal(ed)
			if err != nil {
				return nil, err
			}
			infos[i].Content = asn1.RawValue{Class: asn1.ClassContextSpecific, Tag: 0, IsCompound: true, Bytes: seq}
		}
	}
	return asn1.Marshal(infos)
}

func hashDoMAC(oid asn1.ObjectIdentifier) (func() hash.Hash, int, error) {
	switch {
	case oid.Equal(oidSHA1MAC):
		return sha1.New, sha1.Size, nil
	case oid.Equal(oidSHA256MAC):
		return sha256.New, sha256.Size, nil
	}
	return nil, 0, fmt.Errorf("PKCS#12: algoritmo de MAC não suportado (%v)", oid)
}

func macPKCS12(h func() hash.Hash, chave, msg []byte) []byte {
	m := hmac.New(h, chave)
	m.Write(msg)
	return m.Sum(nil)
}

// senhaBMPString: senha em UTF-16BE terminada em dois zeros (RFC 7292, apêndice B.1).
func senhaBMPString(s string) []byte {
	out := []byte{}
	for _, r := range utf16.Encode([]rune(s)) {
		out = append(out, byte(r>>8), byte(r))
	}
	return append(out, 0, 0)
}

// pbkdfPKCS12 é a derivação de chave do PKCS#12 (RFC 7292, apêndice B.2); id 3 = chave do MAC.
// u = tamanho do hash, v = tamanho do bloco (64 para SHA-1 e SHA-256).
func pbkdfPKCS12(h func() hash.Hash, u, v int, sal, senha []byte, iteracoes int, id byte, tamanho int) []byte {
	repetir := func(p []byte) []byte {
		if len(p) == 0 {
			return nil
		}
		n := v * ((len(p) + v - 1) / v)
		out := make([]byte, n)
		for i := range out {
			out[i] = p[i%len(p)]
		}
		return out
	}
	D := make([]byte, v)
	for i := range D {
		D[i] = id
	}
	I := append(repetir(sal), repetir(senha)...)
	um := big.NewInt(1)
	var A []byte
	for len(A) < tamanho {
		m := h()
		m.Write(D)
		m.Write(I)
		Ai := m.Sum(nil)
		for j := 1; j < iteracoes; j++ {
			m = h()
			m.Write(Ai)
			Ai = m.Sum(nil)
		}
		A = append(A, Ai...)
		if len(A) >= tamanho {
			break
		}
		B := repetir(Ai)[:v]
		Bmais1 := new(big.Int).Add(new(big.Int).SetBytes(B), um)
		for k := 0; k < len(I); k += v {
			Ij := new(big.Int).SetBytes(I[k : k+v])
			Ij.Add(Ij, Bmais1)
			b := Ij.Bytes()
			switch {
			case len(b) > v:
				b = b[len(b)-v:]
			case len(b) < v:
				b = append(make([]byte, v-len(b)), b...)
			}
			copy(I[k:k+v], b)
		}
	}
	return A[:tamanho]
}
