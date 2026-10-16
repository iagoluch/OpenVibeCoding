# Guia do usuário do OpenVibeCoding

[Português](guia.md) · [English](../guide.md) · [Español](../es/guia.md) · [简体中文](../zh-CN/guide.md)

O OpenVibeCoding (OVC) é um agente de programação para terminal, escrito em Go com Bubble Tea. Ele inspeciona projetos, propõe alterações revisáveis, executa comandos aprovados, mantém o contexto de tarefas, testa sites no Chromium e pode executar ações aprovadas na área de trabalho, quando há suporte da plataforma.

## Instalar e iniciar

Baixe um binário em [Releases](https://github.com/iagoluch/OpenVibeCoding/releases/latest) ou use o instalador, que baixa o pacote correspondente ao sistema/CPU e verifica o SHA-256.

Linux/macOS:
```sh
git clone https://github.com/iagoluch/OpenVibeCoding.git
cd OpenVibeCoding
bash scripts/install.sh
# Se necessário, adicione ~/.local/bin ao PATH:
export PATH="$HOME/.local/bin:$PATH"
ovc --workspace /caminho/do/projeto
```

Windows (PowerShell): `./scripts/install.ps1 -AddToPath`; depois, reabra o terminal. Se a política de execução bloquear o script, use `powershell -ExecutionPolicy Bypass -File scripts/install.ps1 -AddToPath`. Para escolher uma versão, use `-Version v0.1.0` no PowerShell ou `OPENVIBECODING_VERSION=v0.1.0` no shell.

Compilar do código-fonte exige **Go 1.26.2 ou superior**: `go build -o ovc ./cmd/ovc`, seguido de `bash scripts/install.sh ./ovc` (Windows: `./scripts/install.ps1 -Binary ./ovc.exe -AddToPath`). `bash scripts/package.sh` gera pacotes Linux/macOS/Windows para amd64/arm64 e o arquivo `SHA256SUMS`; empacotar também exige Python 3. Executar um binário de release não exige Go. Os testes de navegador também precisam de Node e Chromium.

## Configuração inicial e credenciais

Na primeira execução, o OVC cria uma configuração padrão. Execute `ovc --setup` para usar o assistente: ele detecta servidores Ollama e LM Studio, lista provedores com planos gratuitos e os respectivos links para obter chaves, pode importar credenciais do OpenCode, salva a chave no cofre do sistema operacional e faz uma requisição real antes de ativar a configuração. Se o modelo preferido estiver indisponível, limitado por taxa ou ocupado, o assistente tenta o próximo modelo do provedor.

Para cadastrar uma credencial com entrada oculta no terminal:
```sh
ovc --credential openai
ovc --credential anthropic
ovc --credential gemini
```

O padrão é o cofre do sistema: Keychain no macOS, Credential Manager no Windows ou Secret Service no Linux. Se ele não estiver disponível, selecione explicitamente `--credential-file`; não existe fallback silencioso para texto puro. Variáveis como `OPENAI_API_KEY`, `ANTHROPIC_API_KEY` e `GEMINI_API_KEY` substituem credenciais salvas. Nunca cole segredos em conversas nem em URLs de endpoints.

Use `ovc --doctor` para verificar configuração e recursos sem exibir chaves. Se credenciais ou configurações mudarem enquanto o runtime estiver ativo, pare o runtime ocioso com `ovc --runtime stop` e reconecte. Os comandos `/model` e `/harness` passam a valer imediatamente.

## Provedores compatíveis

| Provedor/adaptador | Suporte |
| --- | --- |
| OpenAI | Streaming nativo de Chat Completions, ferramentas e uso opcional |
| Anthropic Claude | Streaming nativo da Messages API, ferramentas e uso |
| Google Gemini | Streaming nativo, chamadas/resultados de funções, imagens e preservação de assinaturas de pensamento |
| DeepSeek, Kimi, Qwen, Mistral | Endpoints compatíveis com OpenAI |
| Presets gratuitos | OpenRouter (`:free` no nome do modelo), Groq, Cerebras, NVIDIA NIM, Z.AI e Google AI Studio; consulte `ovc --setup` |
| Locais/personalizados | Qualquer endpoint compatível; HTTP em loopback é permitido e Ollama/LM Studio são detectados |

Modelos, limite de contexto, suporte a ferramentas, limite de saída, temperatura e preços ficam no catálogo de provedores. Confirme disponibilidade e endpoints regionais com o provedor. Rotas realmente gratuitas são registradas como US$ 0; serviços pagos exibem preços publicados por milhão de tokens; preços sem fonte confiável são marcados como desconhecidos. Requisições têm timeout, resposta limitada, cancelamento, novas tentativas para HTTP 429/5xx e erros sem dados sensíveis. Redirecionamentos são recusados; HTTP só é aceito em loopback.

Os presets são atualizados nas releases. `config.json` guarda apenas alterações feitas por você; as entradas não editadas recebem novos IDs, contextos e preços do catálogo.

## Fluxo de trabalho

1. Peça ao agente para inspecionar o projeto e planejar a alteração. As ferramentas respeitam nomes protegidos e padrões de `.gitignore`, limitam leituras/pesquisas e recusam caminhos fora do workspace.
2. Revise o diff unificado. **Ctrl+Y** aprova aquela operação; **Ctrl+N** rejeita. É possível substituir, criar, mover e excluir arquivos. Mudanças concorrentes invalidam propostas antigas.
3. Aprove comandos de teste/build. No Linux compatível, eles rodam no Bubblewrap, com acesso ao workspace, arquivos de runtime do sistema somente leitura e sem rede por padrão. Se não for possível impor isolamento, a execução falha de forma fechada.
4. Inspecione stdout, stderr, código de saída e `/changes`. A sessão pode ser retomada depois.

`--allow-unsandboxed-shell` permite explicitamente executar comandos no host em qualquer plataforma. Cada comando ainda precisa de aprovação, mas executa com privilégios do usuário e pode acessar arquivos e rede do host; isso não é sandbox de sistema operacional. Consulte [garantias de segurança](seguranca.md).

### Controles e comandos

Enter envia; Ctrl+J ou Alt+Enter insere nova linha; Alt+Seta para cima/baixo recupera entradas; Tab completa comandos slash; PgUp/PgDn rola a tela. Esc ou Ctrl+C cancela a operação atual. Ctrl+C quando ocioso encerra o cliente. A revisão de arquivos é por operação/arquivo e mostra diffs completos; não há seleção por hunk.

| Comandos | Finalidade |
| --- | --- |
| `/help`, `/status`, `/config` | Ajuda, estado da sessão/runtime e configurações efetivas |
| `/model [provider/model]`, `/provider [name]` | Selecionar provedor/modelo registrado |
| `/harness [review\|inspect]` | `review` permite operações aprovadas; `inspect` só permite leitura do projeto |
| `/sessions`, `/resume <id>`, `/new` | Navegar entre sessões persistentes |
| `/rename <title>`, `/delete <other-session-id>` | Gerenciar sessões |
| `/init`, `/undo`, `/doctor` | Propor `AGENTS.md`, desfazer a última alteração aplicada, diagnosticar runtime |
| `/context`, `/cost`, `/changes` | Contexto, uso/custo salvo e alterações de arquivos observadas |
| `/capabilities` | Verificar dependências de automação e sandbox |
| `/permissions [browser\|desktop\|clipboard\|network\|install on\|off]` | Consultar ou alterar permissões deste cliente |
| `/clear`, `/exit` | Limpar a visualização sem apagar dados; sair |

Um `AGENTS.md` na raiz do workspace é incluído na mensagem de sistema de cada turno (limite de 16 KiB). Ele informa convenções, mas é evidência, não autorização. `/init` pede ao agente que inspecione o projeto e proponha um arquivo usando o fluxo normal de revisão. Novas sessões recebem o título da primeira solicitação, que pode ser substituído por `/rename`.

`/undo` reverte alterações aplicadas uma por vez e sobrevive a reinicializações. Recusa-se a agir se o arquivo mudou depois da operação, se o caminho foi recriado ou se existe um turno ativo. Permissões extras não removem a aprovação por operação. Rede e instalação começam desativadas; instalar pacotes exige permissão separada. Uma shell aprovada é um ambiente geral de programação, então a política do sistema operacional — não apenas categorias de ferramentas — delimita o acesso real.

## Runtime compartilhado e sessões

O primeiro terminal inicia um runtime local e os demais o reutilizam. O daemon controla SQLite, configurações de provedores, auditoria e coordenação das tarefas. A comunicação usa HTTP autenticado em loopback, token aleatório em arquivo privado e trava do sistema operacional para evitar daemons duplicados. Navegadores não podem invocá-lo via requisições cross-origin.

```sh
ovc --runtime start
ovc --runtime status
ovc --runtime stop
ovc --resume SESSION_ID --workspace /caminho/do/projeto
ovc --export SESSION_ID > session.json
```

Cada cliente tem identidade e aprovações independentes. Turnos são serializados por sessão **e workspace**. Fechar um cliente cancela sua solicitação atual sem interromper tarefas de outros clientes. O daemon continua disponível até ser parado explicitamente; a parada é recusada enquanto houver tarefas ativas. Encerramento do sistema operacional cancela tarefas antes de fechar o armazenamento.

O SQLite guarda mensagens originais, resultados de ferramentas, aprovações, ciclo das tarefas, alterações de configuração, resumos e uso/custo de tokens. Migrações são transacionais. Após uma interrupção, o resultado de uma ferramenta fica marcado como incerto; alterações não são reproduzidas automaticamente. Exportações incluem histórico, resumo, uso e auditoria, e exigem que a sessão esteja ociosa. Capturas de tela são transitórias e não ficam no banco.

O preparo de contexto resume progressivamente trocas antigas completas sem apagar mensagens recentes ou o histórico original. IDs de origem permitem recuperar evidências via `recall`. Estimativas de tokens são conservadoras; valores exatos dependem do provedor. Resumos podem conter erros: evidências originais prevalecem e aprovações antigas não autorizam novas ações. Uma tarefa atual grande demais falha com mensagem explicativa, em vez de ser descartada silenciosamente.

## Testes de navegador

```sh
ovc --setup-browser
# Na TUI:
/permissions browser on
```

A configuração instala Playwright 1.58.2 e Chromium na pasta QA do aplicativo. Revise e aprove um fluxo com origens permitidas e etapas como `open`, `fill`, `click`, `press`, `select`, `expect_text`, `expect_url` e `screenshot`. Cada fluxo usa um contexto novo; origens não autorizadas, service workers, WebSockets e downloads são bloqueados. Resultados incluem texto DOM observado, título/URL, asserções, mensagens do console, erros de página e requisições falhas. Capturas podem ser analisadas por modelos com suporte a imagens durante o turno e são descartadas depois. Use dados sintéticos e ambientes de teste dedicados.

## Uso do computador/desktop

Ative `/permissions desktop on`; área de transferência exige `/permissions clipboard on` separadamente. Cada ação é revisada. As ações incluem inspeção, capturas, movimento/clique de ponteiro, rolagem, digitação/teclas, abertura/foco/fechamento de aplicativos e clipboard, conforme o suporte do backend.

| Plataforma | Backend e limites |
| --- | --- |
| Linux X11 | `xdotool`, ImageMagick `import` para capturas e `xclip` para clipboard; requer `DISPLAY` ativo. Wayland não é suportado pelo backend integrado. |
| macOS | AppleScript/System Events, `screencapture` e CoreGraphics via JavaScript for Automation para ponteiro/rolagem. Requer permissões de Acessibilidade e Gravação de Tela. |
| Windows | PowerShell, APIs Win32, entrada/clipboard/captura por Windows Forms. Requer desktop interativo; não controla UAC/áreas seguras nem contorna permissões do SO. |

`automation.applications` é uma lista de permissões para abrir/focar apps, não isolamento de processo para ações globais de mouse/teclado. Confira o alvo visível antes de aprovar uma entrada. Adaptadores nativos externos são configurados por `automation.desktop_backend` como array de executável/argumentos. Veja o [contrato de automação](automacao.md).

## Configuração, manutenção e verificação

Consulte [config.example.json](../../config.example.json). Os dados ficam em `os.UserConfigDir()/openvibecoding` (por exemplo, `~/.config`, Application Support no macOS ou AppData no Windows); `OPENVIBECODING_HOME` substitui o diretório. `OPENVIBECODING_PROVIDER` e `OPENVIBECODING_MODEL` substituem os padrões. Segredos de provedores não entram na configuração JSON nem na exportação.

O diretório contém `config.json`, arquivos SQLite de sessões, `runtime.json`/`runtime.lock`, `runtime.log` com metadados, credenciais opcionais e dependências de QA. No Unix, arquivos/diretórios usam permissões restritas; no Windows valem as ACLs do perfil. `--debug` registra metadados de diagnóstico, não corpos de requisição nem segredos. Pare o runtime antes de copiar os dados ou atualizar binários.

Atualize rodando o instalador com novo binário/release (`OPENVIBECODING_VERSION=v…` ou PowerShell `-Version v…`); configuração e sessões são preservadas. Pare o runtime e use `scripts/uninstall.sh` ou `scripts/uninstall.ps1` para remover o launcher. Apague os dados separadamente somente se quiser perder sessões e credenciais. Se necessário, ajuste o PATH no arquivo de inicialização do shell; a instalação PowerShell não altera política de execução nem pede privilégios de administrador.

Para validação, consulte [testes e diagnóstico](testes.md), [arquitetura](arquitetura.md) e [referências técnicas](referencias.md). Os testes locais cobrem sandbox, navegador e desktop no Linux, protocolos de provedores, concorrência/autenticação/aprovação do runtime, persistência e recuperação de contexto. Execução nativa em macOS/Windows, integração com cofres nessas máquinas e requisições reais faturáveis exigem ambientes/contas correspondentes. Compilar de forma cruzada não comprova o comportamento nativo.
