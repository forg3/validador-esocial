package web_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

type canteiroFalso struct {
	*httptest.Server
	resultados map[string]map[string]string
	posts      int
}

func novoCanteiroFalso(t *testing.T, token string) *canteiroFalso {
	f := &canteiroFalso{resultados: map[string]map[string]string{}}
	mux := http.NewServeMux()
	ok := func(r *http.Request) bool { return r.Header.Get("Authorization") == "Bearer "+token }
	mux.HandleFunc("GET /esocial/api/pendentes", func(w http.ResponseWriter, r *http.Request) {
		if !ok(r) {
			http.Error(w, "{}", 401)
			return
		}
		w.Write([]byte(`{"eventos":[{"id":"e1","tipo":"S-2210","ambiente":"restrita","xml":"<evtCAT/>"}]}`))
	})
	mux.HandleFunc("POST /esocial/api/{id}/resultado", func(w http.ResponseWriter, r *http.Request) {
		if !ok(r) {
			http.Error(w, "{}", 401)
			return
		}
		f.posts++
		var m map[string]string
		json.NewDecoder(r.Body).Decode(&m)
		f.resultados[r.PathValue("id")] = m
		w.Write([]byte(`{}`))
	})
	f.Server = httptest.NewServer(mux)
	t.Cleanup(f.Close)
	return f
}

func postCanteiro(t *testing.T, mux http.Handler, cookie *http.Cookie, csrf, caminho string) string {
	t.Helper()
	req := httptest.NewRequest("POST", caminho, strings.NewReader(url.Values{"csrf_token": {csrf}}.Encode()))
	req.Host = "localhost:8000"
	req.AddCookie(cookie)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-CSRF-Token", csrf)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("%s: %d", caminho, rec.Code)
	}
	return rec.Body.String()
}

func TestCanteiroBuscarEnviarResultadoIdempotente(t *testing.T) {
	srv, db, mux, cleanup := prepararServidorSeguranca(t)
	defer cleanup()
	cfg := novoCanteiroFalso(t, "lxag_t.ok")
	t.Setenv("CANTEIRO_URL", cfg.URL)
	t.Setenv("CANTEIRO_TOKEN", "lxag_t.ok")
	cookie, csrf := autenticar(t, mux, srv), srv.TokenCSRF()

	if corpo := postCanteiro(t, mux, cookie, csrf, "/canteiro/buscar"); !strings.Contains(corpo, "1 evento(s) novo(s)") {
		t.Fatalf("lista não trouxe o evento: %s", corpo)
	}
	// buscar de novo não duplica
	if corpo := postCanteiro(t, mux, cookie, csrf, "/canteiro/buscar"); !strings.Contains(corpo, "0 evento(s) novo(s)") {
		t.Fatalf("segunda busca duplicou: %s", corpo)
	}
	// ainda não transmitido: não envia
	if corpo := postCanteiro(t, mux, cookie, csrf, "/canteiro/cp-e1/resultado"); !strings.Contains(corpo, "ainda não foi transmitido") || cfg.posts != 0 {
		t.Fatalf("enviou antes da hora: %s", corpo)
	}
	// o fluxo existente transmite e registra o recibo
	if err := db.AtualizarStatusEvento("cp-e1", "aceito", "1.2.345", "PROT1", "ok"); err != nil {
		t.Fatal(err)
	}
	if corpo := postCanteiro(t, mux, cookie, csrf, "/canteiro/cp-e1/resultado"); !strings.Contains(corpo, "Resultado enviado") {
		t.Fatalf("resultado não enviado: %s", corpo)
	}
	r := cfg.resultados["e1"]
	if r["status"] != "aceito" || r["recibo"] != "1.2.345" || r["protocolo"] != "PROT1" {
		t.Fatalf("corpo enviado: %v", r)
	}
	if ev, _ := db.ObterEvento("cp-e1"); ev != nil {
		t.Fatal("o XML deveria sair da fila local depois do resultado final")
	}
	// reenvio: nada a duplicar
	postCanteiro(t, mux, cookie, csrf, "/canteiro/cp-e1/resultado")
	if cfg.posts != 1 {
		t.Fatalf("reenvio duplicou: %d POSTs", cfg.posts)
	}
}

func TestCanteiroTokenRecusadoERedeCaida(t *testing.T) {
	srv, _, mux, cleanup := prepararServidorSeguranca(t)
	defer cleanup()
	cfg := novoCanteiroFalso(t, "lxag_t.ok")
	t.Setenv("CANTEIRO_URL", cfg.URL)
	t.Setenv("CANTEIRO_TOKEN", "lxag_t.errado")
	cookie, csrf := autenticar(t, mux, srv), srv.TokenCSRF()

	corpo := postCanteiro(t, mux, cookie, csrf, "/canteiro/buscar")
	if !strings.Contains(corpo, "recusou o token") || strings.Contains(corpo, "lxag_t.errado") {
		t.Fatalf("mensagem de 401 inadequada: %s", corpo)
	}
	cfg.Close()
	corpo = postCanteiro(t, mux, cookie, csrf, "/canteiro/buscar")
	if !strings.Contains(corpo, "Não foi possível falar com o Canteiro Prime") {
		t.Fatalf("mensagem de rede inadequada: %s", corpo)
	}
}
