package canteiro

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const tokenTeste = "lxag_t1.segredo"

// servidorFalso imita as duas rotas do Canteiro Prime. O resultado é idempotente por evento.
func servidorFalso(t *testing.T) (*httptest.Server, map[string]Resultado, *int) {
	t.Helper()
	recebidos := map[string]Resultado{}
	chamadas := 0
	mux := http.NewServeMux()
	auth := func(w http.ResponseWriter, r *http.Request) bool {
		if r.Header.Get("Authorization") != "Bearer "+tokenTeste {
			http.Error(w, `{"detail":"Token do agente inválido ou revogado."}`, http.StatusUnauthorized)
			return false
		}
		return true
	}
	mux.HandleFunc("GET /esocial/api/pendentes", func(w http.ResponseWriter, r *http.Request) {
		if !auth(w, r) {
			return
		}
		w.Write([]byte(`{"eventos":[{"id":"ev1","tipo":"S-2210","ambiente":"restrita","xml":"<x/>"}]}`))
	})
	mux.HandleFunc("POST /esocial/api/{id}/resultado", func(w http.ResponseWriter, r *http.Request) {
		if !auth(w, r) {
			return
		}
		chamadas++
		var res Resultado
		if json.NewDecoder(r.Body).Decode(&res) != nil {
			http.Error(w, "", http.StatusUnprocessableEntity)
			return
		}
		recebidos[r.PathValue("id")] = res // mesmo id: sobrescreve, não duplica
		w.Write([]byte(`{"id":"` + r.PathValue("id") + `","status":"` + res.Status + `"}`))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, recebidos, &chamadas
}

func TestPendentes200(t *testing.T) {
	srv, _, _ := servidorFalso(t)
	evs, err := (&Cliente{URL: srv.URL, Token: tokenTeste}).Pendentes(context.Background())
	if err != nil || len(evs) != 1 || evs[0].ID != "ev1" || evs[0].XML != "<x/>" || evs[0].AmbienteNum() != 2 {
		t.Fatalf("lista inesperada: %+v err=%v", evs, err)
	}
}

func TestPendentes401(t *testing.T) {
	srv, _, _ := servidorFalso(t)
	_, err := (&Cliente{URL: srv.URL, Token: "lxag_t1.errado"}).Pendentes(context.Background())
	if err == nil || !strings.Contains(err.Error(), "token") {
		t.Fatalf("esperava erro de token, veio %v", err)
	}
	if strings.Contains(err.Error(), "errado") {
		t.Fatal("o token não pode aparecer na mensagem de erro")
	}
}

func TestResultadoEnviadoEIdempotente(t *testing.T) {
	srv, recebidos, chamadas := servidorFalso(t)
	c := &Cliente{URL: srv.URL, Token: tokenTeste}
	res := Resultado{Status: "aceito", Recibo: "1.2.0000000000000000001", Mensagem: "ok"}
	for i := 0; i < 2; i++ {
		if err := c.EnviarResultado(context.Background(), "ev1", res); err != nil {
			t.Fatal(err)
		}
	}
	if *chamadas != 2 || len(recebidos) != 1 || recebidos["ev1"] != res {
		t.Fatalf("reenvio deveria manter um único resultado: %+v", recebidos)
	}
}

func TestErroDeRedeClaro(t *testing.T) {
	srv, _, _ := servidorFalso(t)
	url := srv.URL
	srv.Close()
	_, err := (&Cliente{URL: url, Token: tokenTeste}).Pendentes(context.Background())
	if err == nil || !strings.Contains(err.Error(), "Não foi possível falar com o Canteiro Prime") {
		t.Fatalf("mensagem pouco clara: %v", err)
	}
	if strings.Contains(err.Error(), tokenTeste) {
		t.Fatal("token vazou no erro")
	}
}

func TestURLInseguraRecusada(t *testing.T) {
	if _, err := (&Cliente{URL: "http://exemplo.com.br", Token: tokenTeste}).Pendentes(context.Background()); err == nil {
		t.Fatal("http fora de localhost deveria ser recusado")
	}
}

func TestConfigArquivoEAmbiente(t *testing.T) {
	dir := t.TempDir()
	if err := SalvarConfig(dir, Config{URL: "https://canteiro.exemplo/", Token: tokenTeste}); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(filepath.Join(dir, "canteiro.json"))
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("permissão %v, esperado 0600", info.Mode().Perm())
	}
	c := CarregarConfig(dir)
	if c.URL != "https://canteiro.exemplo" || c.Token != tokenTeste {
		t.Fatalf("config: %+v", c)
	}
	t.Setenv("CANTEIRO_URL", "https://outro.exemplo")
	t.Setenv("CANTEIRO_TOKEN", "lxag_t2.x")
	if c := CarregarConfig(dir); c.URL != "https://outro.exemplo" || c.Token != "lxag_t2.x" {
		t.Fatalf("env deveria prevalecer: %+v", c)
	}
}

func TestNaoSegueRedirecionamento(t *testing.T) {
	alvo := 0
	outro := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { alvo++ }))
	defer outro.Close()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, outro.URL, http.StatusFound)
	}))
	defer srv.Close()
	if _, err := (&Cliente{URL: srv.URL, Token: tokenTeste}).Pendentes(context.Background()); err == nil || alvo != 0 {
		t.Fatalf("redirecionamento seguido (alvo=%d, err=%v)", alvo, err)
	}
}
