// Package canteiro é o cliente do agente local para o Canteiro Prime: busca os eventos
// pendentes do assinante e devolve o recibo ou o erro. Não assina nem transmite (isso é
// do fluxo que já existe) e não guarda nada: o XML só passa por aqui.
package canteiro

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const arquivoConfig = "canteiro.json"

// Config guarda o endereço do Canteiro e o token do agente (lxag_<tenant>.<segredo>).
type Config struct {
	URL   string `json:"url"`
	Token string `json:"token"`
}

// CarregarConfig lê dados/canteiro.json; as variáveis CANTEIRO_URL e CANTEIRO_TOKEN têm precedência.
func CarregarConfig(dir string) Config {
	var c Config
	if b, err := os.ReadFile(filepath.Join(dir, arquivoConfig)); err == nil {
		_ = json.Unmarshal(b, &c)
	}
	if v := os.Getenv("CANTEIRO_URL"); v != "" {
		c.URL = v
	}
	if v := os.Getenv("CANTEIRO_TOKEN"); v != "" {
		c.Token = v
	}
	c.URL = strings.TrimRight(strings.TrimSpace(c.URL), "/")
	c.Token = strings.TrimSpace(c.Token)
	return c
}

// SalvarConfig grava o arquivo com permissão 0600 (o token é uma credencial).
func SalvarConfig(dir string, c Config) error {
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, arquivoConfig), b, 0o600)
}

// Pendente é um evento que o Canteiro quer ver assinado e transmitido.
type Pendente struct {
	ID       string `json:"id"`
	Tipo     string `json:"tipo"`
	Ambiente string `json:"ambiente"` // restrita | producao
	XML      string `json:"xml"`
}

// AmbienteNum converte para o código do eSocial (1 produção, 2 restrita).
func (p Pendente) AmbienteNum() int {
	if p.Ambiente == "producao" {
		return 1
	}
	return 2
}

// Resultado é o corpo de POST /esocial/api/{id}/resultado.
type Resultado struct {
	Status    string `json:"status"` // transmitido | aceito | rejeitado
	Protocolo string `json:"protocolo,omitempty"`
	Recibo    string `json:"recibo,omitempty"`
	Mensagem  string `json:"mensagem,omitempty"`
}

type Cliente struct {
	URL, Token string
	HTTP       *http.Client // opcional
}

func (c *Cliente) base() (string, error) {
	u, err := url.Parse(c.URL)
	if c.URL == "" || c.Token == "" || err != nil || u.Host == "" {
		return "", errors.New("Canteiro Prime não configurado: informe o endereço e o token do agente")
	}
	host := u.Hostname()
	ip := net.ParseIP(host)
	local := host == "localhost" || (ip != nil && ip.IsLoopback())
	if u.Scheme != "https" && !(u.Scheme == "http" && local) {
		return "", errors.New("o endereço do Canteiro Prime precisa começar com https://")
	}
	return strings.TrimRight(c.URL, "/"), nil
}

func (c *Cliente) fazer(ctx context.Context, metodo, caminho string, corpo any, saida any) error {
	base, err := c.base()
	if err != nil {
		return err
	}
	var rd io.Reader
	if corpo != nil {
		b, _ := json.Marshal(corpo)
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, metodo, base+caminho, rd)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Accept", "application/json")
	if corpo != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	cli := c.HTTP
	if cli == nil {
		cli = &http.Client{Timeout: 30 * time.Second}
	}
	resp, err := cli.Do(req)
	if err != nil {
		// Só o texto fixo e o endereço: nunca o token.
		return fmt.Errorf("Não foi possível falar com o Canteiro Prime em %s (verifique a internet e o endereço): %w", base, semURL(err))
	}
	defer resp.Body.Close()
	dados, _ := io.ReadAll(io.LimitReader(resp.Body, 20<<20))
	switch {
	case resp.StatusCode == http.StatusUnauthorized:
		return errors.New("o Canteiro Prime recusou o token (inválido ou revogado): gere outro em /esocial/agente")
	case resp.StatusCode >= 300:
		return fmt.Errorf("o Canteiro Prime respondeu %d ao %s %s", resp.StatusCode, metodo, caminho)
	}
	if saida != nil {
		if err := json.Unmarshal(dados, saida); err != nil {
			return errors.New("resposta do Canteiro Prime fora do formato esperado")
		}
	}
	return nil
}

func semURL(err error) error {
	var ue *url.Error
	if errors.As(err, &ue) {
		return ue.Err
	}
	return err
}

// Pendentes lista os eventos prontos para assinar e transmitir.
func (c *Cliente) Pendentes(ctx context.Context) ([]Pendente, error) {
	var r struct {
		Eventos []Pendente `json:"eventos"`
	}
	if err := c.fazer(ctx, http.MethodGet, "/esocial/api/pendentes", nil, &r); err != nil {
		return nil, err
	}
	return r.Eventos, nil
}

// EnviarResultado devolve o recibo ou o erro. Repetir o mesmo resultado é seguro (o Canteiro é idempotente).
func (c *Cliente) EnviarResultado(ctx context.Context, id string, r Resultado) error {
	return c.fazer(ctx, http.MethodPost, "/esocial/api/"+url.PathEscape(id)+"/resultado", r, nil)
}
