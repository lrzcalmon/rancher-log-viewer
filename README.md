# 🐮 Rancher Log Viewer

**Autor:** Luiz Ricardo Z. Calmon  
**Versão:** 1.0.0  
**Linguagem:** Go (Golang)

Aplicativo nativo para Windows (.exe) que permite navegar clusters, namespaces e pods no Rancher, visualizando logs em tempo real com interface interativa no terminal.

---

## 📋 Índice

- [Requisitos](#requisitos)
- [Instalação](#instalação)
- [Criando a Chave de API no Rancher](#criando-a-chave-de-api-no-rancher)
- [Primeira Execução](#primeira-execução)
- [Navegação](#navegação)
- [Visualização de Logs](#visualização-de-logs)
- [Atalhos de Teclado](#atalhos-de-teclado)
- [Arquivos Gerados](#arquivos-gerados)
- [Solução de Problemas](#solução-de-problemas)

---

## Requisitos

- **Windows 10/11** (64 bits)
- **Windows Terminal** (recomendado para melhor experiência visual) ou CMD/PowerShell
- Acesso de rede ao servidor Rancher

---

## Instalação

1. Copie o arquivo `rancher-logs.exe` para uma pasta de sua preferência (ex: `C:\Tools\Rancher-log\`)
2. Execute o aplicativo diretamente — não requer instalação, dependências ou permissões de administrador

### Compilando a partir do código

Requer Go 1.25+:

```bash
GOOS=windows GOARCH=amd64 go build -o rancher-logs.exe .
```

---

## Criando a Chave de API no Rancher

Para autenticar no Rancher, você precisa gerar uma **API Key** (chave de acesso). Siga os passos:

### Passo 1 — Acessar o Rancher

1. Abra o navegador e acesse a URL do seu Rancher (ex: `https://rancher.empresa.com.br`)
2. Faça login com suas credenciais

### Passo 2 — Gerar a API Key

1. No canto superior direito, clique no **ícone do seu usuário** (avatar)
2. Selecione **"Account & API Keys"** (ou "Conta e Chaves de API")
3. Clique em **"Create API Key"** (ou "Criar Chave de API")
4. Preencha os campos:
   - **Description** (Descrição): `Rancher Log Viewer` (ou qualquer nome descritivo)
   - **Expires** (Expira em): Defina conforme sua política de segurança (ex: 90 dias ou sem expiração)
   - **Scope** (Escopo): Pode deixar em "No Scope" para acesso completo, ou restringir a um cluster específico
5. Clique em **"Create"**

### Passo 3 — Copiar as Credenciais

Após a criação, o Rancher exibirá:

| Campo | Exemplo | Uso no App |
|-------|---------|------------|
| **Access Key (Username)** | `token-abc12` | Campo "Username" na configuração |
| **Secret Key (Password)** | `xxxxxxxxxxxxxxxxxxxx...` | Campo "Password" na configuração |

> ⚠️ **IMPORTANTE:** O Secret Key é exibido **apenas uma vez**. Copie e guarde em local seguro. Se perder, será necessário criar uma nova chave.

### Passo 4 — Validar a Chave (Opcional)

Para testar se a chave funciona, execute no PowerShell:

```powershell
curl -k "https://SEU_RANCHER/v3/clusters" -u "SEU_ACCESS_KEY:SEU_SECRET_KEY"
```

Se retornar JSON com os clusters, a chave está funcionando.

---

## Primeira Execução

Ao executar `rancher-logs.exe` pela primeira vez, será exibida a tela de configuração:

```
🐮 Rancher Log Viewer — Configuração Inicial

  Rancher URL:
  > https://rancher.empresa.com.br

  Username:
  > token-abc12

  Password:
  > ••••••••••••••••

  Ignorar SSL (s/n):
  > s

  [ Salvar e Continuar ]
```

| Campo | Descrição |
|-------|-----------|
| **Rancher URL** | URL base do Rancher (sem barra no final) |
| **Username** | Access Key gerado no passo anterior |
| **Password** | Secret Key gerado no passo anterior |
| **Ignorar SSL** | `s` para ignorar certificados autoassinados (padrão), `n` para validar SSL |

As configurações são salvas em `config.json` na mesma pasta do executável.

---

## Navegação

O fluxo de navegação segue a hierarquia:

```
Cluster → Namespace → Pod → Logs
```

### Seleção de Cluster

Lista todos os clusters disponíveis com nome e ID.

### Seleção de Namespace

Lista todos os namespaces do cluster selecionado.

### Seleção de Pod

Lista os pods do namespace com indicador de status colorido:
- 🟢 **Verde** — Running
- 🟡 **Amarelo** — Pending
- 🔴 **Vermelho** — Failed / CrashLoopBackOff
- ⚪ **Cinza** — Outros estados

### Navegação com ESC

| Tela atual | Ação do ESC |
|------------|-------------|
| Logs | Volta para a lista de Pods |
| Pods | Volta para a lista de Namespaces |
| Namespaces | Volta para a lista de Clusters |
| Clusters | Sai do aplicativo |

---

## Visualização de Logs

A tela de logs exibe o streaming em tempo real com:

- **Destaque de cores automático:**
  - 🔴 Vermelho — linhas com `ERROR`, `ERR`, `FATAL`, `EXCEPTION`
  - 🟡 Amarelo — linhas com `WARN`, `WARNING`
  - ⚪ Normal — demais linhas

- **Modos de exibição:**
  - **TXT** (padrão) — Exibe os logs como texto plano, linha a linha
  - **JSON** — Detecta conteúdo JSON e formata com indentação (pretty-print)

---

## Atalhos de Teclado

### Em todas as telas de lista (Clusters, Namespaces, Pods)

| Tecla | Ação |
|-------|------|
| `↑` / `↓` | Navegar pelos itens |
| `Enter` | Selecionar item |
| `/` | Filtrar/buscar na lista |
| `Esc` | Voltar à tela anterior |
| `Ctrl+C` | Sair do aplicativo |

### Na tela de Logs

| Tecla | Ação |
|-------|------|
| `↑` / `↓` | Scroll vertical |
| `←` / `→` | Scroll horizontal (10 colunas por vez) |
| `Home` | Resetar scroll horizontal |
| `Tab` | Alternar modo TXT / JSON |
| `+` | Aumentar linhas iniciais (+200, máx 5000) |
| `-` | Diminuir linhas iniciais (-200, mín 200) |
| `/` | Buscar texto no log |
| `n` | Próxima ocorrência da busca |
| `N` | Ocorrência anterior da busca |
| `Ctrl+S` | Salvar log em arquivo |
| `Esc` | Voltar (cancela busca ou volta aos pods) |
| `q` | Voltar (mesmo que Esc) |
| `Ctrl+C` | Sair do aplicativo |

### Na busca (após pressionar `/`)

| Tecla | Ação |
|-------|------|
| Digitar | Insere texto na busca |
| `Enter` | Confirma busca e destaca resultados |
| `Backspace` | Apaga último caractere |
| `Esc` | Cancela a busca |

---

## Arquivos Gerados

O aplicativo pode gerar os seguintes arquivos na mesma pasta do executável:

| Arquivo | Descrição |
|---------|-----------|
| `config.json` | Configurações de conexão (criado na primeira execução) |
| `debug.log` | Log de diagnóstico com detalhes de erros e operações |
| `logs_<namespace>_<pod>_<data>.txt` | Log salvo em modo TXT |
| `logs_<namespace>_<pod>_<data>.json` | Log salvo em modo JSON |

### Exemplo de config.json

```json
{
  "rancher_url": "https://rancher.empresa.com.br",
  "username": "token-abc12",
  "password": "SUA_SECRET_KEY_AQUI",
  "insecure_skip_verify": true,
  "max_lines": 1000
}
```

### Parâmetro max_lines (Linhas Iniciais)

| Propriedade | Valor |
|-------------|-------|
| Valor padrão | 1000 |
| Valor mínimo | 200 |
| Valor máximo | 5000 |
| Incremento | 200 por vez |

- Define quantas linhas são carregadas da API ao conectar (`tailLines`)
- Na primeira execução, o valor é definido automaticamente como **1000**
- Se o campo não existir no `config.json`, será assumido o padrão de 1000
- Use as teclas `+` e `-` na tela de logs para ajustar (aplica na próxima conexão)
- O novo valor é salvo automaticamente no `config.json`
- Todas as linhas recebidas (iniciais + streaming) são mantidas em memória — sem corte
- O footer exibe o total de linhas em memória em tempo real
- **Ctrl+S** salva o log **completo** do pod (via requisição dedicada à API), não apenas o que está em memória

---

## Solução de Problemas

### Erro: "server returned HTTP 502"

- O Rancher ou o proxy (Ingress/Nginx) está temporariamente indisponível
- O app tenta automaticamente até **3 vezes** com delay progressivo
- Valide com cURL: `curl -k "https://SEU_RANCHER/v3/clusters" -u "user:pass"`
- Se o cURL também falhar, o problema é na infraestrutura

### Erro: "authentication failed (HTTP 401)"

- Username ou Password incorretos
- A API Key pode ter expirado
- Solução: gere uma nova chave no Rancher e delete o `config.json` para reconfigurar

### Erro: "connection error" / timeout

- Verifique conectividade de rede com o Rancher
- Se usar VPN, confirme que está conectado
- Tente acessar a URL do Rancher pelo navegador

### Namespace sem pods

- O app exibe um aviso amarelo e retorna automaticamente à seleção de namespace
- Não é necessária nenhuma ação — selecione outro namespace

### Reconfigurar

- Delete o arquivo `config.json` da pasta do executável
- Execute novamente o `rancher-logs.exe` — a tela de configuração aparecerá

### Logs de diagnóstico

- Consulte o arquivo `debug.log` para detalhes técnicos de erros
- Cada execução adiciona informações ao final do arquivo

---

## Informações Técnicas

| Item | Detalhe |
|------|---------|
| Linguagem | Go (Golang) |
| Plataforma | Windows 64 bits |
| Interface | Terminal UI (Bubble Tea + Lip Gloss) |
| Autenticação | Basic Auth (API Key) |
| APIs consumidas | Rancher v3 (clusters, namespaces) + K8s Proxy (pods, logs) |
| Streaming | HTTP chunked + bufio.Scanner |
| Portabilidade | Executável único, sem dependências externas |

---

---

*Rancher Log Viewer v1.0.0 — Desenvolvido por Luiz Ricardo Z. Calmon*
