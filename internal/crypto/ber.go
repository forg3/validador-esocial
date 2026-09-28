package crypto

import (
	"errors"
	"fmt"
)

// berParaDER converte uma estrutura ASN.1 em BER para DER: tamanho indefinido (0x80 … 00 00) vira
// definido e OCTET STRING construída (em pedaços) vira primitiva com os pedaços concatenados — os bytes
// de conteúdo não mudam. Usada na leitura de .pfx do ICP-Brasil (pkcs12ber.go).
func berParaDER(ber []byte) ([]byte, error) {
	der, resto, err := converterTLV(ber, 0)
	if err != nil {
		return nil, err
	}
	if len(resto) != 0 {
		return nil, fmt.Errorf("BER: %d bytes sobrando após a estrutura", len(resto))
	}
	return der, nil
}

const profundidadeMaxima = 64 // ASN.1 malformado não pode estourar a pilha

func converterTLV(b []byte, prof int) (der []byte, resto []byte, err error) {
	if prof > profundidadeMaxima {
		return nil, nil, errors.New("BER: aninhamento profundo demais")
	}
	if len(b) < 2 {
		return nil, nil, errors.New("BER: estrutura truncada")
	}
	i := 1 // identificador de 1 ou mais bytes
	if b[0]&0x1f == 0x1f {
		for i < len(b) && b[i]&0x80 != 0 {
			i++
		}
		i++
	}
	if i >= len(b) {
		return nil, nil, errors.New("BER: tag truncada")
	}
	tag, construida := b[:i], b[0]&0x20 != 0

	indefinido := b[i] == 0x80
	var conteudo []byte
	if indefinido {
		if !construida {
			return nil, nil, errors.New("BER: tamanho indefinido em tipo primitivo")
		}
		resto = b[i+1:]
	} else {
		n, tamLen, err := lerTamanho(b[i:])
		if err != nil {
			return nil, nil, err
		}
		ini := i + tamLen
		if n < 0 || ini+n > len(b) {
			return nil, nil, errors.New("BER: conteúdo além do fim")
		}
		conteudo, resto = b[ini:ini+n], b[ini+n:]
	}
	if !construida {
		return montarTLV(tag, conteudo), resto, nil
	}

	var filhos [][]byte
	fonte := conteudo
	if indefinido {
		fonte = resto
	}
	for {
		if indefinido {
			if len(fonte) >= 2 && fonte[0] == 0 && fonte[1] == 0 {
				resto = fonte[2:]
				break
			}
			if len(fonte) == 0 {
				return nil, nil, errors.New("BER: fim de conteúdo (00 00) ausente")
			}
		} else if len(fonte) == 0 {
			break
		}
		filho, prox, err := converterTLV(fonte, prof+1)
		if err != nil {
			return nil, nil, err
		}
		filhos = append(filhos, filho)
		fonte = prox
	}

	if len(tag) == 1 && tag[0] == 0x24 { // OCTET STRING construída → primitiva
		var junto []byte
		for _, f := range filhos {
			v, err := valorDER(f)
			if err != nil {
				return nil, nil, err
			}
			junto = append(junto, v...)
		}
		return montarTLV([]byte{0x04}, junto), resto, nil
	}
	var corpo []byte
	for _, f := range filhos {
		corpo = append(corpo, f...)
	}
	return montarTLV(tag, corpo), resto, nil
}

func lerTamanho(b []byte) (n, usados int, err error) {
	if len(b) == 0 {
		return 0, 0, errors.New("BER: tamanho truncado")
	}
	if b[0]&0x80 == 0 {
		return int(b[0]), 1, nil
	}
	k := int(b[0] & 0x7f)
	if k == 0 || k > 4 || len(b) < 1+k {
		return 0, 0, errors.New("BER: tamanho inválido")
	}
	for _, x := range b[1 : 1+k] {
		n = n<<8 | int(x)
	}
	return n, 1 + k, nil
}

// valorDER devolve o conteúdo de um TLV já em DER.
func valorDER(tlv []byte) ([]byte, error) {
	i := 1
	if tlv[0]&0x1f == 0x1f {
		for i < len(tlv) && tlv[i]&0x80 != 0 {
			i++
		}
		i++
	}
	n, usados, err := lerTamanho(tlv[i:])
	if err != nil {
		return nil, err
	}
	return tlv[i+usados : i+usados+n], nil
}

func montarTLV(tag, valor []byte) []byte {
	out := append([]byte{}, tag...)
	n := len(valor)
	if n < 0x80 {
		out = append(out, byte(n))
	} else {
		var bytesTam []byte
		for x := n; x > 0; x >>= 8 {
			bytesTam = append([]byte{byte(x)}, bytesTam...)
		}
		out = append(out, 0x80|byte(len(bytesTam)))
		out = append(out, bytesTam...)
	}
	return append(out, valor...)
}
