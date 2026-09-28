### Validador eSocial — v1.3-alpha

Primeira versão **testada com certificado A1 de verdade na Produção Restrita do eSocial** (28/09/2026, e-CNPJ do
ICP-Brasil). Fluxo completo aceito com recibo: **S-1000** (empregador) → **S-1005** (estabelecimento) →
**S-2200** (admissão) → **S-2220** (ASO). Os XMLs de S-2210 e S-2240 usam o mesmo caminho corrigido.

---

### Guia de Download por Plataforma

| Arquivo do Pacote | Sistema Operacional Alvo | Arquitetura / Processador |
| :--- | :--- | :--- |
| **`ghcr.io/forg3/validador-esocial:v1.3-alpha`** | Container OCI (Docker / Podman / Kubernetes) | Linux amd64 (GitHub Packages) |
| **`validador-esocial_1.3-alpha_linux_amd64.deb`** | Debian, Ubuntu, Mint e derivados | Linux amd64 (arm64 também disponível) |
| **`validador-esocial_1.3-alpha_linux_amd64.rpm`** | Fedora, RHEL, CentOS, openSUSE | Linux amd64 (arm64 também disponível) |
| **`validador-esocial-1.3.0-windows-amd64.msi`** | Windows 10/11 e Windows Server (instalador) | 64-bit (x86_64 / amd64) |
| **`validador-esocial-1.3-alpha-linux-amd64.tar.gz`** | Linux (Ubuntu, Debian, Fedora, RHEL, Arch) | 64-bit (x86_64 / amd64) |
| **`validador-esocial-1.3-alpha-windows-amd64.zip`** | Windows 10, Windows 11, Windows Server | 64-bit (x86_64 / amd64) |
| **`validador-esocial-1.3-alpha-darwin-arm64.tar.gz`** | macOS (Sonoma, Sequoia ou superior) | **Apple Silicon (M1 a M6+)** |
| **`validador-esocial-1.3-alpha-darwin-amd64.tar.gz`** | macOS (High Sierra a Monterey ou superior) | Processadores **Intel 64-bit** |

Todas as releases incluem `SHA256SUMS.txt` para verificação de integridade.

---

### O que mudou nesta versão

Tudo abaixo apareceu **só com o governo de verdade** — o webservice simulado aceitava:

| # | Mudança | Resposta do eSocial antes |
|---|---|---|
| 1 | Leitura de `.pfx` em **BER** (formato comum do A1 do ICP-Brasil), reescrito em DER em memória com a senha conferida pelo MAC | o arquivo não abria ("indefinite length found (not DER)") |
| 2 | **Renegociação TLS** que o servidor do eSocial pede para o certificado do cliente | `tls: no renegotiation` |
| 3 | **Id do evento** com a raiz do CNPJ + `000000` (zeros à direita) | ocorrência 609 — código inválido |
| 4 | `ideEmpregador` com a **raiz do CNPJ** nos eventos de SST e no construtor genérico | ocorrência 599 — eventos de outro empregador |
| 5 | Assinatura com **`Reference URI=""`** sobre o documento inteiro (e reassinatura sem a assinatura anterior) | ocorrência 142 — assinatura inválida |
| 6 | Consulta no serviço **`retornoProcessamento` v1_1_0**; falha SOAP `<s:Fault>` vira erro, não "sucesso vazio" | `ActionNotSupported` tratado como código 0 |
| 7 | **Grupo do lote** pelo tipo de evento: 1 tabelas, 2 não periódicos (SST), 3 periódicos | ocorrência 101 — tipo de evento não aceito no lote |
| 8 | Novo gerador do **S-2200** (admissão, S-1.3) e do **S-1005** (estabelecimento), com os XSD oficiais embutidos; modelo do S-2200 no catálogo refeito no leiaute S-1.3 | sem S-2200, os eventos de SST voltavam com a ocorrência 1557 |
| 9 | S-2200 sem raça/cor válida não passa (Lei 14.553/2023 proíbe "6 – não informado") | ocorrência 1871 |
| 10 | S-1005 só informa a alíquota RAT quando é diferente da legal para o CNAE | ocorrência 1504 |
| 11 | XSD do S-1000 trocado pelo oficial (pacote de 01/07/2026) | — |

Testes de integração com a Produção Restrita (fora da suíte padrão):
`ESOCIAL_PFX=… ESOCIAL_PFX_SENHA=… go test -tags restrita ./internal/soap/ ./internal/crypto/`

### Ainda não testado

- Certificados A3 exigem a build com `-tags pkcs11` e um token físico.
- Envio em **Produção** (tpAmb=1): o mesmo código, mas cada empresa deve testar antes na Produção Restrita.
