# 📖 Rancher Log Viewer — Documentação Técnica

**Autor:** Luiz Ricardo Z. Calmon  
**Versão:** 1.0.0  
**Linguagem:** Go 1.25 (Golang)  
**Plataforma alvo:** Windows 64 bits  
**Última atualização:** Junho 2026

---

## 📋 Índice

1. [Visão Geral](#1-visão-geral)
2. [Arquitetura do Sistema](#2-arquitetura-do-sistema)
3. [Estrutura de Diretórios](#3-estrutura-de-diretórios)
4. [Dependências Externas](#4-dependências-externas)
5. [Descrição dos Arquivos e Funções](#5-descrição-dos-arquivos-e-funções)
   - [main.go](#51-maingo)
   - [config/config.go](#52-configconfiggo)
   - [api/client.go](#53-apiclientgo)
   - [logger/logger.go](#54-loggerloggergo)
   - [ui/setup.go](#55-uisetupgo)
   - [ui/clusters.go](#56-uiclustersgo)
   - [ui/namespaces.go](#57-uinamespacesgo)
   - [ui/pods.go](#58-uipodsgo)
   - [ui/logs.go](#59-uilogsgo)
   - [ui/about.go](#510-uiaboutgo)
6. [Fluxo de Execução](#6-fluxo-de-execução)
7. [APIs Consumidas](#7-apis-consumidas)
8. [Tratamento de Erros](#8-tratamento-de-erros)
9. [Compilação](#9-compilação)

---

## 1. Visão Geral

O **Rancher Log Viewer** é um aplicativo CLI (Command-Line Interface) com interface interativa de terminal que permite navegar por clusters Kubernetes gerenciados pelo Rancher, selecionar namespaces e pods, e visualizar logs em tempo real com streaming contínuo.

### Principais funcionalidades

- Configuração guiada na primeira execução
- Listagem dinâmica de clusters, namespaces e pods
- Streaming de logs em tempo real com `follow=true`
- Destaque de cores por severidade (ERROR, WARN)
- Busca de texto nos logs com navegação por ocorrências
- Alternância entre modos TXT e JSON (pretty-print)
- Scroll vertical e horizontal nos logs
- Salvamento de logs em arquivo (.txt ou .json)
- Tela "Sobre" com informações do aplicativo
- Retry automático para erros HTTP transientes (502/503/504)
- Navegação hierárquica com ESC para voltar entre telas

---

## 2. Arquitetura do Sistema

O sistema segue uma arquitetura modular em 4 pacotes (packages):

```
┌──────────────────────────────────────────────────┐
│                    main.go                        │
│         Orquestração do fluxo principal           │
│    (config → cluster → namespace → pod → logs)    │
├──────────┬──────────┬──────────┬─────────────────┤
│  config  │   api    │  logger  │       ui         │
│          │          │          │                   │
│ Config   │ Client   │ Init()   │ SetupModel       │
│ Load()   │ doGet()  │ Log()    │ ClusterModel     │
│ Save()   │ GetXxx() │ Info()   │ NamespaceModel   │
│ Exists() │ Stream() │ Errorf() │ PodModel         │
│          │          │ Close()  │ LogModel         │
│          │          │          │ AboutModel       │
└──────────┴──────────┴──────────┴─────────────────┘
```

### Padrão de UI: Bubble Tea (The Elm Architecture)

Todas as telas de UI implementam a interface `tea.Model` do framework Bubble Tea, que segue o padrão **Model-Update-View (MVU)**:

- **Model** — Struct com o estado completo da tela
- **Init()** — Comando inicial ao montar a tela
- **Update(msg)** — Recebe eventos (teclas, dados, resize) e retorna novo estado
- **View()** — Renderiza o estado como string para o terminal

Cada tela é executada como um `tea.Program` independente. O `main.go` orquestra a sequência, criando e destruindo programas conforme o fluxo de navegação.

---

## 3. Estrutura de Diretórios

```
Rancher-Go-log/
├── main.go                 # Ponto de entrada e orquestração do fluxo
├── go.mod                  # Definição do módulo e dependências
├── go.sum                  # Checksums das dependências
├── README.md               # Documentação do usuário
├── Documents.md            # Documentação técnica (este arquivo)
│
├── config/
│   └── config.go           # Gerenciamento de configuração (config.json)
│
├── api/
│   └── client.go           # Cliente HTTP para Rancher API e K8s Proxy
│
├── logger/
│   └── logger.go           # Sistema de logging em arquivo (debug.log)
│
└── ui/
    ├── setup.go            # Tela de configuração inicial
    ├── clusters.go         # Tela de seleção de cluster
    ├── namespaces.go       # Tela de seleção de namespace
    ├── pods.go             # Tela de seleção de pod
    ├── logs.go             # Tela de visualização de logs
    └── about.go            # Tela "Sobre" com informações do app
```

### Arquivos gerados em runtime

| Arquivo | Descrição |
|---------|-----------|
| `config.json` | Credenciais e URL do Rancher |
| `debug.log` | Log de diagnóstico (append) |
| `logs_<ns>_<pod>_<timestamp>.txt` | Log salvo em modo TXT |
| `logs_<ns>_<pod>_<timestamp>.json` | Log salvo em modo JSON |

Todos são criados na mesma pasta do executável.

---

## 4. Dependências Externas

| Pacote | Versão | Uso |
|--------|--------|-----|
| `charmbracelet/bubbletea` | v1.3.10 | Framework de UI interativa no terminal (padrão MVU) |
| `charmbracelet/bubbles` | v1.0.0 | Componentes prontos: `list`, `viewport`, `textinput` |
| `charmbracelet/lipgloss` | v1.1.0 | Estilização de texto (cores, bold, padding, margens) |

Todas as outras dependências em `go.mod` são indiretas (transitive), puxadas automaticamente pelos três pacotes acima.

---

## 5. Descrição dos Arquivos e Funções

---

### 5.1 `main.go`

**Pacote:** `main`  
**Responsabilidade:** Ponto de entrada da aplicação. Orquestra todo o fluxo de navegação e trata erros fatais e não-fatais.

#### Variáveis de estilo

```go
var errorStyle, successStyle, infoStyle, warnStyle lipgloss.Style
```

Estilos globais para mensagens coloridas no terminal:
- `errorStyle` — Vermelho + negrito para erros fatais
- `successStyle` — Verde para mensagens de sucesso
- `infoStyle` — Azul para mensagens informativas
- `warnStyle` — Amarelo para warnings não-fatais

#### Funções

| Função | Assinatura | Descrição |
|--------|-----------|-----------|
| `main` | `func main()` | Ponto de entrada. Trata flags `--about`/`--version`/`-v`. Inicializa logger, carrega config, e executa o loop principal de navegação: cluster → namespace → pod → logs. |
| `ensureConfig` | `func ensureConfig() (*config.Config, error)` | Verifica se `config.json` existe. Se sim, carrega. Se não, exibe o wizard de setup, valida os campos obrigatórios (URL, Username, Password) e salva. |
| `selectCluster` | `func selectCluster(client *api.Client) (string, string, error)` | Busca clusters via API, exibe tela de seleção. Retorna `(clusterID, clusterName, error)`. Retorna `"about"` como clusterID se o usuário pressionou `?`. |
| `selectNamespace` | `func selectNamespace(client *api.Client) (string, error)` | Busca namespaces do cluster selecionado via API, exibe tela de seleção. Retorna string vazia se ESC. |
| `selectPod` | `func selectPod(client *api.Client, namespace, clusterName string) (string, error)` | Busca pods do namespace via API K8s Proxy, exibe tela de seleção com status colorido. Retorna string vazia se ESC. |
| `streamLogs` | `func streamLogs(client *api.Client, namespace, podName string, cfg *config.Config) error` | Abre streaming com `tailLines=maxLines`, cria `LogModel` com config e client (para save completo), e executa a UI de logs. |
| `printError` | `func printError(context string, err error)` | Exibe erro fatal em vermelho e registra no debug.log. |
| `printWarning` | `func printWarning(context string, err error)` | Exibe warning em amarelo e registra no debug.log. Não encerra o app. |
| `showAbout` | `func showAbout()` | Exibe a tela "Sobre" do aplicativo. |

#### Fluxo de navegação (loops aninhados)

```
main()
├── ensureConfig()               // Config (uma vez)
└── for {                        // Loop externo: clusters
    ├── selectCluster()
    └── for nsLoop {             // Loop meio: namespaces
        ├── selectNamespace()
        └── for {                // Loop interno: pods
            ├── selectPod()
            └── streamLogs()     // Logs (ESC volta para pods)
            }
        }
    }
```

#### Tratamento de erros

- **Erros fatais** (config, clusters, namespaces): `printError()` + `os.Exit(1)`
- **Erros não-fatais** (pods vazios, erro de streaming): `printWarning()` + `break` para tela anterior
- **Sinal "about"**: `continue` no loop de clusters para reexibir a lista

---

### 5.2 `config/config.go`

**Pacote:** `config`  
**Responsabilidade:** Gerenciamento do arquivo de configuração `config.json`.

#### Constantes

```go
const DefaultMaxLines = 1000
```

#### Struct

```go
type Config struct {
    RancherURL         string `json:"rancher_url"`
    Username           string `json:"username"`
    Password           string `json:"password"`
    InsecureSkipVerify bool   `json:"insecure_skip_verify"`
    MaxLines           int    `json:"max_lines"`
}
```

| Campo | Tipo | Descrição |
|-------|------|-----------|
| `RancherURL` | string | URL base do Rancher (ex: `https://rancher.empresa.com.br`) |
| `Username` | string | Access Key da API Key do Rancher |
| `Password` | string | Secret Key da API Key do Rancher |
| `InsecureSkipVerify` | bool | Se `true`, ignora validação de certificado SSL |
| `MaxLines` | int | Buffer máximo de linhas no viewer (200–5000, padrão: 1000) |

#### Funções

| Função | Assinatura | Descrição |
|--------|-----------|-----------|
| `ConfigPath` | `func ConfigPath() string` | Retorna o caminho absoluto de `config.json` (mesma pasta do executável). Fallback para diretório corrente se `os.Executable()` falhar. |
| `Exists` | `func Exists() bool` | Verifica se `config.json` existe no caminho retornado por `ConfigPath()`. |
| `Load` | `func Load() (*Config, error)` | Lê e deserializa o `config.json` para a struct `Config`. Aplica default de 1000 se `max_lines` estiver ausente ou fora do intervalo 200–5000. |
| `Save` | `func Save(cfg *Config) error` | Serializa a struct `Config` com indentação e grava em `config.json`. Permissão `0644`. |

#### Detalhes de implementação

- O arquivo é salvo com `json.MarshalIndent` para legibilidade humana
- O caminho é resolvido a partir de `os.Executable()`, garantindo que o config fique junto ao .exe independente do diretório de trabalho
- O campo `max_lines` é validado ao carregar: se ausente, zero ou fora do range (200–5000), recebe o padrão `DefaultMaxLines` (1000)
- Ao ajustar o buffer com `+`/`-` na tela de logs, o config é salvo automaticamente com o novo valor

---

### 5.3 `api/client.go`

**Pacote:** `api`  
**Responsabilidade:** Cliente HTTP para comunicação com a API do Rancher (v3) e Kubernetes Proxy.

#### Structs

```go
type Client struct {
    httpClient *http.Client   // Cliente HTTP com TLS e timeout
    baseURL    string         // URL base do Rancher (sem barra final)
    username   string         // Access Key
    password   string         // Secret Key
    clusterID  string         // ID do cluster selecionado (definido em runtime)
}

type ClusterInfo struct {
    ID          string        // ID do cluster (ex: "c-abc12")
    DisplayName string        // Nome de exibição do cluster
}

type PodInfo struct {
    Name   string             // Nome do pod
    Status string             // Fase do pod (Running, Pending, Failed...)
}
```

#### Funções

| Função | Assinatura | Descrição |
|--------|-----------|-----------|
| `NewClient` | `func NewClient(cfg *config.Config) *Client` | Cria o cliente HTTP com TLS configurado (InsecureSkipVerify), timeout de 30s e credenciais. O `clusterID` começa vazio. |
| `SetCluster` | `func (c *Client) SetCluster(id string)` | Define o cluster ativo. Chamado após o usuário selecionar um cluster. |
| `GetClusters` | `func (c *Client) GetClusters() ([]ClusterInfo, error)` | Lista todos os clusters acessíveis. Endpoint: `GET /v3/clusters`. Retorna ID e nome de cada cluster. |
| `doGet` | `func (c *Client) doGet(fullURL string) ([]byte, error)` | Método HTTP GET genérico com autenticação Basic Auth, retry automático e tratamento de erros. Usado por todas as funções de consulta. |
| `GetNamespaces` | `func (c *Client) GetNamespaces() ([]string, error)` | Lista namespaces do cluster ativo. Endpoint: `GET /v3/clusters/{id}/namespaces`. |
| `GetPods` | `func (c *Client) GetPods(namespace string) ([]PodInfo, error)` | Lista pods de um namespace. Endpoint: `GET /k8s/clusters/{id}/api/v1/namespaces/{ns}/pods` (K8s Proxy). |
| `StreamLogs` | `func (c *Client) StreamLogs(namespace, podName string, tailLines int) (io.ReadCloser, error)` | Abre streaming de logs com carga inicial limitada a `tailLines`. Endpoint: `GET .../log?follow=true&timestamps=true&tailLines=N`. |
| `FetchAllLogs` | `func (c *Client) FetchAllLogs(namespace, podName string) (io.ReadCloser, error)` | Busca o log **completo** do pod (sem `follow`, sem `tailLines`). Usado pelo Ctrl+S para salvar o arquivo completo. Timeout de 120s. |
| `isRetryable` | `func isRetryable(statusCode int) bool` | Helper que retorna `true` para status 502, 503 e 504 (erros transientes). |

#### Mecanismo de Retry (`doGet`)

O método `doGet` implementa retry automático para erros transientes:

```
Tentativa 0: executa imediatamente
Tentativa 1: aguarda 2 segundos
Tentativa 2: aguarda 4 segundos
Tentativa 3: aguarda 6 segundos
```

- **Retries em:** HTTP 502, 503, 504 e erros de conexão
- **Sem retry em:** HTTP 401/403 (erro de autenticação) e outros status
- Cada tentativa registra log detalhado no `debug.log`

#### Streaming de Logs (`StreamLogs`)

- Usa um `http.Client` separado **sem timeout** (streaming pode durar horas)
- Compartilha o mesmo `Transport` (TLS) do cliente principal
- Retorna `resp.Body` (io.ReadCloser) para leitura contínua pelo `LogModel`
- O parâmetro `follow=true` mantém a conexão aberta para novos logs
- O parâmetro `timestamps=true` inclui timestamps em cada linha
- O parâmetro `tailLines=N` limita a carga inicial às últimas N linhas (valor de `max_lines` do config)

#### Download Completo (`FetchAllLogs`)

- Requisição separada **sem** `follow` e **sem** `tailLines`
- Retorna TODO o log disponível do pod desde o início
- Timeout de 120 segundos para pods com logs extensos
- Usado pelo `Ctrl+S` para garantir que o arquivo salvo contenha o log completo
- Fallback: se falhar, salva apenas as linhas em memória

#### Autenticação

Todas as requisições usam **HTTP Basic Auth**:
```
Authorization: Basic base64(username:password)
```
Onde `username` é o Access Key e `password` é o Secret Key gerados no Rancher.

---

### 5.4 `logger/logger.go`

**Pacote:** `logger`  
**Responsabilidade:** Sistema de logging thread-safe em arquivo para diagnóstico.

#### Variáveis internas

```go
var logFile *os.File    // Handle do arquivo debug.log
var mu      sync.Mutex  // Mutex para escrita thread-safe
```

#### Funções

| Função | Assinatura | Descrição |
|--------|-----------|-----------|
| `Init` | `func Init() error` | Abre/cria o arquivo `debug.log` em modo append na pasta do executável. Fallback para diretório corrente. |
| `Log` | `func Log(level, msg string)` | Escreve uma linha no log com timestamp e nível. Thread-safe via Mutex. Silenciosamente ignora se o arquivo não foi inicializado. |
| `Info` | `func Info(msg string)` | Wrapper para `Log("INFO", msg)`. |
| `Error` | `func Error(msg string)` | Wrapper para `Log("ERROR", msg)`. |
| `Errorf` | `func Errorf(format string, args ...interface{})` | Log de erro com formatação tipo `fmt.Sprintf`. |
| `Close` | `func Close()` | Fecha o arquivo de log. Chamado via `defer` no `main()`. Thread-safe. |

#### Formato do log

```
[2026-06-18 14:30:45] [INFO] === Rancher Log Viewer started ===
[2026-06-18 14:30:45] [INFO] Loading existing config.json
[2026-06-18 14:30:46] [INFO] GET https://rancher.empresa.com.br/v3/clusters
[2026-06-18 14:30:46] [ERROR] HTTP 502 (attempt 1): Bad Gateway
[2026-06-18 14:30:48] [INFO] Retry 1/3 after 2s...
```

#### Detalhes relevantes

- O arquivo é aberto em modo **append** (`os.O_APPEND`), preservando logs de execuções anteriores
- Todas as operações de escrita são protegidas por `sync.Mutex` para segurança em goroutines concorrentes (ex: streaming de logs em background)
- O `defer logger.Close()` no `main()` garante flush e fechamento limpo

---

### 5.5 `ui/setup.go`

**Pacote:** `ui`  
**Responsabilidade:** Tela de configuração inicial (wizard) exibida na primeira execução.

#### Struct

```go
type SetupModel struct {
    inputs     []textinput.Model   // 4 campos de entrada
    focusIndex int                  // Índice do campo focado (0-4, onde 4 = botão)
    done       bool                 // Se true, configuração completa
    Config     *config.Config       // Config resultante após submit
}
```

#### Campos do formulário

| Índice | Label | Placeholder | Tipo |
|--------|-------|-------------|------|
| 0 | Rancher URL | `https://rancher.empresa.com.br` | Texto (256 chars) |
| 1 | Username | `seu-usuario` | Texto (128 chars) |
| 2 | Password | `sua-senha` | Password (oculto, 256 chars) |
| 3 | Ignorar SSL (s/n) | `s/n (default: s)` | Texto (3 chars) |

#### Funções

| Função | Assinatura | Descrição |
|--------|-----------|-----------|
| `NewSetupModel` | `func NewSetupModel() SetupModel` | Cria o modelo com 4 inputs configurados. O primeiro campo já recebe foco. |
| `Init` | `func (m SetupModel) Init() tea.Cmd` | Retorna `textinput.Blink` para animação do cursor. |
| `Update` | `func (m SetupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd)` | Trata navegação entre campos (Tab/Enter/Up/Down) e submit. Ao confirmar, monta `config.Config` e sai. |
| `updateInputs` | `func (m *SetupModel) updateInputs(msg tea.Msg) tea.Cmd` | Propaga mensagens para todos os inputs simultaneamente. |
| `View` | `func (m SetupModel) View() string` | Renderiza o formulário com labels, inputs e botão "Salvar e Continuar". |
| `Done` | `func (m SetupModel) Done() bool` | Retorna se a configuração foi finalizada com sucesso. |

#### Comportamento de navegação

- **Tab / Enter / Down** — Avança para o próximo campo
- **Shift+Tab / Up** — Volta para o campo anterior
- **Enter no botão** — Submete o formulário e sai
- **Esc / Ctrl+C** — Cancela e sai sem salvar

#### Lógica de SSL

O valor do campo "Ignorar SSL" é convertido para boolean:
- Qualquer valor diferente de `"n"` resulta em `InsecureSkipVerify = true`
- O padrão (campo vazio ou `"s"`) ignora SSL

---

### 5.6 `ui/clusters.go`

**Pacote:** `ui`  
**Responsabilidade:** Tela de seleção de cluster com lista interativa e filtragem.

#### Structs

```go
type clusterItem struct {
    id          string   // ID do cluster (ex: "c-abc12")
    displayName string   // Nome de exibição
}

type ClusterModel struct {
    list         list.Model   // Componente de lista do bubbles
    selectedID   string       // ID do cluster selecionado
    selectedName string       // Nome do cluster selecionado
    quit         bool         // Se true, usuário cancelou
    showAbout    bool         // Se true, usuário pressionou '?'
}
```

#### Delegate de renderização (`clusterItemDelegate`)

Cada item é renderizado como:
```
  ▸ nome-do-cluster (c-abc12)     ← item selecionado (cor #170)
    outro-cluster (c-xyz12)        ← item normal
```

O ID do cluster é exibido em cinza (`#241`) ao lado do nome.

#### Funções

| Função | Assinatura | Descrição |
|--------|-----------|-----------|
| `NewClusterModel` | `func NewClusterModel(clusters []api.ClusterInfo) ClusterModel` | Cria a lista de clusters com filtro ativado. Título: "🐮 Selecione o Cluster". |
| `Update` | `func (m ClusterModel) Update(msg tea.Msg) (tea.Model, tea.Cmd)` | Trata Enter (seleciona), Esc (volta), `?` (sobre). Redimensiona com WindowSizeMsg. |
| `SelectedID` | `func (m ClusterModel) SelectedID() string` | Retorna o ID do cluster selecionado. |
| `SelectedName` | `func (m ClusterModel) SelectedName() string` | Retorna o nome do cluster selecionado. |
| `Quit` | `func (m ClusterModel) Quit() bool` | Retorna se o usuário cancelou (Esc/Ctrl+C). |
| `ShowAbout` | `func (m ClusterModel) ShowAbout() bool` | Retorna se o usuário pressionou `?` para ver "Sobre". |

#### Atalhos

| Tecla | Ação |
|-------|------|
| `↑/↓` | Navegar na lista |
| `Enter` | Selecionar cluster |
| `/` | Filtrar por nome (built-in do `list.Model`) |
| `?` | Exibir tela "Sobre" |
| `Esc` | Sair do aplicativo |

---

### 5.7 `ui/namespaces.go`

**Pacote:** `ui`  
**Responsabilidade:** Tela de seleção de namespace com lista interativa.

#### Estilos compartilhados

Este arquivo define dois estilos usados por todas as telas de lista:

```go
var listTitleStyle   lipgloss.Style   // Azul, negrito, padding esquerdo
var statusBarStyle   lipgloss.Style   // Cinza, padding esquerdo
```

#### Struct

```go
type NamespaceModel struct {
    list     list.Model   // Componente de lista
    selected string       // Namespace selecionado
    quit     bool         // Se true, usuário cancelou
}
```

#### Funções

| Função | Assinatura | Descrição |
|--------|-----------|-----------|
| `NewNamespaceModel` | `func NewNamespaceModel(namespaces []string) NamespaceModel` | Cria lista de namespaces com filtro. Título: "📦 Selecione o Namespace". |
| `Update` | `func (m NamespaceModel) Update(msg tea.Msg) (tea.Model, tea.Cmd)` | Trata Enter (seleciona), Esc (volta para clusters). |
| `Selected` | `func (m NamespaceModel) Selected() string` | Retorna o namespace selecionado. |
| `Quit` | `func (m NamespaceModel) Quit() bool` | Retorna se o usuário cancelou. |

---

### 5.8 `ui/pods.go`

**Pacote:** `ui`  
**Responsabilidade:** Tela de seleção de pod com indicadores de status coloridos.

#### Struct

```go
type podItem struct {
    name   string   // Nome do pod
    status string   // Fase: Running, Pending, Failed, CrashLoopBackOff...
}
```

#### Cores de status

| Status | Cor | Código |
|--------|-----|--------|
| `Running` | 🟢 Verde | `#82` |
| `Pending` | 🟡 Amarelo | `#220` |
| `Failed` / `CrashLoopBackOff` | 🔴 Vermelho | `#196` |
| Outros | ⚪ Cinza | `#245` |

#### Renderização de cada item

```
  ▸ meu-pod-abc123 [Running]     ← selecionado
    outro-pod-def456 [Pending]    ← normal, "Pending" em amarelo
```

#### Funções

| Função | Assinatura | Descrição |
|--------|-----------|-----------|
| `NewPodModel` | `func NewPodModel(pods []api.PodInfo, namespace, clusterName string) PodModel` | Cria lista de pods. Título: "🚀 {clusterName} > {namespace} — Pods". |
| `Update` | `func (m PodModel) Update(msg tea.Msg) (tea.Model, tea.Cmd)` | Trata Enter (seleciona pod), Esc (volta para namespaces), Ctrl+C (sai). |
| `Selected` | `func (m PodModel) Selected() string` | Retorna o nome do pod selecionado. |

---

### 5.9 `ui/logs.go`

**Pacote:** `ui`  
**Responsabilidade:** Tela principal de visualização de logs em tempo real. O componente mais complexo do sistema.

#### Estilos

| Variável | Cor | Uso |
|----------|-----|-----|
| `logTitleStyle` | Azul, negrito | Título da tela de logs |
| `errorLineStyle` | Vermelho (#196) | Linhas com ERROR/FATAL/EXCEPTION |
| `warnLineStyle` | Amarelo (#220) | Linhas com WARN/WARNING |
| `normalStyle` | Branco (#252) | Linhas normais |
| `searchHighlight` | Fundo amarelo (#226), texto preto | Linhas que correspondem à busca |
| `searchInputStyle` | Rosa (#205), negrito | Campo de busca |
| `savedMsgStyle` | Verde (#82), negrito | Mensagem de confirmação de salvamento |
| `footerStyle` | Cinza (#241) | Barra de atalhos |

#### Tipos de mensagem

```go
type logLineMsg string      // Nova linha de log recebida do stream
type logErrorMsg string     // Erro na leitura do stream
type logDoneMsg struct{}    // Stream finalizado
```

#### Struct principal

```go
type LogModel struct {
    viewport    viewport.Model   // Componente de viewport scrollável
    lines       []string         // Linhas formatadas (com cores)
    rawLines    []string         // Linhas brutas (sem formatação)
    title       string           // Título da tela
    podName     string           // Nome do pod
    namespace   string           // Namespace do pod
    reader      io.ReadCloser    // Stream de logs (resp.Body)
    scanner     *bufio.Scanner   // Scanner persistente para leitura
    ready       bool             // Se o viewport está inicializado
    err         string           // Mensagem de erro atual
    width       int              // Largura do terminal
    height      int              // Altura do terminal
    xOffset     int              // Deslocamento horizontal (scroll)
    jsonMode    bool             // Se true, formata JSON com indentação
    searching   bool             // Se true, campo de busca está ativo
    searchQuery string           // Texto sendo buscado
    searchHits  []int            // Índices das linhas com match
    searchIdx   int              // Índice do match atual
    statusMsg   string           // Mensagem temporária na barra de status
    maxLines    int              // Linhas iniciais carregadas da API (200–5000)
    cfg         *config.Config   // Referência ao config para salvar maxLines
    client      *api.Client      // Cliente API para FetchAllLogs (save completo)
}
```

#### Funções

| Função | Assinatura | Descrição |
|--------|-----------|-----------|
| `NewLogModel` | `func NewLogModel(podName, namespace string, reader io.ReadCloser, cfg *config.Config, client *api.Client) LogModel` | Inicializa o modelo com scanner de buffer 1MB, lê `maxLines` do config (padrão 1000). Recebe o client para save completo. |
| `Init` | `func (m LogModel) Init() tea.Cmd` | Dispara a primeira leitura de linha via `readNextLine()`. |
| `readNextLine` | `func (m LogModel) readNextLine() tea.Cmd` | Retorna um `tea.Cmd` que lê a próxima linha do scanner. Executado em goroutine pelo Bubble Tea. Retorna `logLineMsg`, `logErrorMsg` ou `logDoneMsg`. |
| `colorize` | `func colorize(line string) string` | Aplica cor baseada em keywords: ERROR/FATAL→vermelho, WARN→amarelo, outros→branco. Busca case-insensitive. |
| `refreshViewport` | `func (m *LogModel) refreshViewport()` | Reconstrói o conteúdo do viewport. Aplica busca (highlight) e scroll horizontal (xOffset). |
| `shiftLeft` | `func shiftLeft(s string, offset int) string` | Remove `offset` runes do início da string (scroll horizontal). Opera em runes para suporte UTF-8. |
| `jumpToHit` | `func (m *LogModel) jumpToHit()` | Posiciona o viewport para centralizar a linha do match atual. Trata wrap-around (último→primeiro e vice-versa). |
| `prettyJSON` | `func prettyJSON(s string) (string, bool)` | Detecta conteúdo JSON em uma linha (buscando `{` ou `[`) e formata com `json.Indent`. Preserva prefixos antes do JSON (ex: timestamps). Retorna `(formatted, true)` se sucesso. |
| `formatLine` | `func (m *LogModel) formatLine(raw string) string` | Formata uma linha: em modo JSON tenta pretty-print, depois aplica colorização. |
| `rebuildLines` | `func (m *LogModel) rebuildLines()` | Reconstrói todas as linhas formatadas a partir das linhas brutas. Chamado ao alternar modo TXT/JSON. |
| `saveLogsToFile` | `func (m *LogModel) saveLogsToFile() string` | Salva o log **completo** via `FetchAllLogs` (requisição dedicada). Fallback para linhas em memória se a requisição falhar. Extensão `.txt` ou `.json` conforme modo. |
| `Update` | `func (m LogModel) Update(msg tea.Msg) (tea.Model, tea.Cmd)` | Handler principal de eventos (detalhado abaixo). |
| `View` | `func (m LogModel) View() string` | Renderiza header + viewport + footer. O footer muda conforme contexto (busca, status, erro, atalhos). |

#### Fluxo de streaming (ciclo de leitura)

```
Init() → readNextLine()
           ↓
     logLineMsg(texto)
           ↓
  Update: append + formatLine + refreshViewport + GotoBottom
           ↓
     readNextLine()  ← loop contínuo
           ↓
     logDoneMsg      ← stream finalizado
```

#### Gerenciamento de memória

**Sem corte de linhas:** Todas as linhas recebidas (carga inicial + streaming) são mantidas em memória em `rawLines` e `lines`. Não há trimming.

O campo `maxLines` controla apenas quantas linhas são carregadas inicialmente da API (`tailLines`). O usuário pode subir no scroll para ver todas as linhas já recebidas.

| Propriedade | Valor |
|-------------|-------|
| Padrão | 1000 linhas iniciais |
| Mínimo | 200 |
| Máximo | 5000 |
| Incremento | 200 por vez (teclas `+`/`-`, aplica na próxima conexão) |

O footer exibe o total de linhas em memória em tempo real (`total:N`).

O valor é salvo automaticamente no `config.json` ao ser alterado pelo usuário com `+`/`-`.

#### Salvamento completo (`Ctrl+S`)

Ao salvar, o app faz uma requisição separada via `FetchAllLogs` para obter o log **completo** do pod (desde o início, sem limite de linhas). Se a requisição falhar, salva apenas as linhas em memória como fallback.

#### Scanner com buffer grande

O `bufio.Scanner` é configurado com buffer de **1 MB** para suportar linhas longas (ex: stack traces, JSON extenso):

```go
sc.Buffer(make([]byte, 0, 1024*1024), 1024*1024)
```

#### Modos de exibição

| Modo | Alternância | Exibição | Salvamento |
|------|------------|----------|------------|
| **TXT** (padrão) | `Tab` | Linhas brutas com cores | `.txt` |
| **JSON** | `Tab` | Pretty-print com indentação 2 espaços | `.json` |

#### Sistema de busca

1. Usuário pressiona `/` → modo busca ativado
2. Digita o texto → campo de busca exibido no footer
3. `Enter` → confirma busca, destaca todas as ocorrências em amarelo
4. `n` → próximo match | `N` → match anterior (wrap-around)
5. `Esc` → cancela/limpa busca

#### Scroll horizontal

- `→` avança 10 colunas | `←` volta 10 colunas | `Home` reseta
- Implementado via `shiftLeft()` que opera em runes (suporte Unicode)
- O xOffset é aplicado em `refreshViewport()` antes de enviar ao viewport

#### Atalhos completos

| Tecla | Ação |
|-------|------|
| `↑/↓` | Scroll vertical |
| `←/→` | Scroll horizontal (10 colunas) |
| `Home` | Reset scroll horizontal |
| `Tab` | Alternar TXT/JSON |
| `+` | Aumentar linhas iniciais (+200, máx 5000) |
| `-` | Diminuir linhas iniciais (-200, mín 200) |
| `/` | Iniciar busca |
| `n/N` | Navegar matches |
| `Ctrl+S` | Salvar log completo (via API) |
| `Esc/q` | Voltar (ou cancelar busca) |
| `Ctrl+C` | Sair do aplicativo |

---

### 5.10 `ui/about.go`

**Pacote:** `ui`  
**Responsabilidade:** Tela "Sobre" com informações do aplicativo.

#### Constantes

```go
const (
    AppName    = "Rancher Log Viewer"
    AppVersion = "1.0.0"
    AppAuthor  = "Luiz Ricardo Z. Calmon"
    AppLang    = "Go (Golang)"
    AppDesc    = "Visualização interativa de logs de pods Kubernetes via Rancher API"
)
```

#### Informações exibidas

| Label | Valor |
|-------|-------|
| Nome | Rancher Log Viewer |
| Versão | 1.0.0 |
| Autor | Luiz Ricardo Z. Calmon |
| Linguagem | Go (Golang) |
| Plataforma | Windows 64 bits |
| Finalidade | Visualização interativa de logs de pods Kubernetes via Rancher API |
| Interface | Terminal UI (Bubble Tea + Lip Gloss) |
| Autenticação | Basic Auth (Rancher API Key) |
| Portabilidade | Executável único, sem dependências externas |

#### Acesso

- **Via linha de comando:** `rancher-logs.exe --about` ou `--version` ou `-v`
- **Via interface:** Pressionar `?` na tela de seleção de clusters

---

## 6. Fluxo de Execução

```
┌─────────────────────────────────┐
│         Início (main)           │
│   Flags: --about / --version?   │──→ Sim ──→ Tela Sobre → Fim
│                                 │
│         Não ↓                   │
├─────────────────────────────────┤
│       Init Logger               │
│    (abre debug.log)             │
├─────────────────────────────────┤
│     config.json existe?         │
│  Sim → config.Load()           │
│  Não → Tela Setup (wizard)     │──→ Esc ──→ Fim
│         → config.Save()        │
├─────────────────────────────────┤
│     api.NewClient(cfg)          │
├─────────────────────────────────┤
│ ┌─ LOOP: Seleção de Cluster ─┐ │
│ │  GET /v3/clusters           │ │
│ │  Tela: ClusterModel         │ │
│ │  Esc → Fim                  │ │
│ │  ? → Tela Sobre → continua  │ │
│ │  Enter → SetCluster(id)     │ │
│ │                             │ │
│ │ ┌─ LOOP: Sel. Namespace ──┐ │ │
│ │ │  GET /v3/clusters/{}/ns │ │ │
│ │ │  Tela: NamespaceModel   │ │ │
│ │ │  Esc → volta cluster    │ │ │
│ │ │  Enter → namespace      │ │ │
│ │ │                         │ │ │
│ │ │ ┌─ LOOP: Sel. Pod ────┐ │ │ │
│ │ │ │  GET /k8s/.../pods  │ │ │ │
│ │ │ │  Tela: PodModel     │ │ │ │
│ │ │ │  Esc → volta ns     │ │ │ │
│ │ │ │  0 pods → warning   │ │ │ │
│ │ │ │  Enter → pod        │ │ │ │
│ │ │ │                     │ │ │ │
│ │ │ │  GET /k8s/.../log   │ │ │ │
│ │ │ │  Tela: LogModel     │ │ │ │
│ │ │ │  Streaming + busca  │ │ │ │
│ │ │ │  Esc → volta pod    │ │ │ │
│ │ │ └─────────────────────┘ │ │ │
│ │ └─────────────────────────┘ │ │
│ └─────────────────────────────┘ │
└─────────────────────────────────┘
```

---

## 7. APIs Consumidas

### Rancher v3 API

| Endpoint | Método | Uso | Retorno |
|----------|--------|-----|---------|
| `/v3/clusters` | GET | Listar clusters | `{ data: [{ id, name }] }` |
| `/v3/clusters/{id}/namespaces` | GET | Listar namespaces | `{ data: [{ id, name }] }` |

### Kubernetes Proxy API (via Rancher)

| Endpoint | Método | Uso | Retorno |
|----------|--------|-----|---------|
| `/k8s/clusters/{id}/api/v1/namespaces/{ns}/pods` | GET | Listar pods | `{ items: [{ metadata.name, status.phase }] }` |
| `.../pods/{pod}/log?follow=true&timestamps=true&tailLines=N` | GET | Stream de logs (últimas N linhas + follow) | Text stream (chunked) |
| `.../pods/{pod}/log?timestamps=true` | GET | Download completo (Ctrl+S) | Text completo |

### Headers enviados

```
Authorization: Basic <base64(username:password)>
Accept: application/json
Content-Type: application/json
```

O endpoint de logs usa `Accept: application/json` mas recebe text/plain streaming.

---

## 8. Tratamento de Erros

### Categorias de erro

| Categoria | Comportamento | Exemplos |
|-----------|--------------|----------|
| **Fatal** | `printError()` + `os.Exit(1)` | Erro de config, falha ao carregar clusters, falha ao carregar namespaces |
| **Warning** | `printWarning()` + volta à tela anterior | Namespace sem pods, erro no streaming de logs |
| **Transiente** | Retry automático (até 3x) | HTTP 502, 503, 504, erros de conexão |
| **Autenticação** | Erro imediato sem retry | HTTP 401, 403 |

### Registro de erros

Todos os erros são registrados em `debug.log` com:
- Timestamp preciso
- Nível (INFO/ERROR)
- URL da requisição
- Status HTTP e corpo da resposta
- Número da tentativa (em caso de retry)

---

## 9. Compilação

### Comando de build

```powershell
$env:GOOS="windows"
$env:GOARCH="amd64"
go build -ldflags="-s -w" -o rancher-logs.exe .
```

### Flags de build

| Flag | Efeito |
|------|--------|
| `-s` | Remove tabela de símbolos (reduz tamanho ~30%) |
| `-w` | Remove informações de debug DWARF |
| `GOOS=windows` | Cross-compile para Windows |
| `GOARCH=amd64` | Arquitetura 64 bits |

### Resultado

Executável único `rancher-logs.exe` (~7.5 MB) sem dependências externas.

---

*Documentação técnica — Rancher Log Viewer v1.0.0*  
*Autor: Luiz Ricardo Z. Calmon*
