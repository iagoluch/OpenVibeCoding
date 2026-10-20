# Arquitetura

[Português](arquitetura.md) · [English](../architecture.md) · [Español](../es/arquitectura.md) · [简体中文](../zh-CN/architecture.md)

## Módulos

| Módulo | Responsabilidade |
| --- | --- |
| `cmd/ovc` | CLI, configuração inicial, entrada oculta de credenciais, ciclo do runtime, diagnóstico/exportação/configuração de navegador |
| `internal/config` | Catálogo de modelos/provedores, protocolo, parâmetros, permissões e configuração segura de credenciais |
| `internal/credentials` | Cofre do SO e fallback explícito em arquivo restrito |
| `internal/provider` | Adaptadores de streaming OpenAI-compatible, Anthropic e Gemini nativos; ferramentas, imagens, uso, erros e novas tentativas |
| `internal/store` | Migrações SQLite, sessões, trocas originais, auditoria, resumos, uso e exportações |
| `internal/runtime` | IPC local autenticado, trava do SO, reutilização de daemon, coordenação por sessão/workspace e roteamento de aprovações |
| `internal/agent` | Loop limitado de modelo/ferramentas, planejamento, política do harness, aprovações, contexto incremental e recuperação de evidências |
| `internal/workspace` | Ferramentas de arquivo confinadas, regras ignore, pesquisa, inspeção Git, gravações/movimentos/exclusões revisáveis, shell e sandbox Linux |
| `internal/automation` | Worker Playwright integrado, backends nativos, capturas, verificações de capacidade e contrato de adaptadores externos |
| `internal/tui` | Transcrição em streaming Bubble Tea, entrada/histórico, seleção de modelo/harness, permissões, revisão e navegação entre sessões |
| `scripts` | Build, criação de arquivos compactados/checksums, download/instalação, atualização e desinstalação |

## Ciclo de um turno

Um cliente se conecta ao runtime existente ou inicia um. Seleciona uma sessão persistida e provedor/modelo/harness. O runtime valida a seleção e assume a coordenação do workspace/sessão durante o turno. Mensagens originais são salvas antes de requisições; o preparo de contexto resume progressivamente trocas antigas sem apagá-las. Chamadas de ferramentas do provedor são montadas e validadas. Ferramentas de leitura executam dentro da fronteira definida; operações que precisam de aprovação são enviadas ao cliente responsável e ficam pausadas até uma resposta. A entrada de auditoria é registrada antes da execução. Saídas observadas e uso são persistidos. O cancelamento é propagado aos provedores, processos e workers.

A TUI continua processando redimensionamento, rolagem, cancelamento e revisão enquanto as requisições executam. Fluxos de ferramentas são limitados a 16 requisições de modelo por turno. Etapas de navegador/desktop são operações reais de worker/API; ambientes incompatíveis retornam erros. Capturas são removidas da saída persistida da ferramenta e anexadas à próxima requisição do modelo apenas em memória transitória.

## Persistência e recuperação

SQLite é a fonte autoritativa. O esquema 1 introduziu sessões/mensagens/auditoria; o esquema 2 adicionou resumos e uso. Migrações são transacionais. WAL e uma conexão de banco serializam gravações dentro do serviço; a trava do daemon impede runtimes concorrentes. Exportações exigem sessão ociosa e incluem histórico, auditoria, resumo e evidências de custo. Trocas de ferramenta interrompidas recebem resultado explicitamente incerto na recuperação; ações nunca são repetidas às cegas.

## Limites por plataforma e verificação

O sandbox de shell integrado é Bubblewrap no Linux. Em outras plataformas, o confinamento necessário não está disponível e falha de forma fechada; o modo host é uma opção explícita. A automação de navegador usa Node/Playwright nos hosts suportados. Controle nativo usa ferramentas X11 no Linux, AppleScript/CoreGraphics no macOS e PowerShell/Win32 no Windows, com adaptadores JSON externos para integrações adicionais. APIs nativas e cofres do SO precisam de validação na máquina real, com as permissões concedidas pelo usuário.

O repositório contém testes unitários, de integração e segurança, além de CI para execução nativa por SO. Testes locais Linux verificam operações reais de navegador, sandbox e X11. Builds cruzados são evidência separada e não comprovam runtime nativo no macOS/Windows. Testes de aceitação com provedores exigem credenciais. Consulte [testes e diagnóstico](testes.md) e [referências técnicas](referencias.md).
