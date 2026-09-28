package web

import (
	"fmt"
	"strings"
)

// ParametrosS1005 cadastra um estabelecimento (inclusive a matriz) na tabela do empregador. O S-2200 só é
// aceito com o local de trabalho cadastrado aqui (ocorrência 272 na produção restrita, 28/09/2026).
type ParametrosS1005 struct {
	ID       string
	Ambiente int
	CNPJ     string // empregador; ideEmpregador usa a raiz
	CNPJEst  string // estabelecimento (14 dígitos; padrão = CNPJ do empregador)
	IniValid string // AAAA-MM
	CNAE     string // CNAE preponderante (7 dígitos)
	AliqRat  string // só se diferente da alíquota legal do CNAE (exige processo); vazio no caso normal
}

// GerarXMLS1005 produz o evtTabEstab (inclusão, S-1.3, aderente a internal/data/xsd/evtTabEstab.xsd).
func GerarXMLS1005(p ParametrosS1005) string {
	if p.ID == "" {
		p.ID = GerarIDEvento(p.CNPJ)
	}
	cnpj := limpaDigitos(p.CNPJ)
	est := limpaDigitos(p.CNPJEst)
	if len(est) != 14 {
		est = cnpj
	}
	var sb strings.Builder
	w := func(format string, args ...any) { fmt.Fprintf(&sb, format+"\n", args...) }
	w(`<?xml version="1.0" encoding="UTF-8"?>`)
	w(`<eSocial xmlns="http://www.esocial.gov.br/schema/evt/evtTabEstab/v_S_01_03_00">`)
	w(`  <evtTabEstab Id="%s">`, escapeXML(p.ID))
	w("    <ideEvento>")
	w("      <tpAmb>%d</tpAmb>", p.Ambiente)
	w("      <procEmi>1</procEmi>")
	w("      <verProc>1.1.0</verProc>")
	w("    </ideEvento>")
	w("    <ideEmpregador>")
	w("      <tpInsc>1</tpInsc>")
	w("      <nrInsc>%s</nrInsc>", raizCNPJ(cnpj))
	w("    </ideEmpregador>")
	w("    <infoEstab>")
	w("      <inclusao>")
	w("        <ideEstab>")
	w("          <tpInsc>1</tpInsc>")
	w("          <nrInsc>%s</nrInsc>", est)
	w("          <iniValid>%s</iniValid>", escapeXML(strings.TrimSpace(p.IniValid)))
	w("        </ideEstab>")
	w("        <dadosEstab>")
	w("          <cnaePrep>%s</cnaePrep>", somenteDigitosOu(limpaDigitos(p.CNAE), "", 7))
	// aliqGilrat só vai quando a alíquota é DIFERENTE da legal para o CNAE (e aí com processo); no caso
	// normal o eSocial usa a da lei e recusa a informada igual (ocorrência 1504, produção restrita).
	if rat := validarDominioOuVazio(p.AliqRat, "1", "2", "3"); rat != "" {
		w("          <aliqGilrat>")
		w("            <aliqRat>%s</aliqRat>", rat)
		w("          </aliqGilrat>")
	}
	w("        </dadosEstab>")
	w("      </inclusao>")
	w("    </infoEstab>")
	w("  </evtTabEstab>")
	sb.WriteString("</eSocial>")
	return sb.String()
}
