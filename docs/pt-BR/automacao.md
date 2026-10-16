# Contrato de automação

[Português](automacao.md) · [English](../automation.md) · [Español](../es/automatizacion.md) · [简体中文](../zh-CN/automation.md)

Navegador e desktop nativo são ferramentas e permissões distintas. Permitir shell não concede nenhuma delas. O harness `inspect` rejeita ambas mesmo quando as permissões estão habilitadas.

## Solicitação de navegador

```json
{
  "origins": ["http://127.0.0.1:3000"],
  "steps": [
    {"action":"open","url":"http://127.0.0.1:3000"},
    {"action":"fill","selector":"#name","text":"Usuário sintético"},
    {"action":"click","selector":"button[type=submit]"},
    {"action":"expect_text","selector":"#result","text":"Salvo"},
    {"action":"screenshot"}
  ]
}
```

O modelo passa o objeto serializado na string `workflow` da ferramenta de navegador. A interface exibe todo o fluxo antes da aprovação. São permitidas no máximo 30 etapas, cada uma com timeout de 10 segundos para localizadores e prazo total de 2 minutos. Origens precisam corresponder exatamente a esquema/host/porta; rotas fora delas são negadas, inclusive redirecionamentos e subrecursos. Isso é uma política de roteamento do navegador, não um sandbox de rede do SO. Use sites de teste confiáveis; não se afirma que toda atividade interna do navegador fora das requisições roteadas esteja isolada. Falhas retornam resultado explicitamente negativo, nunca uma asserção inventada como aprovada.

Uma captura PNG fica em memória transitória. O agente remove a imagem da saída persistida da ferramenta, envia as imagens mais recentes a um modelo com suporte visual e mantém apenas um marcador na sessão. Ao retomar, é necessária nova captura. Um endpoint/modelo sem suporte a imagens pode rejeitar esse tipo de solicitação.

## Solicitação nativa

```json
{"action":"click","x":400,"y":220,"button":1}
```

O modelo passa o objeto serializado como string `action` da ferramenta de desktop. Mouse, teclado, clipboard e capturas usam APIs/ferramentas da plataforma, não a ferramenta shell. Cada ação tem prazo de 15 segundos. O usuário pode parar com Esc/Ctrl+C, cancelando o worker ativo. O cancelamento não desfaz entradas já concluídas. Ações bloqueadas pelo SO ou indisponíveis retornam erros, sem conceder privilégios adicionais.

No Linux, nomes de tecla usam a sintaxe do xdotool (`ctrl+l`, `Return`); no Windows, a sintaxe SendKeys do Windows Forms; no macOS, códigos de tecla virtual de 0 a 127. Fechar janela no Linux exige ID numérico; no Windows/macOS, o alvo é a janela em primeiro plano. Abrir/focar exige uma entrada exata na lista permitida configurada. Ações globais de ponteiro/teclado afetam o desktop inteiro e são aprovadas individualmente.

No Unix, grupos de subprocessos de automação em andamento são encerrados ao cancelar; aplicativos destacados intencionalmente permanecem como estado do usuário. No Windows, o cancelamento usa taskkill na árvore do processo. Isso não reverte ações concluídas nem garante encerrar processos deliberadamente destacados. A ferramenta não interage com prompts de privilégio do SO nem concede permissões de Acessibilidade por conta própria.

## Backend de desktop externo

Configure um executável local confiável:

```json
{"automation":{"desktop":false,"desktop_backend":["/caminho/absoluto/para/meu-adaptador"],"applications":["meu-app"]}}
```

Depois da permissão de desktop e aprovação da ação, o executável recebe exatamente um objeto JSON pela entrada padrão. Deve devolver resultados observados em stdout, sair com código diferente de zero em caso de falha, respeitar cancelamento e impor políticas adicionais de SO/aplicativo. A saída é limitada a 8 MiB. Para devolver captura, use `{"image":"BASE64_PNG","mime":"image/png"}`; a imagem recebe o mesmo tratamento transitório. Nunca emita credenciais ou conteúdo privado da área de transferência, salvo se exigido especificamente pela ação aprovada. Argumentos adicionais são configuração fixa do usuário, nunca texto de comando gerado pelo modelo.

A interface permite integrações específicas de plataforma — por exemplo UI Automation, frameworks de acessibilidade do macOS ou ferramentas Wayland autorizadas pelo compositor — sem alterar o núcleo do agente. A existência do executável é apenas uma verificação preliminar; cada ação ainda verifica seus resultados e permissões.
