package web_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// Com Referrer-Policy no-referrer o navegador envia "Origin: null" em POST do proprio site;
// a origem e comprovada por Sec-Fetch-Site.
func TestCSRFOrigemNulaESecFetchSite(t *testing.T) {
	srv, _, mux, cleanup := prepararServidorSeguranca(t)
	defer cleanup()

	casos := []struct {
		nome, origin, sfs string
		quer              int
	}{
		{"null + same-origin", "null", "same-origin", http.StatusSeeOther},
		{"null + none", "null", "none", http.StatusSeeOther},
		{"null + cross-site", "null", "cross-site", http.StatusForbidden},
		{"null + same-site", "null", "same-site", http.StatusForbidden},
		{"null sem Sec-Fetch-Site", "null", "", http.StatusForbidden},
		{"outro host", "https://evil.example", "", http.StatusForbidden},
		{"outro host + same-origin", "https://evil.example", "same-origin", http.StatusForbidden},
		{"mesmo host", "http://localhost:8000", "same-origin", http.StatusSeeOther},
	}
	for _, c := range casos {
		form := url.Values{"senha": {senhaTeste}, "csrf_token": {srv.TokenCSRF()}}
		req := httptest.NewRequest("POST", "/login", strings.NewReader(form.Encode()))
		req.Host = "localhost:8000"
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Origin", c.origin)
		if c.sfs != "" {
			req.Header.Set("Sec-Fetch-Site", c.sfs)
		}
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != c.quer {
			t.Errorf("%s: esperado %d, obtido %d", c.nome, c.quer, rec.Code)
		}
	}
}
