package web

import (
	"testing"

	"github.com/forg3/esocial-emissor-livre/internal/esocial"
)

func TestGerarXMLS2200PassaNoXSDOficial(t *testing.T) {
	if !esocial.XmllintDisponivel() {
		t.Skip("xmllint ausente")
	}
	x := GerarXMLS2200(ParametrosS2200{Ambiente: 2, CNPJ: "36380294000171", CPF: "52998224725", Nome: "Trabalhador de Teste",
		Sexo: "M", RacaCor: "3", GrauInstr: "07", Nascimento: "1990-05-10", Logradouro: "Rua das Palmeiras", Numero: "100",
		Bairro: "Centro", CEP: "80010-000", CodMunic: "4106902", UF: "PR", Matricula: "TESTE-0001", DataAdmissao: "2026-09-01",
		Cargo: "Pedreiro", CBO: "715210", Salario: "2800,00", HorasSemanais: "44"})
	if err := esocial.ValidarXSD([]byte(x), "S-2200"); err != nil {
		t.Fatalf("S-2200 fora do XSD oficial: %v\n%s", err, x)
	}
}

func TestGerarXMLS2200SemRacaCorNaoPassaNoXSD(t *testing.T) {
	if !esocial.XmllintDisponivel() {
		t.Skip("xmllint ausente")
	}
	x := GerarXMLS2200(ParametrosS2200{Ambiente: 2, CNPJ: "36380294000171", CPF: "52998224725", Nome: "X", Sexo: "M",
		RacaCor: "6", Nascimento: "1990-05-10", Logradouro: "Rua", Matricula: "M1", DataAdmissao: "2026-09-01"})
	if esocial.ValidarXSD([]byte(x), "S-2200") == nil {
		t.Fatal("raça/cor 6 (proibida desde 2023) deveria ficar vazia e reprovar no XSD")
	}
}

func TestGerarXMLS1005PassaNoXSDOficial(t *testing.T) {
	if !esocial.XmllintDisponivel() {
		t.Skip("xmllint ausente")
	}
	x := GerarXMLS1005(ParametrosS1005{Ambiente: 2, CNPJ: "36380294000171", IniValid: "2026-09", CNAE: "7112000"})
	if err := esocial.ValidarXSD([]byte(x), "S-1005"); err != nil {
		t.Fatalf("S-1005 fora do XSD oficial: %v\n%s", err, x)
	}
}
