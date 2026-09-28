//go:build restrita

package crypto

import (
	"errors"
	"os"
	"testing"
)

// Carrega um .pfx real (ex.: o e-CNPJ do ICP-Brasil em BER) direto, sem reexportar. Senha errada tem de
// dar errSenhaPFX. `ESOCIAL_PFX=… ESOCIAL_PFX_SENHA=… go test -tags restrita -run PFXReal ./internal/crypto/`
func TestPFXRealCarregaDireto(t *testing.T) {
	caminho, senha := os.Getenv("ESOCIAL_PFX"), os.Getenv("ESOCIAL_PFX_SENHA")
	if caminho == "" || senha == "" {
		t.Skip("defina ESOCIAL_PFX e ESOCIAL_PFX_SENHA")
	}
	cert, err := CarregarA1Arquivo(caminho, senha)
	if err != nil {
		t.Fatalf("não carregou: %v", err)
	}
	t.Logf("ok: %s · CNPJ %s · válido até %s", cert.RazaoSocial(), cert.CNPJ(), cert.ValidoAte().Format("02/01/2006"))
	if _, err := CarregarA1Arquivo(caminho, senha+"x"); !errors.Is(err, errSenhaPFX) {
		t.Errorf("senha errada deveria dar errSenhaPFX, deu: %v", err)
	}
}
