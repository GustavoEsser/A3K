# Pipeline A3K — Documentação

Documentação completa dos workflows de CI/CD e distribuição do projeto A3K.

---

## Visão geral

O projeto possui três workflows independentes, cada um com responsabilidade bem definida:

| Workflow | Arquivo | Gatilho |
|----------|---------|---------|
| [CI](#ci) | `workflows/ci.yml` | Push em `main` · Pull Request para `main` |
| [Release](#release) | `workflows/release.yml` | Push de tag `v*` |
| [Security](#security) | `workflows/security.yml` | Pull Request para `main` |

---

## CI

**Arquivo:** `workflows/ci.yml`

Roda a cada push em `main` e em todo PR. Os quatro jobs são **independentes e paralelos** — nenhum depende do resultado do outro.

```
push main  /  pull request
    ├── Build
    ├── Test
    ├── Lint
    └── GoReleaser check
```

### Jobs

#### Build
Verifica que o projeto compila nas três plataformas alvo antes de qualquer merge.

| Passo | Comando |
|-------|---------|
| go vet | `go vet ./...` |
| linux/amd64 | `CGO_ENABLED=0 go build -trimpath -o /dev/null .` |
| linux/arm64 | `CGO_ENABLED=0 GOARCH=arm64 go build -trimpath -o /dev/null .` |
| darwin/arm64 | `CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -o /dev/null .` |

> `CGO_ENABLED=0` produz binários estáticos sem dependência de libc — obrigatório para cross-compile e para evitar o bug `dyld: missing LC_UUID` no macOS 26.

#### Test
Executa a suite de testes com detector de race conditions e salva o relatório de cobertura.

```bash
go test -v -race -coverprofile=coverage.out ./...
```

O arquivo `coverage.out` é salvo como artifact do GitHub Actions por 7 dias.

#### Lint
Roda o `golangci-lint` com a configuração em `.golangci.yml`.

**Linters ativos:**

| Categoria | Linters |
|-----------|---------|
| Corretude | `errcheck`, `govet`, `staticcheck`, `ineffassign`, `unused` |
| Segurança | `gosec` (G101–G602, exceto G304¹) |
| Estilo | `revive`, `misspell`, `gofmt`, `goimports`, `gocritic` |
| Erros | `wrapcheck`, `errorlint` |
| Contexto | `noctx` |
| Manutenibilidade | `cyclop` (max 15), `gocognit` (max 130), `maintidx` |

> ¹ G304 (file inclusion via variable) está excluído intencionalmente — o kubeconfig é carregado por path configurável pelo usuário.

#### GoReleaser check
Valida o arquivo `.goreleaser.yaml` sem compilar nada. Garante que a configuração de release está correta antes de chegar na tag.

```bash
goreleaser check
```

---

## Release

**Arquivo:** `workflows/release.yml`  
**Gatilho:** push de qualquer tag no formato `v*` (ex: `v1.2.0`, `v2.0.0-beta.1`)

Fluxo **sequencial com dependências**:

```
git tag v1.x.x && git push origin v1.x.x
          │
          ▼
       [test]           ← gate: falha aqui cancela tudo
          │
          ▼
     [goreleaser]       ← needs: test
          │
          ▼
       [attest]         ← needs: goreleaser
```

### Permissões

| Job | Permissões |
|-----|-----------|
| `test` | `contents: read` (herda global) |
| `goreleaser` | `contents: write` · `id-token: write` · `attestations: write` |
| `attest` | `contents: read` · `id-token: write` · `attestations: write` |

> `id-token: write` é obrigatório para o cosign obter o token OIDC do GitHub e realizar a assinatura keyless.

### Job: test (gate)

Roda `go test ./...` antes de qualquer publicação. Se falhar, o pipeline para aqui e nada é publicado.

### Job: goreleaser

Instala as ferramentas e executa o GoReleaser:

| Ferramenta | Uso |
|------------|-----|
| `cosign` (sigstore) | Assina o `checksums.txt` via OIDC keyless |
| `syft` (anchore) | Gera o SBOM em formato SPDX JSON |
| `goreleaser ~> v2` | Orquestra tudo |

**O que o GoReleaser produz:**

```
dist/
├── a3k_v1.x.x_linux_amd64.tar.gz       ← binário + README + LICENSE + configs/config.yaml
├── a3k_v1.x.x_linux_arm64.tar.gz
├── a3k_v1.x.x_darwin_amd64.tar.gz
├── a3k_v1.x.x_darwin_arm64.tar.gz
├── a3k_v1.x.x_windows_amd64.zip
├── a3k_v1.x.x_linux_amd64.deb
├── a3k_v1.x.x_linux_arm64.deb
├── a3k_v1.x.x_linux_amd64.rpm
├── a3k_v1.x.x_linux_arm64.rpm
├── a3k_v1.x.x_linux_amd64.apk
├── a3k_v1.x.x_linux_arm64.apk
├── a3k_v1.x.x_linux_amd64.sbom.spdx.json   ← SBOM por plataforma
├── a3k_v1.x.x_linux_arm64.sbom.spdx.json
├── a3k_v1.x.x_darwin_amd64.sbom.spdx.json
├── a3k_v1.x.x_darwin_arm64.sbom.spdx.json
├── a3k_v1.x.x_windows_amd64.sbom.spdx.json
├── checksums.txt                            ← SHA256 de todos os assets
├── checksums.txt.pem                        ← certificado cosign
└── checksums.txt.sig                        ← assinatura cosign
```

Além dos arquivos, o GoReleaser também:
- Cria/atualiza a **GitHub Release** com changelog automático agrupado por tipo de commit
- Faz commit no repositório `GustavoEsser/homebrew-tap` atualizando a Cask

**Changelog — grupos:**

| Grupo | Regexp |
|-------|--------|
| ✨ New Features | `feat(...)!?:` |
| 🐛 Bug Fixes | `fix(...)!?:` · `refactor(...)!?:` |
| 🔒 Security | `sec(urity)(...)!?:` |
| ⚙️ Other Changes | tudo que não encaixar acima |

Commits com prefixo `docs:`, `test:`, `chore:` e mensagens de merge são **excluídos** do changelog.

### Job: attest

Baixa o `dist/` salvo pelo job anterior e gera attestations de provenance SLSA nível 3 via GitHub Actions nativo.

| Attestation | Arquivos |
|-------------|---------|
| Archives | `dist/*.tar.gz` · `dist/*.zip` · `dist/checksums.txt` |
| SBOMs | `dist/*.sbom.spdx.json` |

Os attestations ficam associados ao GitHub Release e permitem verificar a cadeia de custódia de cada artefato.

### Verificando a assinatura cosign

Após a release, qualquer pessoa pode verificar a autenticidade do `checksums.txt`:

```bash
cosign verify-blob \
  --certificate-identity-regexp="https://github.com/GustavoEsser/a3k/.github/workflows/release.yml" \
  --certificate-oidc-issuer="https://token.actions.githubusercontent.com" \
  --cert checksums.txt.pem \
  --signature checksums.txt.sig \
  checksums.txt
```

### Instalação após release

**Homebrew**
```bash
brew tap GustavoEsser/tap
brew install a3k
```

**Go install**
```bash
go install github.com/flysecurity/a3k@v1.x.x
```

**Download direto**
```bash
# Exemplo linux/amd64
curl -Lo a3k.tar.gz https://github.com/GustavoEsser/a3k/releases/download/v1.x.x/a3k_v1.x.x_linux_amd64.tar.gz
tar -xzf a3k.tar.gz
./a3k --help
```

---

## Security

**Arquivo:** `workflows/security.yml`  
**Gatilho:** Pull Request para `main`

Roda apenas em PRs. Bloqueia o merge se a mudança introduzir dependências com vulnerabilidades ou licenças não permitidas.

```
pull request para main
    └── Dependency review
          ├── bloqueia deps com CVE severity >= HIGH
          └── bloqueia licenças: GPL-2.0 · GPL-3.0 · AGPL-3.0
```

O resultado é comentado diretamente no PR (`pull-requests: write`), mostrando quais pacotes foram bloqueados e o motivo.

---

## Segredos necessários

| Secret | Onde configurar | Uso |
|--------|----------------|-----|
| `GITHUB_TOKEN` | Automático (GitHub Actions) | Criar GitHub Release, upload de assets |
| `HOMEBREW_TAP_GITHUB_TOKEN` | Settings → Secrets → Actions | Escrever no repositório `GustavoEsser/homebrew-tap` |

O `HOMEBREW_TAP_GITHUB_TOKEN` deve ser um PAT clássico ou fine-grained com permissão de `contents: write` no repositório `GustavoEsser/homebrew-tap`.

---

## Como criar uma release

```bash
# 1. garantir que main está limpo e os testes passam
make test

# 2. criar a tag anotada e empurrar — isso dispara o pipeline
git tag -a v1.2.0 -m "Release v1.2.0"
git push origin v1.2.0

# ou usando o Makefile (roda test + lint antes de tagar)
make release TAG=v1.2.0
```

O pipeline completo (test → goreleaser → attest) leva cerca de **3–5 minutos**.

---

## Estrutura de arquivos do pipeline

```
.github/
├── README.md                  ← este arquivo
└── workflows/
    ├── ci.yml                 ← CI contínuo (build · test · lint · goreleaser-check)
    ├── release.yml            ← Release (goreleaser · cosign · sbom · homebrew · attest)
    └── security.yml           ← Segurança (dependency-review em PRs)

.goreleaser.yaml               ← Configuração completa do GoReleaser v2
.golangci.yml                  ← Configuração dos 15 linters
```
