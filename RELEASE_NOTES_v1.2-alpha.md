### Validador eSocial — v1.2-alpha

Release de **pente fino e pentest** sobre a v1.1.1-alpha: a assinatura A1 passa a ser real pela interface, a
importação valida o XML na entrada, evento rejeitado nunca é assinado e as tabelas oficiais 13/14/15/17/27 ficam
completas. Continua sendo a versão de testes de conformidade com o Leiaute S-1.3 (NT 07/2026).

---

### Guia de Download por Plataforma

| Arquivo do Pacote | Sistema Operacional Alvo | Arquitetura / Processador |
| :--- | :--- | :--- |
| **`ghcr.io/forg3/validador-esocial:v1.2-alpha`** | Container OCI (Docker / Podman / Kubernetes) | Linux amd64 (GitHub Packages) |
| **`validador-esocial_1.2-alpha_linux_amd64.deb`** | Debian, Ubuntu, Mint e derivados | Linux amd64 (arm64 também disponível) |
| **`validador-esocial_1.2-alpha_linux_amd64.rpm`** | Fedora, RHEL, CentOS, openSUSE | Linux amd64 (arm64 também disponível) |
| **`validador-esocial-1.2.0-windows-amd64.msi`** | Windows 10/11 e Windows Server (instalador) | 64-bit (x86_64 / amd64) |
| **`validador-esocial-1.2-alpha-linux-amd64.tar.gz`** | Linux (Ubuntu, Debian, Fedora, RHEL, Arch) | 64-bit (x86_64 / amd64) |
| **`validador-esocial-1.2-alpha-windows-amd64.zip`** | Windows 10, Windows 11, Windows Server | 64-bit (x86_64 / amd64) |
| **`validador-esocial-1.2-alpha-darwin-arm64.tar.gz`** | macOS (Sonoma, Sequoia ou superior) | **Apple Silicon (M1 a M6+)** |
| **`validador-esocial-1.2-alpha-darwin-amd64.tar.gz`** | macOS (High Sierra a Monterey ou superior) | Processadores **Intel 64-bit** |

Todas as releases incluem `SHA256SUMS.txt` para verificação de integridade.

---

### O que mudou nesta versão

| # | Mudança | Por quê |
|---|---|---|
| 1 | **Assinatura A1 real pela fila** (avulsa e em lote) quando a senha do certificado é informada; sem senha, continua simulada e marcada como tal | Antes a interface sempre gerava o envelope simulado, e a transmissão real exige assinatura real |
| 2 | XML de ASO importado é **validado no XSD na entrada**; o que não passa entra como **rejeitado**, com o motivo | Entrava como "pronto" sem validação |
| 3 | Evento **rejeitado não é assinado**, e toda assinatura revalida o XML antes | Assinar aceitava evento rejeitado ou nunca validado |
| 4 | **Tabelas completas** do leiaute S-1.3: 13 (45 itens), 14 (248), 15 (60), 17 (29), 27 (1.450) | Eram amostras |
| 5 | Certificado enviado gravado com **nome fixo**, só `.pfx`/`.p12` | O nome vinha de quem enviava |
| 6 | `xmllint --nonet` | Validação nunca acessa a rede |
| 7 | Código morto removido, API obsoleta de teste trocada e **`gofmt`** em todo o código | Manutenção |

### Pentest (instância rodando)

Acesso sem login (303 para `/login`), `Host` forjado (421), força bruta no login (429 após 8 tentativas),
escrita sem CSRF ou com `Origin` de outro site (403), XXE e "billion laughs" (recusados, memória estável):
tudo bloqueado. `govulncheck` sem vulnerabilidade alcançável. Detalhes e falsos positivos do `gosec`:
`docs/security-audit/pente-fino-2026-09-26.md`.

### Ainda não testado

- Assinatura e transmissão com **certificado A1 de verdade em Produção Restrita** (aguarda o e-CNPJ).
- Certificados A3 exigem a build com `-tags pkcs11` e um token físico.
