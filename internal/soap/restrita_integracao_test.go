//go:build restrita

// Teste real contra a PRODUÇÃO RESTRITA do eSocial (ambiente de testes do governo, tpAmb=2), com certificado
// A1 de verdade. Não roda na suíte normal: exige a tag `restrita` e as variáveis abaixo.
//
//	ESOCIAL_PFX=/caminho/do/e-cnpj.pfx ESOCIAL_PFX_SENHA=... go test -tags restrita -run Restrita -v ./internal/soap/
//
// Opcional: ESOCIAL_CLASSTRIB (Tabela 08; padrão "99") e ESOCIAL_PROTOCOLO para só consultar um lote já enviado.
// Dados enviados à produção restrita não têm efeito jurídico. A senha nunca é impressa.
package soap

import (
	"context"
	"io"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/forg3/esocial-emissor-livre/internal/crypto"
	"github.com/forg3/esocial-emissor-livre/internal/esocial"
)

const ambienteRestrita = 2

func TestRestritaEnviaS1000EConsulta(t *testing.T) {
	caminho, senha := os.Getenv("ESOCIAL_PFX"), os.Getenv("ESOCIAL_PFX_SENHA")
	if caminho == "" || senha == "" {
		t.Skip("defina ESOCIAL_PFX e ESOCIAL_PFX_SENHA")
	}
	cert, err := crypto.CarregarA1Arquivo(caminho, senha)
	if err != nil {
		t.Fatalf("certificado: %v", err)
	}
	t.Logf("certificado: %s · CNPJ %s · válido até %s", cert.RazaoSocial(), cert.CNPJ(), cert.ValidoAte().Format("02/01/2006"))

	cliente, err := NovoClienteSOAP(cert)
	if err != nil {
		t.Fatalf("cliente SOAP: %v", err)
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancelar()

	protocolo := os.Getenv("ESOCIAL_PROTOCOLO")
	if protocolo == "" {
		classTrib := os.Getenv("ESOCIAL_CLASSTRIB")
		if classTrib == "" {
			classTrib = "99"
		}
		raiz := cert.CNPJ()[:8]
		ev := esocial.NovoEventoS1000Inclusao(esocial.GerarIDEvento(esocial.TpInscCNPJ, raiz, 1),
			ambienteRestrita, esocial.TpInscCNPJ, raiz, time.Now().Format("2006-01"), classTrib, 0, 0)
		xmlEv, err := esocial.GerarXMLS1000(ev)
		if err != nil {
			t.Fatalf("gerar S-1000: %v", err)
		}
		assinado, err := crypto.AssinarXML(xmlEv, cert)
		if err != nil {
			t.Fatalf("assinar: %v", err)
		}
		env, err := cliente.EnviarLote(ctx, assinado, ambienteRestrita)
		if err != nil {
			t.Fatalf("envio à produção restrita: %v", err)
		}
		t.Logf("envio: código %d — %s · protocolo %s", env.CodigoResposta, env.DescricaoResposta, env.ProtocoloEnvio)
		for _, o := range env.Ocorrencias {
			t.Logf("  ocorrência %d (tipo %d) em %s: %s", o.Codigo, o.Tipo, o.Localizacao, o.Descricao)
		}
		if env.ProtocoloEnvio == "" {
			t.Fatalf("o eSocial não devolveu protocolo (lote recusado na recepção)")
		}
		protocolo = env.ProtocoloEnvio
		time.Sleep(45 * time.Second) // o eSocial processa o lote de forma assíncrona
	}

	for tentativa := 1; tentativa <= 4; tentativa++ {
		c, err := cliente.ConsultarLote(ctx, protocolo, ambienteRestrita)
		if err != nil {
			t.Fatalf("consulta: %v", err)
		}
		t.Logf("consulta %d: código %d — %s", tentativa, c.CodigoResposta, c.DescricaoResposta)
		if c.CodigoResposta == 0 && os.Getenv("ESOCIAL_XML_BRUTO") != "" {
			t.Logf("resposta bruta da consulta:\n%s", c.XMLBruto)
		}
		if c.CodigoResposta != 101 { // 101 = lote aguardando processamento
			for _, o := range c.Ocorrencias {
				t.Logf("  ocorrência %d (tipo %d) em %s: %s", o.Codigo, o.Tipo, o.Localizacao, o.Descricao)
			}
			for _, r := range c.Eventos {
				t.Logf("  evento %s: código %d — %s · recibo %s", r.IdEvento, r.CodigoResposta, r.DescricaoResposta, r.NumeroRecibo)
				for _, o := range r.Ocorrencias {
					t.Logf("    ocorrência %d (tipo %d) em %s: %s", o.Codigo, o.Tipo, o.Localizacao, o.Descricao)
				}
			}
			return
		}
		time.Sleep(30 * time.Second)
	}
	t.Logf("lote ainda em processamento; consulte depois com ESOCIAL_PROTOCOLO=%s", protocolo)
}

// TestRestritaWSDL imprime as ações SOAP e os namespaces que o eSocial publica nos dois serviços (o WSDL
// exige o certificado). Serve para conferir as constantes deste pacote quando o eSocial mudar versão.
func TestRestritaWSDL(t *testing.T) {
	caminho, senha := os.Getenv("ESOCIAL_PFX"), os.Getenv("ESOCIAL_PFX_SENHA")
	if caminho == "" || senha == "" {
		t.Skip("defina ESOCIAL_PFX e ESOCIAL_PFX_SENHA")
	}
	cert, err := crypto.CarregarA1Arquivo(caminho, senha)
	if err != nil {
		t.Fatalf("certificado: %v", err)
	}
	cliente, err := NovoClienteSOAP(cert)
	if err != nil {
		t.Fatalf("cliente SOAP: %v", err)
	}
	re := regexp.MustCompile(`(soapAction|targetNamespace|namespace)="([^"]+)"`)
	for _, url := range []string{URLRestritaEnvio, URLRestritaConsulta} {
		resp, err := cliente.httpClient.Get(url + "?singleWsdl")
		if err != nil {
			t.Fatalf("%s: %v", url, err)
		}
		corpo, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		vistos := map[string]bool{}
		t.Logf("%s → HTTP %d", url, resp.StatusCode)
		for _, m := range re.FindAllStringSubmatch(string(corpo), -1) {
			if !vistos[m[0]] {
				vistos[m[0]] = true
				t.Logf("  %s = %s", m[1], m[2])
			}
		}
	}
}
