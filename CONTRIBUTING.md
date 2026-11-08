# Como contribuir

1. Abra uma issue descrevendo o problema ou a ideia. Respondemos em até 48 horas úteis, sem SLA.
2. Crie uma branch a partir da `main` (`fix/ajuste-xsd`, `feat/meu-recurso`).
3. Rode `go vet ./...` e `go test ./...` (os testes de XSD precisam do `xmllint`).
4. Mudança em XML de evento deve passar pela validação contra o XSD oficial do leiaute vigente.
5. Não inclua certificados, senhas, tokens nem dados reais de trabalhadores em código, testes ou issues.
6. Abra o Pull Request explicando o que mudou e por quê.

Falhas de segurança: veja [SECURITY.md](SECURITY.md). Ao contribuir, você concorda em licenciar sua parte sob a [MIT](LICENSE).
