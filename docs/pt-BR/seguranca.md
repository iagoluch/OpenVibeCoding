# Segurança e garantias de execução

[Português](seguranca.md) · [English](../security.md) · [Español](../es/seguridad.md) · [简体中文](../zh-CN/security.md)

## Ferramentas de arquivos do workspace

`os.Root` dificulta escapes do diretório raiz. Caminhos absolutos, caminhos com `..` e componentes de links simbólicos são recusados. Locais conhecidos de segredos (`.env*`, `.ssh`, `.aws`, nomes de credenciais e extensões de chaves privadas), diretórios gerados e correspondências de `.gitignore` são excluídos das ferramentas comuns de leitura/pesquisa/listagem. Leituras aceitam arquivos de texto UTF-8 regulares de até 128 KiB. Isso é uma política de bloqueio/ignore, não um detector universal de segredos. Exclusões herdadas são conservadoras: uma exceção em diretório interno não reabre um caminho excluído por um ancestral.

Uma proposta não grava nada. Substituições e exclusões mostram diff unificado; movimentações informam origem/destino e exibem o conteúdo afetado. A aprovação vale para uma operação. O digest original do arquivo é verificado novamente antes da alteração. Substituições usam arquivos temporários exclusivos, fsync e rename; movimentos criam o destino exclusivamente antes de remover a origem. Se a origem não puder ser removida, o sistema relata o resultado parcial, sem fingir sucesso. Diretórios-pai podem ser criados dentro da raiz; um movimento nunca sobrescreve um destino não relacionado.

Essa fronteira protege contra caminhos fornecidos pelo modelo, não contra processos locais hostis simultâneos, hard links ou sistemas de arquivos montados e controlados pelo mesmo usuário do SO. A verificação de digest não é um compare-and-swap do sistema de arquivos inteiro; evite edições externas concorrentes durante a aplicação. A exclusão mútua do runtime por workspace evita que dois turnos do agente alterem o mesmo workspace ao mesmo tempo.

## Isolamento de processos no Linux

A execução padrão de shell usa Bubblewrap e falha de forma fechada se ele estiver ausente ou for bloqueado pela política do host. O sandbox expõe `/usr` e diretórios necessários do runtime do sistema somente leitura, um `/tmp` efêmero, dispositivos limitados e o projeto selecionado em `/workspace`. O diretório home do host e `/proc` não ficam disponíveis. Arquivos protegidos dentro do projeto são mascarados; dependências continuam disponíveis para builds. São solicitados namespaces separados, nova sessão e encerramento junto ao processo pai. Não há promessa de mitigar vulnerabilidades do kernel ou canais laterais.

A rede é isolada por padrão. Uma permissão explícita de rede compartilha a rede do host e monta arquivos de resolução/certificados necessários; a aprovação identifica que a execução tem rede. A permissão de instalação libera a ferramenta de instalação de pacotes. Código shell geral pode executar ações equivalentes dentro dos limites de filesystem/rede permitidos; categorias de ferramenta, sozinhas, não impõem política semântica a qualquer programa.

Não existe sandbox de processos integrado no macOS ou Windows. Solicitações que exigem confinamento falham de forma fechada nessas plataformas. `--allow-unsandboxed-shell` habilita deliberadamente a execução com privilégios do usuário do host, em qualquer plataforma, mediante aprovação por comando e aviso explícito. Ela pode acessar caminhos do host, rede e aplicativos instalados. Não use modo host quando for necessária uma barreira forte do SO. Privilégios não são elevados automaticamente.

Processos filhos recebem uma pequena lista permitida de variáveis de ambiente, sem credenciais do provedor. Mesmo assim, modo host pode ler credenciais em arquivos locais. Comandos têm limite de 60 segundos e saídas stdout/stderr limitadas. No Unix, cancelar shell no host encerra o grupo de processos; workers de automação também usam grupos. No Windows, o cancelamento usa taskkill na árvore do processo. Processos destacados intencionalmente e ações concluídas podem não ser reversíveis ou encerráveis. Isso é cancelamento, não isolamento do SO no macOS/Windows.

## Runtime

Um token aleatório de 256 bits autentica a comunicação IPC por loopback. O arquivo de token é privado, nunca é exibido e os clientes rejeitam endereços ou redirecionamentos que não sejam loopback. Requisições são limitadas, métodos/rotas são validados e solicitações do navegador cross-origin são rejeitadas. Travas do SO evitam daemons duplicados e são liberadas após falhas. Metadados só são substituídos depois da obtenção da trava. Incompatibilidades de protocolo falham em vez de reutilizar estado incompatível.

IDs de aprovação são aleatórios, de uso único e ligados à identidade do cliente que originou a solicitação. Só o runtime pode alterar o armazenamento; gravações diretas de mensagens/resumos/uso pelo cliente são rejeitadas. Sessões ativas não podem ser excluídas, recuperadas, renomeadas ou exportadas. Turnos são serializados por sessão e workspace. Desconectar um cliente cancela sua própria solicitação; outros clientes continuam. Parar o runtime é recusado com tarefas ativas; encerramento do SO as cancela antes de fechar o armazenamento.

O usuário local autenticado configura permissões e endpoints. Autenticação não torna confiável um comando gerado por IA: mutações, comandos shell e ações de automação ainda exigem aprovação, e o modo `inspect` os recusa.

## Segredos, auditoria e contexto

Credenciais vêm de variáveis de ambiente, cofre do SO ou fallback em arquivo selecionado explicitamente. Arquivos fallback Unix com permissões para grupo/outros são recusados. Novos arquivos usam 0600 e diretórios 0700; no Windows valem ACLs do perfil. Segredos não são incluídos no JSON de configuração. Valores conhecidos de credenciais e controles de terminal são ocultados antes de exibição/persistência. Segredos arbitrários de aplicativos não podem ser detectados com confiabilidade; não peça às ferramentas para imprimi-los. Sessões não são criptografadas em repouso.

O SQLite registra propostas e aprovações antes da execução. Falha de persistência interrompe a operação. Início/fim de tarefas e mudanças de configuração são auditados. Uma falha entre mutação e registro durável gera recuperação com resultado incerto, sem repetir automaticamente a ação. Logs do runtime contêm metadados operacionais, não corpos de requisição ou imagens.

O histórico original é a fonte autoritativa. Resumos estruturados preservam IDs de origem e não concedem permissões. `recall` recupera originais em trechos limitados. Capturas são transitórias: só um marcador fica armazenado, e a imagem é enviada ao modelo configurado durante a tarefa atual. O usuário deve autorizar a captura, escolher um modelo com suporte a imagens e considerar o que está visível na tela.

## Navegador e desktop nativo

Navegador e desktop são ferramentas separadas e exigem permissões/aprovações independentes. Clipboard exige uma permissão adicional. O worker Playwright usa perfil novo e política exata de origem para páginas, bloqueia downloads, service workers e WebSockets, limita etapas/tempo/saída e reporta falhas observadas. A política de roteamento não equivale a isolamento de rede do SO.

Entradas nativas atuam no desktop visível do usuário. Abrir/focar aplicativos usa uma lista permitida; gestos globais de mouse/teclado não ficam confinados a um processo. O usuário revisa o alvo e a ação antes de aprovar. Permissões de Acessibilidade/Gravação de Tela, áreas bloqueadas/UAC e restrições do compositor são respeitadas. Wayland não é suportado pelo backend X11 integrado do Linux. Adaptadores externos são configuração confiável do usuário e devem impor sua própria política de SO. Consulte o [contrato de automação](automacao.md).
