package web

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/forg3/esocial-emissor-livre/internal/canteiro"
	"github.com/forg3/esocial-emissor-livre/internal/storage"
)

// Eventos vindos do Canteiro Prime entram na fila local com este prefixo no ID. O fluxo de
// validar, assinar e transmitir é o da fila; aqui só se busca e se devolve o resultado.
// O XML é apagado da fila assim que o Canteiro recebe recibo ou rejeição (dado de saúde).
const prefixoCanteiro = "cp-"

type DadosViewCanteiro struct {
	Titulo        string
	MenuAtivo     string
	Config        *storage.Configuracao
	URL           string
	TemToken      bool
	Eventos       []storage.Evento
	MensagemFlash string
	FlashErro     bool
}

func (s *Servidor) cliente() *canteiro.Cliente {
	c := canteiro.CarregarConfig(s.dirDados)
	return &canteiro.Cliente{URL: c.URL, Token: c.Token}
}

func (s *Servidor) handleCanteiro(w http.ResponseWriter, r *http.Request) {
	s.canteiroComFlash(w, r, "", false)
}

func (s *Servidor) canteiroComFlash(w http.ResponseWriter, r *http.Request, msg string, erro bool) {
	cfg, _ := s.db.ObterConfiguracao()
	c := canteiro.CarregarConfig(s.dirDados)
	todos, _ := s.db.ListarEventos()
	var meus []storage.Evento
	for _, e := range todos {
		if strings.HasPrefix(e.ID, prefixoCanteiro) {
			e.XMLGerado, e.XMLAssinado = "", "" // a tela não precisa do XML
			meus = append(meus, e)
		}
	}
	s.render(w, r, "canteiro", DadosViewCanteiro{
		Titulo: "Canteiro Prime", MenuAtivo: "canteiro", Config: cfg,
		URL: c.URL, TemToken: c.Token != "", Eventos: meus, MensagemFlash: msg, FlashErro: erro,
	})
}

func (s *Servidor) handleSalvarConfigCanteiro(w http.ResponseWriter, r *http.Request) {
	atual := canteiro.CarregarConfig(s.dirDados)
	c := canteiro.Config{URL: strings.TrimSpace(r.FormValue("url")), Token: strings.TrimSpace(r.FormValue("token"))}
	if c.Token == "" { // campo em branco mantém o token já salvo
		c.Token = atual.Token
	}
	if err := c.Validar(); err != nil {
		s.canteiroComFlash(w, r, err.Error(), true)
		return
	}
	if err := canteiro.SalvarConfig(s.dirDados, c); err != nil {
		s.canteiroComFlash(w, r, "Falha ao salvar a configuração: "+err.Error(), true)
		return
	}
	s.canteiroComFlash(w, r, "Configuração salva neste computador.", false)
}

func (s *Servidor) handleBuscarCanteiro(w http.ResponseWriter, r *http.Request) {
	cfg, _ := s.db.ObterConfiguracao()
	ctx, cancelar := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancelar()
	pend, err := s.cliente().Pendentes(ctx)
	if err != nil {
		s.canteiroComFlash(w, r, err.Error(), true)
		return
	}
	novos, jaTinha, outroAmb := 0, 0, 0
	for _, p := range pend {
		if p.ID == "" || p.Tipo == "" || strings.TrimSpace(p.XML) == "" {
			continue
		}
		id := prefixoCanteiro + p.ID
		if ex, _ := s.db.ObterEvento(id); ex != nil {
			jaTinha++
			continue
		}
		if p.AmbienteNum() != cfg.Ambiente {
			outroAmb++
			continue
		}
		if err := s.db.SalvarEvento(&storage.Evento{ID: id, Tipo: p.Tipo, Ambiente: p.AmbienteNum(), Status: "pronto", XMLGerado: p.XML}); err != nil {
			s.canteiroComFlash(w, r, "Falha ao guardar o evento na sessão: "+err.Error(), true)
			return
		}
		novos++
	}
	msg := fmt.Sprintf("%d evento(s) novo(s) trazido(s) do Canteiro Prime; %d já estavam na lista.", novos, jaTinha)
	if outroAmb > 0 {
		msg += fmt.Sprintf(" %d ignorado(s): ambiente diferente do configurado aqui (troque em Certificado & Empresa).", outroAmb)
	}
	s.canteiroComFlash(w, r, msg, outroAmb > 0)
}

// handleResultadoCanteiro devolve ao Canteiro o que a fila local registrou do evento.
func (s *Servidor) handleResultadoCanteiro(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ev, err := s.db.ObterEvento(id)
	if err != nil || ev == nil || !strings.HasPrefix(id, prefixoCanteiro) {
		s.canteiroComFlash(w, r, "Evento não localizado.", true)
		return
	}
	res := canteiro.Resultado{Status: ev.Status, Recibo: ev.Recibo, Protocolo: ev.Protocolo, Mensagem: ev.MensagemRetorno}
	switch ev.Status {
	case "aceito", "rejeitado", "transmitido":
	default:
		s.canteiroComFlash(w, r, "Este evento ainda não foi transmitido ao eSocial (situação: "+ev.Status+"). Valide, assine e transmita antes de enviar o resultado.", true)
		return
	}
	ctx, cancelar := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancelar()
	if err := s.cliente().EnviarResultado(ctx, strings.TrimPrefix(id, prefixoCanteiro), res); err != nil {
		s.canteiroComFlash(w, r, "Resultado NÃO enviado (o evento continua aqui, tente de novo): "+err.Error(), true)
		return
	}
	if ev.Status == "transmitido" { // ainda falta o recibo: mantém para consultar depois
		s.canteiroComFlash(w, r, "Canteiro avisado de que o evento foi transmitido. Consulte o recibo e envie de novo.", false)
		return
	}
	_ = s.db.ExcluirEvento(id) // resultado final entregue: o XML sai desta máquina
	s.canteiroComFlash(w, r, "Resultado enviado ao Canteiro Prime ("+ev.Status+"); o XML foi removido da fila local.", false)
}
