//go:build restrita

// Evento de SST (S-2220, ASO) gerado pela própria interface (internal/web) e enviado à PRODUÇÃO RESTRITA.
// Sem o S-2200 do trabalhador cadastrado antes, o eSocial recusa por regra de negócio — o que este teste
// confere é que o XML passa no esquema e a assinatura é aceita (a recusa esperada não é de esquema nem de
// assinatura). Mesmas variáveis do restrita_integracao_test.go.
package soap_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/forg3/esocial-emissor-livre/internal/crypto"
	"github.com/forg3/esocial-emissor-livre/internal/soap"
	"github.com/forg3/esocial-emissor-livre/internal/web"
)

func TestRestritaEnviaS2220(t *testing.T) {
	caminho, senha := os.Getenv("ESOCIAL_PFX"), os.Getenv("ESOCIAL_PFX_SENHA")
	if caminho == "" || senha == "" {
		t.Skip("defina ESOCIAL_PFX e ESOCIAL_PFX_SENHA")
	}
	cert, err := crypto.CarregarA1Arquivo(caminho, senha)
	if err != nil {
		t.Fatalf("certificado: %v", err)
	}
	cliente, err := soap.NovoClienteSOAP(cert)
	if err != nil {
		t.Fatalf("cliente SOAP: %v", err)
	}
	hoje := time.Now().Format("2006-01-02")
	xmlEv := web.GerarXMLS2220(web.ParametrosS2220{
		ID: web.GerarIDEvento(cert.CNPJ()), Ambiente: 2, CNPJ: cert.CNPJ(),
		CPFTrabalhador: "52998224725", Matricula: "TESTE-0001", TipoExame: "0", DataASO: hoje,
		ResultadoASO: "1", ProcRealizado: "0295", OrdExame: "1", IndResult: "1",
		NomeMedico: "Medico de Teste", CRMMedico: "12345", UFMedico: "PR",
		NomeCoord: "Coordenador de Teste", CPFCoord: "11144477735", CRMCoord: "54321", UFCoord: "PR",
	})
	assinado, err := crypto.AssinarXML([]byte(xmlEv), cert)
	if err != nil {
		t.Fatalf("assinar: %v", err)
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancelar()
	env, err := cliente.EnviarLote(ctx, assinado, 2)
	if err != nil {
		t.Fatalf("envio: %v", err)
	}
	t.Logf("envio: código %d — %s · protocolo %s", env.CodigoResposta, env.DescricaoResposta, env.ProtocoloEnvio)
	if env.ProtocoloEnvio == "" {
		for _, o := range env.Ocorrencias {
			t.Logf("  ocorrência %d em %s: %s", o.Codigo, o.Localizacao, o.Descricao)
		}
		t.Fatal("lote recusado na recepção")
	}
	time.Sleep(45 * time.Second)
	c, err := cliente.ConsultarLote(ctx, env.ProtocoloEnvio, 2)
	if err != nil {
		t.Fatalf("consulta: %v", err)
	}
	t.Logf("consulta: código %d — %s", c.CodigoResposta, c.DescricaoResposta)
	for _, r := range c.Eventos {
		t.Logf("  evento %s: código %d — %s · recibo %s", r.IdEvento, r.CodigoResposta, r.DescricaoResposta, r.NumeroRecibo)
		for _, o := range r.Ocorrencias {
			t.Logf("    ocorrência %d (tipo %d) em %s: %s", o.Codigo, o.Tipo, o.Localizacao, o.Descricao)
			if strings.Contains(strings.ToLower(o.Descricao), "assinatura") || strings.Contains(strings.ToLower(o.Descricao), "schema") {
				t.Errorf("recusa de assinatura ou de esquema: %s", o.Descricao)
			}
		}
	}
}
