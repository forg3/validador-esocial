//go:build restrita

// Fluxo completo de SST na PRODUÇÃO RESTRITA, com os geradores da interface (internal/web): S-2200
// cadastra um trabalhador fictício (CPF válido gerado a cada execução) e, com o recibo dele, o S-2220 (ASO)
// do mesmo vínculo tem de voltar com recibo. Mesmas variáveis do restrita_integracao_test.go; exige o
// S-1000 do empregador já aceito na produção restrita.
package soap_test

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/forg3/esocial-emissor-livre/internal/crypto"
	"github.com/forg3/esocial-emissor-livre/internal/soap"
	"github.com/forg3/esocial-emissor-livre/internal/web"
)

func cpfValidoAleatorio() string {
	d := make([]int, 9, 11)
	for i := range d {
		d[i] = rand.Intn(10)
	}
	for _, peso := range []int{10, 11} {
		soma := 0
		for i, v := range d {
			soma += v * (peso - i)
		}
		dv := soma * 10 % 11
		if dv == 10 {
			dv = 0
		}
		d = append(d, dv)
	}
	s := ""
	for _, v := range d {
		s += fmt.Sprint(v)
	}
	return s
}

// enviarEConsultar manda um evento assinado e devolve o recibo (vazio se recusado), registrando tudo.
func enviarEConsultar(t *testing.T, cliente *soap.ClienteSOAP, cert crypto.Certificado, nome string, xmlEv []byte) string {
	t.Helper()
	assinado, err := crypto.AssinarXML(xmlEv, cert)
	if err != nil {
		t.Fatalf("%s: assinar: %v", nome, err)
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancelar()
	env, err := cliente.EnviarLote(ctx, assinado, 2)
	if err != nil {
		t.Fatalf("%s: envio: %v", nome, err)
	}
	t.Logf("%s: envio %d — %s · protocolo %s", nome, env.CodigoResposta, env.DescricaoResposta, env.ProtocoloEnvio)
	if env.ProtocoloEnvio == "" {
		for _, o := range env.Ocorrencias {
			t.Logf("  ocorrência %d em %s: %s", o.Codigo, o.Localizacao, o.Descricao)
		}
		return ""
	}
	for tentativa := 0; tentativa < 6; tentativa++ {
		time.Sleep(30 * time.Second)
		c, err := cliente.ConsultarLote(ctx, env.ProtocoloEnvio, 2)
		if err != nil {
			t.Fatalf("%s: consulta: %v", nome, err)
		}
		if c.CodigoResposta == 101 {
			continue
		}
		for _, r := range c.Eventos {
			t.Logf("%s: evento %d — %s · recibo %s", nome, r.CodigoResposta, r.DescricaoResposta, r.NumeroRecibo)
			for _, o := range r.Ocorrencias {
				t.Logf("  ocorrência %d (tipo %d) em %s: %s", o.Codigo, o.Tipo, o.Localizacao, o.Descricao)
			}
			if r.CodigoResposta == 201 || r.CodigoResposta == 202 {
				return r.NumeroRecibo
			}
		}
		return ""
	}
	t.Logf("%s: lote ainda em processamento (protocolo %s)", nome, env.ProtocoloEnvio)
	return ""
}

func TestRestritaCadastraTrabalhadorEEnviaASO(t *testing.T) {
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
	cpf, matricula := cpfValidoAleatorio(), fmt.Sprintf("T%d", time.Now().Unix())
	admissao := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	t.Logf("trabalhador fictício: CPF %s · matrícula %s", cpf, matricula)

	s2200 := web.GerarXMLS2200(web.ParametrosS2200{Ambiente: 2, CNPJ: cert.CNPJ(), CPF: cpf, Nome: "Trabalhador de Teste",
		Sexo: "M", RacaCor: "3", GrauInstr: "07", Nascimento: "1990-05-10", Logradouro: "Rua de Teste", Numero: "100",
		Bairro: "Centro", CEP: "80010000", CodMunic: "4106902", UF: "PR", Matricula: matricula, DataAdmissao: admissao,
		Cargo: "Pedreiro", CBO: "715210", Salario: "2800.00", HorasSemanais: "44"})
	if recibo := enviarEConsultar(t, cliente, cert, "S-2200", []byte(s2200)); recibo == "" {
		t.Fatal("S-2200 sem recibo")
	}

	s2220 := web.GerarXMLS2220(web.ParametrosS2220{Ambiente: 2, CNPJ: cert.CNPJ(), CPFTrabalhador: cpf, Matricula: matricula,
		TipoExame: "0", DataASO: admissao, ResultadoASO: "1", ProcRealizado: "0295", OrdExame: "1", IndResult: "1",
		NomeMedico: "Medico de Teste", CRMMedico: "12345", UFMedico: "PR",
		NomeCoord: "Coordenador de Teste", CPFCoord: "11144477735", CRMCoord: "54321", UFCoord: "PR"})
	if recibo := enviarEConsultar(t, cliente, cert, "S-2220", []byte(s2220)); recibo == "" {
		t.Fatal("S-2220 sem recibo")
	}
}

// TestRestritaCadastraEstabelecimento envia o S-1005 da matriz (uma vez por empregador na produção
// restrita). CNAE por variável ESOCIAL_CNAE (padrão 7112000); ESOCIAL_RAT só se a alíquota for diferente da legal.
func TestRestritaCadastraEstabelecimento(t *testing.T) {
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
	cnae, rat := os.Getenv("ESOCIAL_CNAE"), os.Getenv("ESOCIAL_RAT")
	if cnae == "" {
		cnae = "7112000"
	}
	x := web.GerarXMLS1005(web.ParametrosS1005{Ambiente: 2, CNPJ: cert.CNPJ(), IniValid: time.Now().Format("2006-01"), CNAE: cnae, AliqRat: rat})
	if recibo := enviarEConsultar(t, cliente, cert, "S-1005", []byte(x)); recibo == "" {
		t.Fatal("S-1005 sem recibo")
	}
}
