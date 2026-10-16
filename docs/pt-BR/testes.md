# Validação e diagnóstico

[Português](testes.md) · [English](../testing.md) · [Español](../es/pruebas.md) · [简体中文](../zh-CN/testing.md)

## Verificações automatizadas

```sh
go test ./...
go test -race ./...
go vet ./...
bash scripts/package.sh
```

A cobertura inclui protocolos de streaming nativos/compatíveis, fragmentos de ferramentas/uso/erros/redirecionamentos, validação de endpoints, permissões e remoção do fallback de credenciais, persistência/recuperação SQLite, resumos incrementais sem repetição, aprovação antes de gravação, autenticação do runtime, prevenção de instâncias duplicadas, exclusão de sessões, exportação, limites de ignore/caminhos/links simbólicos, diffs desatualizados, criação/movimento/exclusão, código de saída/cancelamento de comandos, validação de comandos slash e encerramento da interface.

Testes de integração do ambiente exigem fixtures explícitas:

```sh
# Instale primeiro as dependências do navegador:
ovc --setup-browser
OPENVIBECODING_TEST_PLAYWRIGHT=/caminho/dos-dados/qa/node_modules/playwright go test ./internal/automation -run TestRealBrowserWorkflow
OPENVIBECODING_TEST_SANDBOX=1 go test ./internal/workspace -run TestSandboxBoundary
# Use apenas um desktop de teste isolado, nunca sua sessão de trabalho normal:
DISPLAY=:93 OPENVIBECODING_TEST_DESKTOP=1 go test ./internal/automation -run TestRealDesktop
```

O teste real de navegador preenche um formulário sintético, clica, verifica o resultado visível, captura uma imagem e confirma o bloqueio de origem não autorizada. O teste real de desktop movimenta/clica/rola, observa a posição do ponteiro, captura a tela e digita em um processo xterm sintético, verificando a entrada por arquivo fixture. Xvfb precisa de `-noreset` para não reiniciar a posição do ponteiro entre processos de ferramenta.

O teste de sandbox confirma acesso ao projeto escolhido, nega um arquivo fixture externo, mascara `.env` e observa uma gravação dentro do workspace. A política de namespaces do host pode impedir Bubblewrap; registre isso como indisponibilidade em vez de recorrer silenciosamente à execução no host.

## Evidências locais e limites

Durante o desenvolvimento, os pacotes passaram nos testes unitários/de integração e no detector de condições de corrida. Testes reais de navegador, Bubblewrap e X11 passaram no Linux. Dois clientes de terminal se conectaram ao mesmo daemon; fechar um não interrompeu o outro. Fixtures cobriram OpenAI, Anthropic e Gemini. A geração de arquivos compactados compilou seis alvos com o launcher `ovc`.

Uma primeira execução de fixture X11 falhou porque o Xvfb reiniciava após a desconexão dos clientes; `-noreset` corrigiu o ambiente e o teste real de mouse/teclado passou. Playwright identifica Amazon Linux como distribuição não suportada e baixa a versão fallback do Ubuntu; o fluxo Chromium foi executado com sucesso nesse ambiente.

Não se afirma que tenham sido executadas requisições pagas a provedores externos nem testes nativos de GUI/cofre de chaves em macOS/Windows. Os adaptadores de provedores foram testados com respostas HTTP/SSE sintéticas. A configuração da CI para testes nativos não significa que uma execução remota específica tenha passado; consulte as execuções registradas.

## Aceitação com um provedor real

1. Crie um projeto descartável com README e dados sintéticos. Cadastre credenciais por `ovc --credential <provider>`; não as cole no projeto ou sessão.
2. Execute `--doctor`, abra o projeto com `--workspace` e selecione `/model provider/model`.
3. Peça para inspecionar o README e criar `hello.txt`. Rejeite uma proposta e verifique que o arquivo não existe; aprove uma nova proposta e confira o arquivo real.
4. Aprove um comando de teste inofensivo e confira saída e código de erro não zero deliberado. Use isolamento quando disponível; modo host exige escolha explícita.
5. Saia, retome e verifique transcrição, `/changes` e `/cost` salvo.
6. Repita com Anthropic/Gemini e endpoint compatível. Confirme continuidade das respostas de ferramentas depois de retomar ou trocar modelo.
7. Para modelos com imagem, habilite navegador/desktop, aprove uma captura e verifique que o modelo se baseia na evidência vista. Ao retomar, confirme que solicita nova captura em vez de inventar detalhes antigos.

## Solução de problemas

- **Inicialização do runtime:** execute `ovc --runtime serve` no terminal para ver erros de setup. `--runtime status` verifica autenticação compatível. Não apague um arquivo de trava ativo; a propriedade usa uma trava do SO. Após uma falha, reconecte normalmente.
- **Credenciais/configuração alteradas:** pare o runtime ocioso e reinicie-o. Alterações por comandos slash de modelo/harness/permissões valem no próximo turno.
- **Cofre indisponível:** desbloqueie/configure o serviço de credenciais ou escolha explicitamente `--credential-file`. Linux requer uma sessão Secret Service. Não enfraqueça permissões de arquivo.
- **Erros HTTP/protocolo do provedor:** verifique conta, endpoint, modelo e protocolo; corpos de erro são omitidos intencionalmente. Desative `stream_usage` em provedores compatíveis que não aceitam esse campo. Modelos somente de texto devem usar `tools:false` e não servem para fluxos com captura de tela.
- **Falha ao preparar contexto:** originais continuam salvos. Revise/exporte a sessão e inicie tarefa menor se o turno exceder o orçamento. Resumos inválidos são rejeitados.
- **Navegador indisponível:** execute `--setup-browser`; confira Node/npm e bibliotecas do Chromium. Downloads exigem rede. As origens permitidas precisam cobrir subrecursos necessários.
- **Desktop indisponível:** confira `/capabilities`, sessão DISPLAY/SO e permissões concedidas. X11 exige xdotool/ImageMagick/xclip. Windows exige desktop interativo e execução do script PowerShell segundo a política vigente. macOS exige Acessibilidade/Gravação de Tela.
- **Cancelamento:** Esc/Ctrl+C cancela ação/worker. Inspecione o alvo depois; cliques, envios, textos e efeitos externos já concluídos não podem ser desfeitos pelo cancelamento.

## Backup

Para exportar uma sessão ociosa: `ovc --export ID > session.json`. Para backup restaurável completo, pare o runtime e copie o diretório inteiro da aplicação, incluindo arquivos auxiliares do SQLite se existirem. Restaure primeiro em um `OPENVIBECODING_HOME` separado e inspecione. Nunca copie apenas um arquivo de banco SQLite enquanto estiver em uso.
