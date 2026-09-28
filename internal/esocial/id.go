package esocial

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"
)

var (
	// regexIDEvento valida o formato canônico de 36 caracteres do eSocial.
	// Padrão do eSocial: "ID" + tipo (1 dígito) + 14 posições inscrição + 14 dígitos timestamp + 5 dígitos sequencial.
	regexIDEvento = regexp.MustCompile(`^ID[1-2][0-9]{14}[0-9]{14}[0-9]{5}$`)
)

// LimparDocumento remove caracteres não numéricos de CPF/CNPJ.
func LimparDocumento(doc string) string {
	var sb strings.Builder
	for _, r := range doc {
		if unicode.IsDigit(r) {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

// InscricaoDoID devolve as 14 posições do número de inscrição no Id do evento, pela regra do eSocial:
// completar com zeros À DIREITA. CNPJ entra pela raiz (8 dígitos, a mesma do ideEmpregador/nrInsc
// do S-1000 e do lote) + "000000"; CPF entra com os 11 dígitos + "000". Zeros à esquerda fazem o
// eSocial recusar o lote com a ocorrência 609 "Código inválido" em evento/@Id (produção restrita,
// 28/09/2026).
func InscricaoDoID(tpInsc int, nrInsc string) string {
	doc := LimparDocumento(nrInsc)
	if tpInsc != TpInscCPF && len(doc) > 8 {
		doc = doc[:8] // CNPJ: só a raiz
	}
	if len(doc) > 14 {
		doc = doc[:14]
	}
	return doc + strings.Repeat("0", 14-len(doc))
}

// GerarIDEvento gera o identificador único obrigatório do evento eSocial (36 caracteres)
// utilizando o timestamp atual UTC/Local:
// Formato: ID + tpInsc (1 dígito) + nrInsc (14 dígitos) + YYYYMMDDHHMMSS (14 dígitos) + sequencial (5 dígitos)
// Exemplo: ID1000000000000012026092212300000001
func GerarIDEvento(tpInsc int, nrInsc string, sequencial int) string {
	return GerarIDEventoComTempo(tpInsc, nrInsc, time.Now(), sequencial)
}

// GerarIDEventoComTempo permite gerar o identificador determinístico com uma data/hora explícita,
// ideal para testes e idempotência.
func GerarIDEventoComTempo(tpInsc int, nrInsc string, t time.Time, sequencial int) string {
	if tpInsc < 1 || tpInsc > 2 {
		tpInsc = 1 // Padrão CNPJ caso inválido
	}

	docFormatado := InscricaoDoID(tpInsc, nrInsc)

	timestamp := t.Format("20060102150405")

	// Sequencial mod 100000 para caber estritamente em 5 dígitos (00001 a 99999)
	seqNormalizado := sequencial
	if seqNormalizado <= 0 {
		seqNormalizado = 1
	} else if seqNormalizado > 99999 {
		seqNormalizado = seqNormalizado % 100000
		if seqNormalizado == 0 {
			seqNormalizado = 1
		}
	}

	return fmt.Sprintf("ID%d%s%s%05d", tpInsc, docFormatado, timestamp, seqNormalizado)
}

// ValidarIDEvento verifica se o identificador segue estritamente as regras de formação do eSocial S-1.3.
func ValidarIDEvento(id string) error {
	if len(id) != 36 {
		return fmt.Errorf("id do evento deve ter exatamente 36 caracteres, possui %d", len(id))
	}
	if !strings.HasPrefix(id, "ID") {
		return errors.New("id do evento deve iniciar com o prefixo 'ID'")
	}
	if !regexIDEvento.MatchString(id) {
		return errors.New("formato de id do evento inválido para o padrão eSocial (deve conter apenas dígitos após 'ID')")
	}
	return nil
}
