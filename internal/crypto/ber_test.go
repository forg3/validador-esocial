package crypto

import (
	"bytes"
	"encoding/asn1"
	"errors"
	"testing"
)

func TestBERParaDERTamanhoIndefinidoEOctetEmPedacos(t *testing.T) {
	// SEQUENCE indefinida { INTEGER 3, OCTET STRING construída { "ab", "cd" } }
	ber := []byte{0x30, 0x80, 0x02, 0x01, 0x03, 0x24, 0x80, 0x04, 0x02, 'a', 'b', 0x04, 0x02, 'c', 'd', 0x00, 0x00, 0x00, 0x00}
	der, err := berParaDER(ber)
	if err != nil {
		t.Fatal(err)
	}
	esperado := []byte{0x30, 0x09, 0x02, 0x01, 0x03, 0x04, 0x04, 'a', 'b', 'c', 'd'}
	if !bytes.Equal(der, esperado) {
		t.Fatalf("der = % x\nesperado % x", der, esperado)
	}
	if de2, _ := berParaDER(der); !bytes.Equal(de2, der) {
		t.Error("DER deveria passar sem mudança")
	}
	if _, err := berParaDER([]byte{0x30, 0x80, 0x02, 0x01}); err == nil {
		t.Error("BER truncado deveria dar erro")
	}
}

// bERdoPFX reescreve um .pfx DER no estilo do ICP-Brasil: casca e ContentInfo com tamanho indefinido e o
// conteúdo protegido pelo MAC numa OCTET STRING em dois pedaços. Os bytes protegidos não mudam.
func bERdoPFX(t *testing.T, der []byte) []byte {
	var pfx struct {
		Version  asn1.RawValue
		AuthSafe struct {
			ContentType asn1.RawValue
			Content     asn1.RawValue `asn1:"tag:0,explicit"`
		}
		MacData asn1.RawValue
	}
	if _, err := asn1.Unmarshal(der, &pfx); err != nil {
		t.Fatal(err)
	}
	var interno []byte
	if _, err := asn1.Unmarshal(pfx.AuthSafe.Content.Bytes, &interno); err != nil {
		t.Fatal(err)
	}
	meio := len(interno) / 2
	p1, _ := asn1.Marshal(interno[:meio])
	p2, _ := asn1.Marshal(interno[meio:])
	var b bytes.Buffer
	b.Write([]byte{0x30, 0x80})
	b.Write(pfx.Version.FullBytes)
	b.Write([]byte{0x30, 0x80})
	b.Write(pfx.AuthSafe.ContentType.FullBytes)
	b.Write([]byte{0xa0, 0x80, 0x24, 0x80})
	b.Write(p1)
	b.Write(p2)
	b.Write([]byte{0, 0, 0, 0, 0, 0})
	b.Write(pfx.MacData.FullBytes)
	b.Write([]byte{0, 0})
	return b.Bytes()
}

func TestCarregarA1EmBERComoOsDoICPBrasil(t *testing.T) {
	senha := senhaAleatoriaTeste(t)
	der, _, folha := gerarPFXTeste(t, senha, "EMPRESA TESTE:12345678000195", "EMPRESA TESTE")
	ber := bERdoPFX(t, der)
	if _, err := carregarA1DER(ber, senha); err == nil {
		t.Fatal("a leitura direta deveria recusar BER (senão o teste não prova nada)")
	}
	cert, err := CarregarA1(ber, senha)
	if err != nil {
		t.Fatalf("BER não carregou: %v", err)
	}
	if !cert.CertificadoFolha().Equal(folha) {
		t.Error("certificado carregado é outro")
	}
	if _, err := CarregarA1(ber, senha+"x"); !errors.Is(err, errSenhaPFX) {
		t.Errorf("senha errada deveria dar errSenhaPFX, deu %v", err)
	}
}
