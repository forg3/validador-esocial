package web

import (
	"fmt"
	"strings"
)

// ParametrosS2200 reúne o mínimo que o eSocial aceita num S-2200 (admissão de empregado celetista,
// leiaute S-1.3): trabalhador, endereço no Brasil, vínculo, contrato, remuneração, local e jornada.
// É o evento que cadastra o trabalhador — sem ele, S-2210, S-2220 e S-2240 voltam com a ocorrência
// 1557 "Não foi localizado o contrato de trabalho" (produção restrita, 28/09/2026).
type ParametrosS2200 struct {
	ID         string
	Ambiente   int
	CNPJ       string // empregador (14 dígitos); ideEmpregador usa a raiz
	CPF        string
	Nome       string
	Sexo       string // M | F
	RacaCor    string // 1 branca, 2 preta, 3 parda, 4 amarela, 5 indígena (obrigatório; 6 é recusado)
	GrauInstr  string // Tabela: 01..12 (ex.: 07 médio completo)
	Nascimento string // AAAA-MM-DD

	Logradouro string
	Numero     string
	Bairro     string
	CEP        string
	CodMunic   string // código IBGE do município (7 dígitos)
	UF         string

	Matricula        string
	DataAdmissao     string // AAAA-MM-DD
	CNPJSindicato    string // CNPJ do sindicato da categoria (14 dígitos)
	Cargo            string
	CBO              string // 6 dígitos
	CodCateg         string // Tabela 01 (101 = empregado geral)
	Salario          string // ex.: "2500.00"
	CNPJLocal        string // estabelecimento onde trabalha (14 dígitos; padrão = CNPJ)
	HorasSemanais    string // ex.: "44"
	DescricaoJornada string
}

// GerarXMLS2200 produz o evtAdmissao (S-1.3, aderente a internal/data/xsd/evtAdmissao.xsd).
func GerarXMLS2200(p ParametrosS2200) string {
	if p.ID == "" {
		p.ID = GerarIDEvento(p.CNPJ)
	}
	cnpj := limpaDigitos(p.CNPJ)
	local := limpaDigitos(p.CNPJLocal)
	if len(local) != 14 {
		local = cnpj
	}
	salario := strings.ReplaceAll(strings.TrimSpace(p.Salario), ",", ".")
	if salario == "" {
		salario = "0.00"
	}
	horas := strings.TrimSpace(p.HorasSemanais)
	if horas == "" {
		horas = "44"
	}
	jornada := strings.TrimSpace(p.DescricaoJornada)
	if jornada == "" {
		jornada = "Segunda a sexta, 8h às 17h48, com 1h de intervalo."
	}
	cargo := strings.TrimSpace(p.Cargo)
	if cargo == "" {
		cargo = "Empregado"
	}

	var sb strings.Builder
	w := func(format string, args ...any) { fmt.Fprintf(&sb, format+"\n", args...) }
	w(`<?xml version="1.0" encoding="UTF-8"?>`)
	w(`<eSocial xmlns="http://www.esocial.gov.br/schema/evt/evtAdmissao/v_S_01_03_00">`)
	w(`  <evtAdmissao Id="%s">`, escapeXML(p.ID))
	w("    <ideEvento>")
	w("      <indRetif>1</indRetif>")
	w("      <tpAmb>%d</tpAmb>", p.Ambiente)
	w("      <procEmi>1</procEmi>")
	w("      <verProc>1.1.0</verProc>")
	w("    </ideEvento>")
	w("    <ideEmpregador>")
	w("      <tpInsc>1</tpInsc>")
	w("      <nrInsc>%s</nrInsc>", raizCNPJ(cnpj))
	w("    </ideEmpregador>")
	w("    <trabalhador>")
	w("      <cpfTrab>%s</cpfTrab>", limpaDigitos(p.CPF))
	w("      <nmTrab>%s</nmTrab>", escapeXML(strings.TrimSpace(p.Nome)))
	w("      <sexo>%s</sexo>", validarDominio(strings.ToUpper(p.Sexo), "M", "F"))
	// Lei 14.553/2023: raça/cor é obrigatória e "6 - não informado" é recusado (ocorrência 1871) — sem
	// valor válido o campo sai vazio e o XSD aponta a falta, em vez de o sistema inventar uma resposta.
	w("      <racaCor>%s</racaCor>", validarDominioOuVazio(p.RacaCor, "1", "2", "3", "4", "5"))
	w("      <grauInstr>%s</grauInstr>", somenteDigitosOu(p.GrauInstr, "07", 2))
	w("      <nascimento>")
	w("        <dtNascto>%s</dtNascto>", validarDataISO(p.Nascimento))
	w("        <paisNascto>105</paisNascto>")
	w("        <paisNac>105</paisNac>")
	w("      </nascimento>")
	w("      <endereco>")
	w("        <brasil>")
	w("          <dscLograd>%s</dscLograd>", escapeXML(strings.TrimSpace(p.Logradouro)))
	w("          <nrLograd>%s</nrLograd>", escapeXML(valorOu(p.Numero, "S/N")))
	if b := strings.TrimSpace(p.Bairro); b != "" {
		w("          <bairro>%s</bairro>", escapeXML(b))
	}
	w("          <cep>%s</cep>", somenteDigitosOu(limpaDigitos(p.CEP), "80000000", 8))
	w("          <codMunic>%s</codMunic>", somenteDigitosOu(limpaDigitos(p.CodMunic), "4106902", 7))
	w("          <uf>%s</uf>", escapeXML(strings.ToUpper(valorOu(p.UF, "PR"))))
	w("        </brasil>")
	w("      </endereco>")
	w("    </trabalhador>")
	w("    <vinculo>")
	w("      <matricula>%s</matricula>", escapeXML(strings.TrimSpace(p.Matricula)))
	w("      <tpRegTrab>1</tpRegTrab>")
	w("      <tpRegPrev>1</tpRegPrev>")
	w("      <cadIni>N</cadIni>")
	w("      <infoRegimeTrab>")
	w("        <infoCeletista>")
	w("          <dtAdm>%s</dtAdm>", validarDataISO(p.DataAdmissao))
	w("          <tpAdmissao>1</tpAdmissao>")
	w("          <indAdmissao>1</indAdmissao>")
	w("          <tpRegJor>1</tpRegJor>")
	w("          <natAtividade>1</natAtividade>")
	w("          <cnpjSindCategProf>%s</cnpjSindCategProf>", somenteDigitosOu(limpaDigitos(p.CNPJSindicato), cnpj, 14))
	w("        </infoCeletista>")
	w("      </infoRegimeTrab>")
	w("      <infoContrato>")
	w("        <nmCargo>%s</nmCargo>", escapeXML(cargo))
	w("        <CBOCargo>%s</CBOCargo>", somenteDigitosOu(limpaDigitos(p.CBO), "717020", 6))
	w("        <codCateg>%s</codCateg>", somenteDigitosOu(p.CodCateg, "101", 3))
	w("        <remuneracao>")
	w("          <vrSalFx>%s</vrSalFx>", escapeXML(salario))
	w("          <undSalFixo>5</undSalFixo>")
	w("        </remuneracao>")
	w("        <duracao>")
	w("          <tpContr>1</tpContr>")
	w("        </duracao>")
	w("        <localTrabalho>")
	w("          <localTrabGeral>")
	w("            <tpInsc>1</tpInsc>")
	w("            <nrInsc>%s</nrInsc>", local)
	w("          </localTrabGeral>")
	w("        </localTrabalho>")
	w("        <horContratual>")
	w("          <qtdHrsSem>%s</qtdHrsSem>", escapeXML(horas))
	w("          <tpJornada>2</tpJornada>")
	w("          <tmpParc>0</tmpParc>")
	w("          <horNoturno>N</horNoturno>")
	w("          <dscJorn>%s</dscJorn>", escapeXML(jornada))
	w("        </horContratual>")
	w("      </infoContrato>")
	w("    </vinculo>")
	w("  </evtAdmissao>")
	sb.WriteString("</eSocial>")
	return sb.String()
}

func valorOu(v, padrao string) string {
	if v = strings.TrimSpace(v); v != "" {
		return v
	}
	return padrao
}

func validarDominioOuVazio(v string, permitidos ...string) string {
	v = strings.TrimSpace(v)
	for _, p := range permitidos {
		if v == p {
			return v
		}
	}
	return ""
}
