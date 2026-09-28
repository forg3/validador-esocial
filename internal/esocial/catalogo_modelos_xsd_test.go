package esocial

import "testing"

// Os modelos do catálogo com XSD oficial embutido têm de passar no esquema já preenchidos pelo construtor
// genérico (o S-2200 antigo usava nomes de antes do S-1.3 e não passaria).
func TestModelosDoCatalogoPassamNoXSDOficial(t *testing.T) {
	if !XmllintDisponivel() {
		t.Skip("xmllint ausente")
	}
	for _, codigo := range []string{"S-1000", "S-1005", "S-2200"} {
		xml, err := GerarXMLEventoGenerico(codigo, map[string]string{"cnpj": "36380294000171"})
		if err != nil {
			t.Fatalf("%s: %v", codigo, err)
		}
		if err := ValidarXSD(xml, codigo); err != nil {
			t.Errorf("%s fora do XSD oficial: %v\n%s", codigo, err, xml)
		}
	}
}
