<p align="center"><img src="assets/banner-pt-br.svg" alt="OpenVibeCoding — Da intenção à mudança verificada" width="100%"></p>

<p align="center">
  <strong>Um agente de programação que trabalha no seu terminal — com você no controle.</strong><br>
  Entenda o projeto. Revise as mudanças. Execute os testes. Confira o resultado.
</p>

<p align="center">
  <a href="README.md">🇧🇷 Português</a> ·
  <a href="README.en.md">🇺🇸 English</a> ·
  <a href="README.es.md">🇪🇸 Español</a> ·
  <a href="README.zh-CN.md">🇨🇳 简体中文</a>
</p>

<p align="center">
  <a href="#instalação">Instalar</a> ·
  <a href="docs/guide.md">Guia</a> ·
  <a href="docs/security.md">Segurança</a> ·
  <a href="docs/architecture.md">Arquitetura</a>
</p>

<p align="center">
  <a href="https://github.com/iagoluch/OpenVibeCoding/stargazers">⭐ Gostou da proposta? Dê uma estrela ao projeto</a>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.26.2%2B-00ADD8?logo=go&logoColor=white" alt="Go 1.26.2 ou superior">
  <img src="https://img.shields.io/badge/licença-MIT-brightgreen" alt="Licença MIT">
  <img src="https://img.shields.io/badge/Linux-macOS-Windows-informational" alt="Linux, macOS e Windows">
</p>

## O que é o OpenVibeCoding?

O **OpenVibeCoding (OVC)** é um agente de programação de código aberto, escrito em Go, que reúne conversa com modelos de IA e ferramentas de desenvolvimento em um fluxo de trabalho revisável no terminal.

Em vez de apenas sugerir código, o OVC pode inspecionar um projeto, propor mudanças em arquivos, pedir sua aprovação antes de aplicá-las e executar comandos aprovados para verificar o resultado. As sessões ficam salvas localmente para que o contexto e as evidências não desapareçam ao fechar o terminal.

**A ideia central é simples:** o modelo propõe, você revisa, o programa executa a operação autorizada e o resultado observado fica disponível.

## Por que experimentar?

| Entender | Alterar com controle | Verificar |
| --- | --- | --- |
| Ler arquivos, pesquisar código, consultar instruções do projeto e inspecionar o Git | Revisar diffs e aprovar ou rejeitar operações individuais em arquivos | Executar comandos aprovados e consultar o código de saída e a saída real |
| Manter sessões e contexto persistentes | Restringir ferramentas de arquivo à pasta de trabalho configurada | Usar fluxos de navegador com Playwright e integrações de desktop quando configurados |

### Escolha o modelo e o provedor

O OVC não exige que você use um único fornecedor. Há adaptadores nativos para OpenAI, Anthropic/Claude e Google Gemini, além de suporte a endpoints compatíveis com a API da OpenAI — incluindo opções para DeepSeek, Kimi, Qwen, Mistral e modelos locais.

O assistente de configuração `ovc --setup` lista opções de provedores e presets gratuitos disponíveis no momento, detecta servidores Ollama e LM Studio e pode importar credenciais já configuradas no OpenCode. A disponibilidade, os limites e as condições de cada provedor podem mudar.

### Sessões persistentes e evidências

O histórico é armazenado localmente em SQLite, incluindo conversas, resumos de contexto, auditoria e uso. Os resultados devem refletir o que realmente aconteceu: um comando reprovado não é tratado como aprovado, e um processo com código de saída diferente de zero não deve ser apresentado como sucesso.

### Automação com aprovação

O OVC inclui fluxos de QA de navegador com Playwright e integrações de desktop dependentes do sistema operacional. Configure apenas as capacidades necessárias para a tarefa. Ações de automação exigem concessões explícitas e aprovação conforme o contrato de cada ferramenta.

## Instalação

### Linux e macOS

O instalador baixa o pacote correspondente ao sistema e à arquitetura e verifica seu checksum SHA-256. Não é necessário instalar Go para usar um binário de release.

```sh
git clone https://github.com/iagoluch/OpenVibeCoding.git
cd OpenVibeCoding
bash scripts/install.sh
ovc --setup
ovc --workspace /caminho/do/seu/projeto
```

Se o comando `ovc` não for encontrado, adicione `~/.local/bin` ao `PATH` e abra um novo terminal.

### Windows (PowerShell)

Na pasta do repositório, execute:

```powershell
./scripts/install.ps1 -AddToPath
```

Feche e reabra o PowerShell após a instalação. Se a política de execução bloquear o script, execute explicitamente:

```powershell
powershell -ExecutionPolicy Bypass -File scripts/install.ps1 -AddToPath
```

Também é possível baixar o pacote pronto para seu sistema na página de [Releases](https://github.com/iagoluch/OpenVibeCoding/releases/latest).

### Primeira execução

`ovc --setup` orienta a configuração de provedores, credenciais e modelo. Depois, aponte o agente para a pasta de um projeto:

```sh
ovc --workspace /caminho/do/seu/projeto
```

Para verificar a configuração sem exibir chaves, use `ovc --doctor`.

### Compilar a partir do código-fonte

É necessário Go **1.26.2 ou superior**:

```sh
go build -o ovc ./cmd/ovc
```

No Linux/macOS, instale o binário gerado com `bash scripts/install.sh ./ovc`. No Windows, compile como `ovc.exe` e use `./scripts/install.ps1 -Binary ./ovc.exe -AddToPath`.

## Um ciclo típico de trabalho

```text
❯ Investigue este parser, proponha uma correção pequena e verifique-a.

ENTENDER   Leia a implementação, as instruções e os testes relevantes.
REVISAR    Examine o diff proposto antes de qualquer alteração.
APROVAR    Confirme ou rejeite a operação individual.
VERIFICAR  Execute o comando autorizado e confira o resultado real.
```

Esse exemplo ilustra o fluxo; não representa uma execução real nem garante que qualquer alteração será bem-sucedida.

Atalhos e comandos úteis:

```text
/model <provedor/modelo>   /harness inspect
/sessions                  /resume <id>
/context                   /cost
/changes                   /permissions
/capabilities              /help
/init · /undo · /doctor
```

As instruções do projeto em `AGENTS.md` são incluídas no contexto de trabalho. Consulte `/help` para os comandos disponíveis na versão instalada.

## Segurança: saiba os limites

- As ferramentas comuns de leitura e escrita de arquivos são restritas ao workspace configurado e aplicam políticas de exclusão para caminhos sensíveis.
- Alterações de arquivos são apresentadas para revisão e exigem aprovação individual antes de serem aplicadas.
- No Linux, a execução de comandos usa Bubblewrap por padrão e falha de forma fechada se o sandbox exigido não estiver disponível.
- **O sandbox de processos integrado é somente para Linux.** No macOS e Windows, comandos que exigem confinamento falham de forma fechada. A opção `--allow-unsandboxed-shell` habilita explicitamente execução com os privilégios do usuário do sistema e não deve ser confundida com isolamento.
- As integrações de desktop dependem de permissões e capacidades específicas de cada plataforma. A compilação cruzada não comprova o comportamento nativo.

Leia [Segurança e limites de execução](docs/security.md) antes de habilitar shell ou automação. Não há garantia de que operações externas, processos destacados ou ações em aplicativos sejam sempre reversíveis.

## Documentação

| Documento | Conteúdo |
| --- | --- |
| [Guia do usuário](docs/guide.md) | Instalação, configuração, provedores e comandos |
| [Arquitetura](docs/architecture.md) | Módulos, ciclo do agente, persistência e responsabilidades por plataforma |
| [Segurança](docs/security.md) | Aprovações, limites de confinamento, credenciais e execução |
| [Automação](docs/automation.md) | Fluxos de navegador, ações de desktop e adaptadores |
| [Testes e diagnóstico](docs/testing.md) | Testes automatizados, validação de ambiente e solução de problemas |
| [Site de apresentação](site/) | Código da landing page estática; a publicação no GitHub Pages requer configuração própria |

## Como ajudar

- **Experimente e dê uma estrela:** [⭐ OpenVibeCoding no GitHub](https://github.com/iagoluch/OpenVibeCoding/stargazers).
- **Reporte um problema:** [abra uma issue](https://github.com/iagoluch/OpenVibeCoding/issues).
- **Explore o código e proponha melhorias:** pull requests são bem-vindos; consulte as instruções do repositório antes de contribuir.

O projeto é distribuído sob a licença [MIT](LICENSE).

---

<p align="center"><sub>OpenVibeCoding — do pedido à mudança verificada, com aprovação humana no fluxo.</sub></p>
