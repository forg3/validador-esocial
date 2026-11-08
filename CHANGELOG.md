# Changelog

Formato baseado em [Keep a Changelog](https://keepachangelog.com/pt-BR/1.1.0/). Versões seguem [SemVer](https://semver.org/lang/pt-BR/).

## [1.4.0-alpha] - 2026-11-08

### Adicionado
- Integração opcional com o Canteiro Prime (PR #6): tela **Canteiro Prime**, busca de CATs (S-2210) pendentes, assinatura e transmissão locais e devolução do recibo ou da rejeição. Configuração em `dados/canteiro.json` (permissão `0600`) ou `CANTEIRO_URL` e `CANTEIRO_TOKEN`.
- `Config.Validar`: confere endereço (https, ou `localhost`) e token antes de salvar.
- `CHANGELOG.md`, `SECURITY.md`, `CONTRIBUTING.md`, `CITATION.cff` e `docs/img/tela-principal.png`.

### Alterado
- README reescrito para pt-BR com instalação por sistema, uso em 5 passos, limitações e resumo em inglês; a matriz de eventos foi para `docs/eventos.md`.
- A release publicada passa a usar o trecho deste arquivo como texto.

### Segurança
- SQLite abre com `secure_delete` ligado.
- O cliente do Canteiro Prime recusa redirecionamentos, para não reenviar o token a outro endereço.
- Rejeição apenas local não é enviada ao Canteiro; resposta 409 (resultado já recebido) apaga o XML da fila local.

## [1.3-alpha] - 2026-09-28

### Adicionado
- S-2200 e S-1005 no leiaute S-1.3.
- Fluxo S-1000, S-1005, S-2200 e S-2220 aceito com recibo na Produção Restrita, com e-CNPJ A1 real.

### Corrigido
- Leitura de `.pfx` em BER, renegociação TLS pedida pelo servidor do eSocial, Id do evento, `ideEmpregador`, `Reference URI=""` na assinatura, namespace e SOAPAction da consulta e grupo do lote por tipo de evento.

## [1.2-alpha] - 2026-09-26

### Corrigido
- Pente fino e pentest: assinatura A1 real pela interface, ASO importado validado, evento rejeitado não é assinado, tabelas 13, 14, 15, 17 e 27 completas, nome fixo do certificado, `xmllint --nonet`.

## [1.1.1-alpha]

### Adicionado
- Relatórios S-5001/S-5011, múltiplos certificados e procurações, transmissão real opcional, suporte a A3 via PKCS#11 (build `-tags pkcs11`) e pacotes `.deb`, `.rpm` e `.msi`.

## [1.1-alpha]

### Segurança
- Auditoria de segurança com correções: autenticação, anti-CSRF, validação de `Host`, limites de entrada e leiautes S-1.3 aderentes ao XSD.

## [1.0-alpha]

- Primeira versão pública.

[1.4.0-alpha]: https://github.com/forg3/validador-esocial/compare/v1.3-alpha...v1.4.0-alpha
[1.3-alpha]: https://github.com/forg3/validador-esocial/releases/tag/v1.3-alpha
[1.2-alpha]: https://github.com/forg3/validador-esocial/releases/tag/v1.2-alpha
[1.1.1-alpha]: https://github.com/forg3/validador-esocial/releases/tag/v1.1.1-alpha
[1.1-alpha]: https://github.com/forg3/validador-esocial/releases/tag/v1.1-alpha
[1.0-alpha]: https://github.com/forg3/validador-esocial/releases/tag/v1.0-alpha
