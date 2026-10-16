// OpenVibeCoding web client: one page, vanilla ES modules, no build step and
// no remote code. The runtime keeps all authority: this file only starts
// turns, renders the stream, answers approvals and reads or mutates storage
// through /store.
//
// A aparência replica o TUI do OpenCode (docs/research/opencode-tui-visual-contract.md):
// a moldura ┃ da mensagem do usuário, a trilha de 42 colunas, a caixa de entrada
// com base em ▀ e os rodapés de 1 linha. O idioma continua pt-BR e os glifos vêm
// do contrato — nada aqui monta HTML.
//
// Rendering rule: every value that comes from the runtime, the model or the
// user's files is inserted with textContent/createElement. Nothing here parses
// markup, evaluates code or builds HTML strings.

import { announce, basename, byId, clear, clock, copyText, count, el, firstLine, highlight, shorten, toast, when } from "./dom.js";
import { ApiError, approve, explain, health, identify, store, uiInfo, uiSelect, uiSetup } from "./api.js";
import { runTurn } from "./stream.js";

// ---------- identity ----------

// A stable, non-secret id for this browser profile. The runtime correlates an
// approval with the client that started the turn, and this is that id.
function clientIdentity() {
  const key = "ovc.client";
  let id = "";
  try {
    id = localStorage.getItem(key) || "";
  } catch {
    id = "";
  }
  if (!id) {
    id = crypto.randomUUID();
    try {
      localStorage.setItem(key, id);
    } catch {
      // Private browsing: the id lives for this page only.
    }
  }
  return id;
}

identify(clientIdentity());

// ---------- preferences ----------

const PREF_KEY = "ovc.prefs";
const DRAFT_PREFIX = "ovc.draft.";
const PREFERENCES = {
  density: "comfortable",
  scale: "medium",
  motion: "system",
  sendOnEnter: true,
  notify: false,
};
const prefs = { ...PREFERENCES, ...readPrefs() };

function readPrefs() {
  try {
    return JSON.parse(localStorage.getItem(PREF_KEY) || "{}") || {};
  } catch {
    return {};
  }
}

function savePrefs() {
  try {
    localStorage.setItem(PREF_KEY, JSON.stringify(prefs));
  } catch {
    // Preferences are a convenience; a failed write must not break the page.
  }
}

function applyPrefs() {
  const root = document.documentElement;
  // O clone é o tema dracula/dark (§0): não existe paleta clara verificada, então
  // data-theme fica fixo e o seletor de tema saiu dos Ajustes.
  root.dataset.theme = "dark";
  root.dataset.density = prefs.density;
  root.dataset.scale = prefs.scale;
  root.dataset.motion = prefs.motion;
}

// ---------- state ----------

const state = {
  info: null,
  protocol: null,
  activeTasks: 0,
  sessions: [],
  session: null,
  running: null,
  queue: [],
  usage: { input: 0, output: 0, cost: 0, hasCost: false },
  context: 0,
  pending: [],
  search: { term: "", hits: [], current: -1, raw: new Map() },
  currentAssistant: null,
  following: true,
  lastPrompt: "",
};

const $ = byId;
const MAX_RENDERED_ENTRIES = 300;
const MAX_TOOL_OUTPUT = 20000;

// ---------- helpers ----------

function action(label, className, handler, attributes = {}) {
  const button = el("button", { class: className, type: "button", ...attributes }, label);
  button.addEventListener("click", handler);
  return button;
}

function view(name) {
  for (const id of ["view-gate", "view-onboarding", "view-chat"]) {
    const node = $(id);
    if (node) node.hidden = id !== `view-${name}`;
  }
  if (name !== "chat") closeSidebar();
}

// O status é derivado, nunca forçado pelo que acabou de ser desenhado: o painel
// só diz "Trabalhando" quando há turno nesta aba ou tarefa viva no serviço.
function setStatus(text, kind = "idle") {
  const chip = $("status-chip");
  chip.textContent = text;
  const pill = $("status-pill");
  pill.className = `pill pill-${kind}`;
  pill.dataset.kind = kind;
  setIndicator(kind);
  paintRailDot(kind);
}

const SPINNER = ["⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"]; // §6, 80 ms por quadro
let spinnerHandle = 0;

function motionReduced() {
  if (prefs.motion === "reduced") return true;
  if (prefs.motion !== "system" || !window.matchMedia) return false;
  return window.matchMedia("(prefers-reduced-motion: reduce)").matches;
}

/** Indicadores vivos: o do status e o da sessão aberta na trilha. */
function liveDots() {
  const dots = [];
  const pill = $("status-dot");
  if (pill) dots.push(pill);
  const rail = document.querySelector('.rail-item[aria-current="true"] .rail-dot');
  if (rail) dots.push(rail);
  return dots;
}

function stopIndicator() {
  if (spinnerHandle && typeof cancelAnimationFrame === "function") cancelAnimationFrame(spinnerHandle);
  spinnerHandle = 0;
}

/** Único movimento estrutural do clone (§10.4); sem rAF (ou reduzido) fica no 1º quadro. */
function startIndicator() {
  const paint = (glyph) => {
    for (const node of liveDots()) node.textContent = glyph;
  };
  if (motionReduced() || typeof requestAnimationFrame !== "function") {
    paint(SPINNER[0]);
    return;
  }
  const tick = (now) => {
    paint(SPINNER[Math.floor(now / 80) % SPINNER.length]);
    spinnerHandle = requestAnimationFrame(tick);
  };
  spinnerHandle = requestAnimationFrame(tick);
}

function setIndicator(kind) {
  stopIndicator();
  if (kind === "working") {
    startIndicator();
    return;
  }
  const dot = $("status-dot");
  dot.textContent = kind === "waiting" ? "!" : "▪";
}

/** Marca de estado na trilha: `!` precisa de decisão, `●` é a sessão aberta. */
function paintRailDot(kind) {
  const dot = document.querySelector('.rail-item[aria-current="true"] .rail-dot');
  if (dot) dot.className = kind === "idle" ? "rail-dot" : "rail-dot rail-dot-live";
}

function renderStatus() {
  const waiting = pendingApprovals().length > 0;
  const working = !waiting && Boolean(state.running || state.activeTasks > 0);
  const kind = waiting ? "waiting" : working ? "working" : "idle";
  setStatus(waiting ? "Aguardando você" : working ? "Trabalhando" : "Pronta", kind);
  const hint = $("composer-hint");
  const usage = $("composer-usage");
  if (hint) hint.hidden = working;
  if (usage) usage.hidden = !working;
  // A linha do composer segue o mesmo estado: parada esperando a sua decisão, ela
  // diz o que o agente espera em vez de repetir o último passo já concluído.
  if (waiting) composerStatus("Aguardando sua decisão para continuar.");
  else if (!working) composerStatus("");
}

function composerStatus(text) {
  $("composer-status").textContent = text || "";
}

function trashNode(node) {
  node.remove();
}

/** Screens that show a turn: the text is opaque, so it goes in as text. */
function truncate(text, limit) {
  const value = String(text || "");
  if (value.length <= limit) return { text: value, truncated: false };
  return { text: value.slice(0, limit), truncated: true };
}

const TITLE_COLUMNS = 40; // §6/§10.6: título de sessão cortado em 40 colunas com …

/** Corta em N colunas mantendo o … do TUI. */
function clip(text, columns = TITLE_COLUMNS) {
  const value = String(text || "");
  return value.length > columns ? `${value.slice(0, columns - 1)}…` : value;
}

/** Splits "tool → result" exactly as the runtime writes it. */
function splitToolEvent(text) {
  const raw = String(text || "");
  const at = raw.indexOf(" → ");
  if (at < 0) return { name: "", result: raw };
  return { name: raw.slice(0, at), result: raw.slice(at + 3) };
}

function describeTool(name) {
  const table = {
    list_files: "Listou os arquivos do projeto",
    read_file: "Leu um arquivo",
    search: "Procurou no projeto",
    git: "Inspecionou o Git",
    capabilities: "Conferiu o que esta máquina oferece",
    recall: "Recuperou uma mensagem anterior",
    write_file: "Alterou um arquivo",
    delete_file: "Apagou um arquivo",
    move_file: "Moveu um arquivo",
    shell: "Rodou um comando",
    install_packages: "Instalou dependências",
    browser: "Usou o navegador automatizado",
    desktop: "Agiu na sua área de trabalho",
  };
  return { label: table[name] || `Usou a ferramenta ${name || "desconhecida"}` };
}

/** Turns a raw tool result into a subject, a hint and the full evidence. */
function splitToolResult(name, result) {
  const text = String(result || "");
  const applied = /^Applied reviewed replacement: (.+)$/m.exec(text);
  if (applied) return { subject: applied[1].trim(), hint: "alteração aplicada", body: text };
  if (/^Rejected by user/.test(text)) return { subject: "operação recusada por você", hint: "", body: text };
  if (text.trim() === "No change") return { subject: "nada mudou", hint: "", body: text };
  const first = firstLine(text, 90);
  return { subject: first, hint: name === "shell" || name === "install_packages" ? "ver a saída" : "ver o resultado", body: text };
}

// ---------- transcript ----------

function transcript() {
  return $("transcript");
}

function atBottom(node, slack = 90) {
  return node.scrollHeight - node.scrollTop - node.clientHeight <= slack;
}

function follow() {
  const box = transcript();
  if (state.following) box.scrollTop = box.scrollHeight;
}

function registerSearchable(turn, text) {
  state.search.raw.set(turn, text);
}

function makeTurn(roleLabel, className) {
  const node = el("li", { class: `turn ${className}` }, el("p", { class: "turn-role" }, roleLabel));
  const body = el("div", { class: "turn-body" });
  node.append(body);
  return { node, body };
}

function addUserTurn(text) {
  const item = makeTurn("Você", "turn-user");
  item.body.textContent = text;
  registerSearchable(item.node, text);
  transcript().append(item.node);
  follow();
  return item;
}

function addAssistantTurn(text = "") {
  const item = makeTurn("Agente", "turn-assistant");
  item.body.textContent = text;
  if (text) registerSearchable(item.node, text);
  transcript().append(item.node);
  follow();
  return item;
}

function addNote(text) {
  const item = makeTurn("Nota", "turn-note");
  item.body.textContent = text;
  registerSearchable(item.node, text);
  transcript().append(item.node);
  follow();
}

function addError(error, context) {
  const info = explain(error, context);
  const node = el("li", { class: "turn turn-error" }, el("p", { class: "turn-role" }, "Não deu certo"));
  const body = el("div", { class: "turn-body" }, el("p", {}, el("strong", {}, info.title)));
  if (info.hint) body.append(el("p", {}, info.hint));
  const actions = el("div", { class: "buttons" });
  if (context === "run" && state.lastPrompt) {
    actions.append(action("Tentar de novo", "secondary-button", () => submit(state.lastPrompt)));
  }
  if (context === "run" || context === "setup" || context === "store") {
    actions.append(action("Trocar de modelo", "secondary-button", openModelDialog));
  }
  if (info.detail) {
    const details = el("details", { class: "evidence" }, el("summary", {}, "Detalhes técnicos"));
    details.append(el("pre", { class: "tool-result" }, info.detail));
    body.append(details);
  }
  body.append(actions);
  node.append(body);
  registerSearchable(node, `${info.title} ${info.hint || ""}`);
  transcript().append(node);
  follow();
  announce(info.title);
}

/** Resultado de ferramenta: linha compacta com o ícone de 2 células do TUI (§7.4). */
function addToolTurn(text) {
  const { name, result } = splitToolEvent(text);
  const { label } = describeTool(name);
  const values = splitToolResult(name, result);
  const failed = /^(error|erro|failed|falha|rejected by user|operação recusada)/i.test(String(values.body || "").trim());
  const details = el("details");
  const summary = el("summary", {},
    el("span", { class: failed ? "tool-icon tool-icon-error" : "tool-icon", "aria-hidden": "true" }, failed ? "✗" : "✓"),
    el("strong", {}, label));
  if (values.subject) summary.append(el("span", { class: "mono" }, values.subject));
  if (values.hint) summary.append(el("span", { class: "tool-hint" }, values.hint));
  details.append(summary);

  const shown = truncate(values.body, MAX_TOOL_OUTPUT);
  details.append(el("pre", { class: "tool-result" }, shown.text));
  if (shown.truncated) {
    details.append(el("p", { class: "muted small" }, "Trecho longo: o painel mostra o começo. O conteúdo completo continua na pasta do projeto."));
  }
  const node = el("li", { class: "turn tool-line" }, details);
  transcript().append(node);
  registerSearchable(node, `${label} ${values.subject}`);
  follow();
}

// ---------- approvals ----------

const APPROVAL_KINDS = [
  { match: /^WRITE\s+(.+)$/m, kind: "write" },
  { match: /^DELETE\s+(.+)$/m, kind: "delete" },
  { match: /^MOVE\s+(.+)$/m, kind: "move" },
  { match: /^UNSANDBOXED HOST COMMAND/m, kind: "host-command" },
  { match: /^CONFINED COMMAND/m, kind: "command" },
  { match: /^BROWSER WORKFLOW/m, kind: "browser" },
  { match: /^DESKTOP ACTION/m, kind: "desktop" },
];

const APPROVAL_COPY = {
  write: {
    badge: "Alterar arquivo",
    question: (d) => `O agente quer gravar o conteúdo revisado em ${d.target || "um arquivo"}.`,
    risk: "Só este arquivo muda. A alteração fica registrada e pode ser desfeita em Detalhes da sessão.",
    diff: true,
  },
  delete: {
    badge: "Apagar arquivo",
    question: (d) => `O agente quer apagar ${d.target || "um arquivo"}.`,
    risk: "Depois de apagar dá para desfazer em Detalhes da sessão, desde que nada tenha mudado o arquivo nesse meio-tempo.",
    diff: true,
  },
  move: {
    badge: "Mover arquivo",
    question: (d) => `O agente quer mover ${d.target || "um arquivo"}.`,
    risk: "O arquivo muda de lugar sem sobrescrever outro. Dá para desfazer depois.",
    diff: true,
  },
  command: {
    badge: "Rodar comando",
    question: () => "O agente quer rodar este comando dentro do isolamento do projeto.",
    risk: "Sem acesso à sua pasta pessoal e sem rede, salvo onde o projeto libera rede. O que o comando imprimir volta para a conversa.",
    diff: false,
  },
  "host-command": {
    badge: "Comando com acesso à máquina",
    question: () => "O agente quer rodar este comando direto na sua máquina.",
    risk: "O comando alcança a rede e os seus arquivos. Leia linha por linha antes de permitir.",
    diff: false,
  },
  browser: {
    badge: "Usar o navegador",
    question: () => "O agente quer abrir páginas e interagir com elas por um navegador automatizado.",
    risk: "Confira os endereços e os passos. A sessão do navegador não usa as suas senhas.",
    diff: false,
  },
  desktop: {
    badge: "Agir na sua área de trabalho",
    question: () => "O agente quer mover o mouse, clicar ou digitar na sua área de trabalho.",
    risk: "Confira o alvo descrito abaixo: as ações acontecem na sua tela, de verdade.",
    diff: false,
  },
  other: {
    badge: "Operação que muda algo",
    question: () => "O agente parou e pediu a sua decisão para seguir.",
    risk: "Leia o texto exato abaixo — é o que será executado se você permitir.",
    diff: false,
  },
};

/** Reads the approval the runtime was handed by the agent. */
function describeApproval(text) {
  const raw = String(text || "");
  for (const { match, kind } of APPROVAL_KINDS) {
    const found = match.exec(raw);
    if (found) return { kind, target: found[1] ? found[1].trim() : "", raw };
  }
  return { kind: "other", target: "", raw };
}

/** Colours a unified diff without ever treating it as markup. */
function diffBlock(text) {
  const block = el("div", { class: "diff-block" });
  for (const line of String(text || "").split("\n")) {
    const className = line.startsWith("+") && !line.startsWith("+++") ? "line-add"
      : line.startsWith("-") && !line.startsWith("---") ? "line-del"
        : "line-ctx";
    block.append(el("span", { class: className }, `${line}\n`));
  }
  return block;
}

function addApprovalTurn(event) {
  const info = describeApproval(event.Text);
  const copy = APPROVAL_COPY[info.kind] || APPROVAL_COPY.other;
  const node = el("li", { class: "turn turn-approval", tabindex: "-1", "aria-label": "Aprovação pendente" });
  node.append(el("p", { class: "turn-role" },
    el("span", { class: "badge badge-warn" }, copy.badge), " precisa da sua decisão"));
  node.append(el("p", {}, copy.question(info)));

  const [head, ...rest] = info.raw.split("\n");
  const bodyText = rest.join("\n").trim();
  if (copy.diff && bodyText) node.append(diffBlock(bodyText));
  else node.append(el("pre", { class: "spec-block" }, bodyText || head));

  const evidence = el("details", { class: "evidence" },
    el("summary", {}, "Ver o texto exato enviado pelo agente"),
    el("pre", { class: "spec-block" }, info.raw));
  node.append(evidence);
  node.append(el("p", { class: "approval-note" }, copy.risk));

  const note = el("p", { class: "approval-note" });
  const allow = action("Permitir esta vez", "primary-button", () => decide(true));
  const deny = action("Recusar", "secondary-button", () => decide(false));
  node.append(el("div", { class: "approval-actions" }, allow, deny), note);

  const record = { id: event.Approval, node, allow, deny, note, decided: false, decide };
  state.pending.push(record);

  async function decide(allowed) {
    if (record.decided) return;
    record.decided = true;
    allow.disabled = true;
    deny.disabled = true;
    node.dataset.decided = "true";
    note.textContent = allowed
      ? "Permitido. O agente segue com a operação."
      : "Recusado. O agente recebe a recusa e escolhe outro caminho.";
    announce(allowed ? "Operação permitida." : "Operação recusada.");
    try {
      await approve(event.Approval, allowed);
      toast("ok", allowed ? "Operação permitida." : "Operação recusada.");
    } catch (error) {
      const explained = explain(error, "approve");
      note.textContent = `${explained.title}. ${explained.hint || ""}`.trim();
      note.classList.add("error-text");
    } finally {
      state.pending = state.pending.filter((item) => item !== record);
      renderStatus();
    }
  }

  transcript().append(node);
  follow();
  renderStatus();
  node.focus();
  announce(`O agente precisa da sua decisão: ${copy.badge}.`);
  notify("O agente precisa da sua decisão", `${copy.badge}${info.target ? ` · ${info.target}` : ""}`);
}

function pendingApprovals() {
  return state.pending.filter((record) => !record.decided);
}

function rejectPending() {
  const [record] = pendingApprovals();
  if (!record) return false;
  record.decide(false);
  return true;
}

// ---------- notifications ----------

function notify(title, body) {
  if (!prefs.notify || !document.hidden) return;
  if (typeof Notification === "undefined" || Notification.permission !== "granted") return;
  try {
    new Notification(title, { body });
  } catch {
    // Blocked constructor: the in-page status still shows what happened.
  }
}

// ---------- usage and counters ----------

function activeModel() {
  const info = state.info;
  if (!info) return null;
  const provider = (info.providers || []).find((item) => item.id === info.active.provider);
  if (!provider) return null;
  return (provider.models || []).find((item) => item.id === info.active.model) || (provider.models || [])[0] || null;
}

function renderCounters() {
  const usage = state.usage;
  $("usage-chip").textContent = usage.input || usage.output
    ? `tokens: ${count(usage.input)} entrada · ${count(usage.output)} saída`
    : "tokens: —";
  const model = activeModel();
  $("context-chip").textContent = state.context && model && model.context
    ? `contexto: ~${Math.min(99, Math.round((state.context / model.context) * 100))}% de ${count(model.context)}`
    : "contexto: —";
  const costChip = $("cost-chip");
  costChip.hidden = !usage.hasCost;
  if (usage.hasCost) costChip.textContent = `custo: US$ ${usage.cost.toFixed(4)}`;
  // §8.2: com trabalho vivo, o rodapé do composer troca os atalhos pelo uso.
  const live = $("composer-usage");
  if (live) {
    live.textContent = state.context
      ? `${count(state.context)} tokens${model && model.context ? ` (${Math.min(99, Math.round((state.context / model.context) * 100))}%)` : ""}`
      : "";
  }
}

/** Cabeçalho da sessão: título, pasta real e números do contexto (§6). */
function renderHead() {
  const session = state.session;
  $("chat-title").textContent = session ? session.Title || "Sem título" : "Sessão";
  $("chat-workspace").textContent = session ? `${basename(session.Workspace)} · ${session.Workspace}` : "";
  const place = $("composer-place");
  if (place) place.textContent = session ? shorten(session.Workspace, 44) : "";
  if (!session) {
    state.usage = { input: 0, output: 0, cost: 0, hasCost: false };
    state.context = 0;
  }
  renderCounters();
}

/** Linha de metadados do composer: agente · modelo provider · harness (§8.3). */
function composerMeta() {
  const active = state.info && state.info.active;
  const button = $("model-button");
  const provider = $("composer-provider");
  const variant = $("composer-variant");
  if (!active || !active.model) {
    button.textContent = "—";
    button.setAttribute("aria-label", "Escolher um modelo");
    provider.textContent = "";
    variant.textContent = "";
    variant.hidden = true;
    return;
  }
  button.textContent = active.model;
  button.setAttribute("aria-label", `Trocar de modelo (em uso: ${active.model})`);
  provider.textContent = active.provider || "";
  variant.textContent = active.harness ? `· ${active.harness}` : "";
  variant.hidden = !active.harness;
}

async function loadUsage() {
  if (!state.session) return;
  try {
    const rows = (await store({ Op: "usage", ID: state.session.ID })) || [];
    const usage = { input: 0, output: 0, cost: 0, hasCost: false };
    for (const row of rows) {
      usage.input += Number(row.Input) || 0;
      usage.output += Number(row.Output) || 0;
      if (typeof row.Cost === "number") {
        usage.cost += row.Cost;
        usage.hasCost = true;
      }
    }
    state.usage = usage;
    renderCounters();
  } catch {
    // Usage is informational; a failure must not break the session view.
  }
}

function addUsage(usage) {
  if (!usage) return;
  state.context = Number(usage.prompt_tokens) || state.context;
  state.usage.input += Number(usage.prompt_tokens) || 0;
  state.usage.output += Number(usage.completion_tokens) || 0;
  renderCounters();
}

// ---------- turns ----------

function turnBody() {
  const active = state.info.active;
  return {
    session: state.session.ID,
    provider: active.provider,
    model: active.model,
    harness: active.harness,
    workspace: state.session.Workspace,
    permissions: state.info.permissions,
    automation: state.info.automation,
  };
}

function setRunning(running) {
  state.running = running;
  $("stop-button").hidden = !running;
  $("send-button").textContent = running ? "Adicionar à fila" : "Enviar";
  composerStatus(running ? "Trabalhando… o texto aparece conforme o agente responde." : "");
  renderStatus();
  renderQueue();
}

function renderQueue() {
  const box = $("queue");
  const list = clear($("queue-list"));
  if (!state.queue.length) {
    box.hidden = true;
    return;
  }
  box.hidden = false;
  state.queue.forEach((prompt, index) => {
    list.append(el("li", {},
      el("p", {}, prompt),
      action("Remover", "link-button", () => {
        state.queue.splice(index, 1);
        renderQueue();
      }, { "aria-label": `Remover da fila: ${firstLine(prompt, 60)}` })));
  });
}

function submit(prompt) {
  const text = String(prompt || "").trim();
  if (!text) return;
  if (!state.session) {
    toast("info", "Crie ou abra uma sessão antes de enviar.");
    return;
  }
  if (state.running) {
    state.queue.push(text);
    renderQueue();
    announce("Mensagem adicionada à fila.");
    return;
  }
  runPrompt(text);
}

async function runPrompt(prompt) {
  state.lastPrompt = prompt;
  addUserTurn(prompt);
  saveDraft("");
  const controller = new AbortController();
  setRunning({ controller });
  state.currentAssistant = null;
  try {
    await runTurn({ ...turnBody(), prompt, signal: controller.signal }, handleEvent);
  } catch (error) {
    if (error && error.name === "AbortError") {
      addNote("Você interrompeu este pedido. O que já tinha sido feito continua na sessão.");
      announce("Pedido interrompido.");
    } else if (error instanceof ApiError && error.kind === "unpaired") {
      setRunning(null);
      showGate("unpaired");
      return;
    } else {
      addError(error, "run");
    }
  } finally {
    for (const record of pendingApprovals()) {
      record.decided = true;
      record.allow.disabled = true;
      record.deny.disabled = true;
      record.note.textContent = "O pedido terminou antes desta decisão; nada foi executado.";
    }
    state.pending = [];
    state.currentAssistant = null;
    setRunning(null);
    notify("O agente terminou", firstLine(prompt, 80));
    await refreshHealth();
    await refreshSessions();
    await loadUsage();
    if (state.queue.length) {
      const next = state.queue.shift();
      renderQueue();
      runPrompt(next);
    }
  }
}

// A D3 do QA: o motor emite um código estável (agent.Event.Code) e o painel pt-BR
// escolhe a frase aqui — nenhuma string em inglês chega ao composer.
const STATUS_COPY = {
  requesting: "Preparando o pedido ao modelo…",
  summarizing: "Resumindo a conversa antes de continuar…",
};

function statusLine(event) {
  return STATUS_COPY[event.Code] || "Trabalhando…";
}

function handleEvent(event) {
  switch (event.Kind) {
    case "delta": {
      if (!state.currentAssistant) state.currentAssistant = addAssistantTurn();
      state.currentAssistant.body.textContent += event.Text;
      follow();
      break;
    }
    case "response": {
      const item = state.currentAssistant || (event.Text ? addAssistantTurn() : null);
      if (item) {
        item.body.textContent = event.Text;
        registerSearchable(item.node, event.Text);
        if (event.Text) announce("Resposta recebida.");
      }
      state.currentAssistant = null;
      addUsage(event.Usage);
      follow();
      break;
    }
    case "tool":
      addToolTurn(event.Text);
      break;
    case "approval":
      addApprovalTurn(event);
      break;
    case "status":
      composerStatus(statusLine(event));
      break;
    case "done":
      if (event.Text) addError(new ApiError({ message: event.Text, status: 200 }), "run");
      break;
    default:
      break;
  }
  if (event.Kind === "done" || event.Kind === "response") updateSearch();
}

function stopTurn() {
  if (state.running) {
    state.running.controller.abort();
    composerStatus("Interrompendo…");
    return;
  }
  rejectPending();
}

// ---------- drafts ----------

function saveDraft(value) {
  if (!state.session) return;
  const key = DRAFT_PREFIX + state.session.ID;
  try {
    if (value) localStorage.setItem(key, value);
    else localStorage.removeItem(key);
  } catch {
    // Drafts are a convenience only.
  }
}

function restoreDraft() {
  const area = $("prompt");
  try {
    area.value = state.session ? localStorage.getItem(DRAFT_PREFIX + state.session.ID) || "" : "";
  } catch {
    area.value = "";
  }
}

// ---------- in-session search ----------

function applySearch(moveToHit = true) {
  const term = $("transcript-search").value.trim();
  state.search.term = term;
  state.search.hits = [];
  state.search.current = -1;
  for (const [node, raw] of state.search.raw) {
    const body = node.querySelector(".turn-body") || node;
    if (term && raw.toLowerCase().includes(term.toLowerCase())) {
      body.replaceChildren(highlight(raw, term));
      state.search.hits.push(node);
    } else {
      body.textContent = raw;
    }
  }
  const summary = $("search-count");
  summary.textContent = !term ? "" : state.search.hits.length ? `${state.search.hits.length} trecho(s)` : "nada encontrado";
  announce(summary.textContent);
  if (moveToHit && state.search.hits.length) focusHit(0);
}

/** New text arrived mid-turn: refresh the active search without moving the reader. */
function updateSearch() {
  const previous = state.search.current;
  if (!state.search.term) return;
  applySearch(false);
  if (state.search.hits.length && previous >= 0) {
    focusHit(Math.min(previous, state.search.hits.length - 1));
  }
}

function focusHit(index) {
  const hits = state.search.hits;
  if (!hits.length) return;
  state.search.current = (index + hits.length) % hits.length;
  for (const node of hits) node.classList.remove("turn-current");
  const node = hits[state.search.current];
  node.classList.add("turn-current");
  node.scrollIntoView({ block: "center" });
  $("search-count").textContent = `${state.search.current + 1} de ${hits.length}`;
}

// ---------- sessions ----------

/** Trilha de 42 colunas: cabeçalho por pasta e título cortado em 40 (§6). */
function renderSessions() {
  const list = clear($("session-list"));
  const filter = $("session-filter").value.trim().toLowerCase();
  const shown = state.sessions.filter((session) => !filter || `${session.Title} ${session.Workspace}`.toLowerCase().includes(filter));
  const groups = new Map();
  for (const session of shown) {
    const key = session.Workspace || "";
    if (!groups.has(key)) groups.set(key, []);
    groups.get(key).push(session);
  }
  for (const [workspace, items] of groups) {
    const group = el("li", { class: "rail-group" });
    group.append(el("p", { class: "rail-group-name", title: workspace }, basename(workspace) || "sem pasta"));
    const sub = el("ul", { class: "rail-items", role: "list" });
    for (const session of items) sub.append(el("li", {}, railItem(session)));
    group.append(sub);
    list.append(group);
  }
  const empty = $("session-empty");
  empty.hidden = shown.length > 0;
  empty.textContent = state.sessions.length
    ? "Nenhuma sessão combina com esse filtro."
    : "Nenhuma sessão ainda. Use Nova sessão para começar um trabalho.";
}

/** Item da trilha: `▪`/`●` de estado, título em 40 colunas e ✕ para fechar (§6). */
function railItem(session) {
  const current = Boolean(state.session && state.session.ID === session.ID);
  const title = session.Title || "Sem título";
  const button = el("button", { class: "rail-item", type: "button", title: `${title} · ${session.Workspace || "pasta não informada"}` });
  if (current) button.setAttribute("aria-current", "true");
  button.append(el("span", { class: "rail-dot", "aria-hidden": "true" }, current ? "●" : "▪"));
  button.append(el("span", { class: "rail-item-title" }, clip(title)));
  button.append(el("span", { class: "rail-close", "aria-hidden": "true" }, "✕"));
  button.addEventListener("click", (event) => {
    if (event.target && event.target.className === "rail-close") {
      closeSession();
      return;
    }
    openSession(session.ID);
  });
  return button;
}

/** ✕ fecha o painel; a sessão continua guardada (apagar é ação do diálogo). */
function closeSession() {
  if (state.running) {
    toast("info", "Há um pedido em andamento nesta aba: espere terminar ou use Parar.");
    return;
  }
  state.session = null;
  state.search.raw.clear();
  state.search.hits = [];
  clear(transcript());
  renderHead();
  renderWelcome();
  renderSessions();
  renderStatus();
  announce("Sessão fechada neste painel. Ela continua guardada na trilha.");
}

async function refreshSessions() {
  try {
    state.sessions = (await store({ Op: "list" })) || [];
  } catch (error) {
    if (error instanceof ApiError && error.kind === "unpaired") {
      showGate("unpaired");
      return;
    }
    state.sessions = [];
    addError(error, "store");
  }
  renderSessions();
}

async function openSession(id) {
  if (!id) return;
  let session;
  try {
    session = await store({ Op: "get", ID: id });
  } catch (error) {
    addError(error, "store");
    return;
  }
  if (!session) return;
  state.session = session;
  state.currentAssistant = null;
  state.search.raw.clear();
  state.search.hits = [];
  state.following = true;
  view("chat");
  renderSessions();
  renderHead();
  composerMeta();
  const box = clear(transcript());
  let entries = [];
  try {
    entries = (await store({ Op: "history", ID: id })) || [];
  } catch (error) {
    addError(error, "store");
  }
  renderHistory(entries);
  detectInterruption(entries);
  restoreDraft();
  await loadUsage();
  applySearch();
  // D1: o estado do painel vem do que existe (turno nesta aba ou tarefa viva no
  // serviço), nunca do que acabou de ser desenhado — abrir uma sessão com
  // ferramentas no histórico não pode dizer "Trabalhando".
  renderStatus();
  box.scrollTop = box.scrollHeight;
  announce(`Sessão ${session.Title || "sem título"} aberta.`);
}

function renderHistory(entries) {
  const shown = entries.length > MAX_RENDERED_ENTRIES ? entries.slice(-MAX_RENDERED_ENTRIES) : entries;
  if (entries.length > shown.length) {
    addNote(`Conversa longa: o painel mostra as ${shown.length} mensagens mais recentes. A sessão completa continua guardada.`);
  }
  if (!shown.length) {
    renderWelcome();
    return;
  }
  const names = new Map();
  for (const entry of shown) {
    const message = entry.Message || {};
    if (message.role === "user") {
      addUserTurn(message.content || "");
    } else if (message.role === "assistant") {
      for (const call of message.tool_calls || []) names.set(call.id, call.function ? call.function.name : "");
      if ((message.content || "").trim()) addAssistantTurn(message.content);
    } else if (message.role === "tool") {
      addToolTurn(`${names.get(message.tool_call_id) || "tool"} → ${message.content || ""}`);
    }
  }
}

/** A session whose last tool call has no result was interrupted mid-operation. */
function detectInterruption(entries) {
  const pending = new Set();
  for (const entry of entries) {
    const message = entry.Message || {};
    for (const call of message.tool_calls || []) pending.add(call.id);
    if (message.role === "tool") pending.delete(message.tool_call_id);
  }
  $("recover-banner").hidden = pending.size === 0;
}

const WORDMARK = {
  left: ["                   ", "█▀▀█ █▀▀█ █▀▀█ █▀▀▄", "█__█ █__█ █^^^ █__█", "▀▀▀▀ █▀▀▀ ▀▀▀▀ ▀~~▀"],
  right: ["             ▄     ", "█▀▀▀ █▀▀█ █▀▀█ █▀▀█", "█___ █__█ █__█ █^^^", "▀▀▀▀ ▀▀▀▀ ▀▀▀▀ ▀▀▀▀"],
};

/** Wordmark literal do TUI (§5.1): árvore muted à esquerda, nome em destaque à direita. */
function wordmark() {
  const box = el("div", { class: "wordmark", "aria-hidden": "true" });
  for (let line = 0; line < WORDMARK.left.length; line++) {
    const row = el("p");
    paintHalf(row, WORDMARK.left[line], "left");
    row.append(" ");
    paintHalf(row, WORDMARK.right[line], "right");
    box.append(row);
  }
  return box;
}

/** `_` = célula vazia com fundo de sombra; `^` = ▀ na cor; `~`/`,` = ▀/▄ na sombra. */
function paintHalf(row, art, half) {
  const cells = [];
  for (const char of art.padEnd(21, " ")) {
    if (char === "_") cells.push([" ", `wm-bg-${half}`]);
    else if (char === "^") cells.push(["▀", `wm-${half}`]);
    else if (char === "~") cells.push(["▀", `wm-fg-${half}`]);
    else if (char === ",") cells.push(["▄", `wm-fg-${half}`]);
    else cells.push([char, `wm-${half}`]);
  }
  let index = 0;
  while (index < cells.length) {
    let end = index;
    while (end + 1 < cells.length && cells[end + 1][1] === cells[index][1]) end++;
    let text = "";
    for (let i = index; i <= end; i++) text += cells[i][0];
    row.append(el("span", { class: cells[index][1] }, text));
    index = end + 1;
  }
}

function renderWelcome() {
  const box = transcript();
  const card = el("li", { class: "turn turn-note home" });
  card.append(el("p", { class: "home-blank-3" }));
  card.append(wordmark());
  card.append(el("p", { class: "home-blank" }));
  const body = el("div", { class: "turn-body home-body" });
  if (!state.session) {
    body.append(el("p", {}, "Escolha a pasta do projeto para a primeira sessão: é o limite que o agente respeita, e é de lá que ele lê e altera arquivos."));
    body.append(el("div", { class: "buttons" }, action("Nova sessão", "primary-button", () => openSessionDialog("create"))));
  } else {
    body.append(el("p", {}, "Descreva o que você quer neste projeto. O agente lê os arquivos, propõe mudanças e para para você aprovar."));
    const examples = [
      "Leia o README deste projeto e me explique, em poucas linhas, o que ele faz.",
      "Rode os testes do projeto e me diga o que passou e o que falhou.",
      "Liste os arquivos com mais linhas e diga por onde o código começa.",
    ];
    const list = el("div", { class: "home-actions" });
    for (const example of examples) {
      const item = el("button", { class: "home-action", type: "button" },
        el("span", { class: "home-mark", "aria-hidden": "true" }, "›"),
        el("span", {}, example));
      item.addEventListener("click", () => {
        const area = $("prompt");
        area.value = example;
        area.focus();
      });
      list.append(item);
    }
    body.append(list);
  }
  card.append(body);
  box.append(card);
}

async function createSession(workspace) {
  const folder = String(workspace || "").trim().replace(/^"(.*)"$/, "$1");
  if (!folder) {
    toast("error", "Informe a pasta do projeto.");
    return;
  }
  try {
    const id = await store({ Op: "new", Workspace: folder });
    await refreshSessions();
    await openSession(id);
    toast("ok", `Sessão criada em ${basename(folder)}.`);
  } catch (error) {
    const info = explain(error, "store");
    toast("error", `${info.title} ${info.hint}`.trim(), "Detalhes", () => addError(error, "store"));
  }
}

async function renameSession(id, title) {
  try {
    await store({ Op: "rename", ID: id, Text: title });
    await refreshSessions();
    if (state.session && state.session.ID === id) $("chat-title").textContent = title;
    toast("ok", "Nome atualizado.");
  } catch (error) {
    addError(error, "store");
  }
}

async function deleteSession(id) {
  try {
    await store({ Op: "delete", ID: id });
    if (state.session && state.session.ID === id) {
      state.session = null;
      clear(transcript());
      renderHead();
      composerMeta();
      renderWelcome();
      renderStatus();
    }
    await refreshSessions();
    toast("ok", "Sessão apagada.");
  } catch (error) {
    addError(error, "store");
  }
}

async function undoSession(id) {
  try {
    const summary = await store({ Op: "undo", ID: id });
    toast("ok", String(summary || "Alteração desfeita."));
    if (state.session && state.session.ID === id) await openSession(id);
  } catch (error) {
    const info = explain(error, "store");
    if (/nothing to undo/i.test(info.detail)) {
      toast("info", "Esta sessão não tem alteração para desfazer.");
      return;
    }
    toast("error", `${info.title} ${info.hint}`.trim(), "Detalhes", () => addError(error, "store"));
  }
}

async function recoverSession(id) {
  try {
    await store({ Op: "recover", ID: id });
    await openSession(id);
    toast("ok", "Sessão retomada: o agente já sabe o que ficou sem resultado.");
  } catch (error) {
    addError(error, "store");
  }
}

async function exportSession(id, title) {
  try {
    const value = await store({ Op: "export", ID: id });
    const bytes = Uint8Array.from(atob(value || ""), (char) => char.charCodeAt(0));
    const url = URL.createObjectURL(new Blob([bytes], { type: "application/json" }));
    const name = String(title || "sessao").replace(/[^\p{L}\p{N}._-]+/gu, "-");
    const link = el("a", { href: url, download: `${name}.json` });
    document.body.append(link);
    link.click();
    trashNode(link);
    window.setTimeout(() => URL.revokeObjectURL(url), 4000);
    toast("ok", "Cópia da conversa salva pelo navegador.");
  } catch (error) {
    addError(error, "store");
  }
}

// ---------- sidebar ----------

/** A busca do transcript mora atrás do `⌕` do cabeçalho, como no TUI (§6). */
function showSearch() {
  $("search-row").hidden = false;
  $("transcript-search-button").setAttribute("aria-expanded", "true");
  $("transcript-search").focus();
}

function toggleSearch() {
  if ($("search-row").hidden) {
    showSearch();
    return;
  }
  $("search-row").hidden = true;
  $("transcript-search-button").setAttribute("aria-expanded", "false");
  $("transcript-search").value = "";
  applySearch();
}

function openSidebar() {
  $("sidebar").hidden = false;
  $("menu-button").setAttribute("aria-expanded", "true");
}

function closeSidebar() {
  if (window.matchMedia("(max-width: 980px)").matches) $("sidebar").hidden = true;
  $("menu-button").setAttribute("aria-expanded", "false");
}

function toggleSidebar() {
  if ($("sidebar").hidden) openSidebar();
  else closeSidebar();
}

// ---------- gate ----------

function showGate(reason) {
  view("gate");
  $("gate-status").textContent = reason === "unpaired"
    ? "Esta aba ainda não está autorizada."
    : reason === "offline"
      ? "O serviço local não respondeu neste endereço."
      : "";
  $("gate-address").textContent = location.origin ? `Endereço deste painel: ${location.origin}` : "";
}

// ---------- onboarding ----------

const onboarding = { provider: "" };

function recommendedProvider(providers) {
  const running = providers.filter((provider) => provider.local && (provider.models || []).length);
  if (running.length) return running[0];
  return providers.find((provider) => provider.free) || providers[0] || null;
}

function renderOnboarding() {
  const providers = state.info.providers || [];
  const body = clear($("onboarding-body"));
  if (!providers.length) {
    body.append(el("p", {}, "O serviço não listou nenhum provedor. No terminal, rode ovc --setup para conectar um modelo."));
    return;
  }
  const recommended = recommendedProvider(providers);
  const pick = providers.find((provider) => provider.id === onboarding.provider) || recommended;
  onboarding.provider = pick.id;
  setStep("provider");

  const list = el("div", { class: "provider-list", role: "radiogroup", "aria-labelledby": "provider-legend" });
  for (const provider of providers) {
    const radio = el("input", { type: "radio", name: "provider", value: provider.id, id: `provider-${provider.id}` });
    radio.checked = provider.id === pick.id;
    radio.addEventListener("change", () => {
      onboarding.provider = provider.id;
      renderOnboarding();
    });
    const badges = el("span", { class: "badges-box" });
    if (recommended && provider.id === recommended.id) {
      badges.append(el("span", { class: "badge" }, provider.local ? "Já rodando aqui" : "Recomendado"));
    }
    if (provider.local) badges.append(el("span", { class: "badge badge-muted" }, "Roda na sua máquina"));
    if (provider.has_key) badges.append(el("span", { class: "badge badge-muted" }, "Chave guardada"));
    list.append(el("label", { class: "provider-option", for: `provider-${provider.id}` }, radio,
      el("span", {}, el("strong", {}, provider.title || provider.id), el("small", {}, provider.detail || ""), badges)));
  }
  const detail = el("div");
  body.append(el("fieldset", {}, el("legend", { id: "provider-legend" }, "1. Provedor"), list), detail);
  renderOnboardingDetail(detail, pick);
}

function setStep(name) {
  for (const item of document.querySelectorAll("#onboarding-steps li")) {
    if (item.dataset.step === name) item.setAttribute("aria-current", "step");
    else item.removeAttribute("aria-current");
  }
}

function renderOnboardingDetail(box, provider) {
  clear(box);
  const models = provider.models || [];
  const needsKey = !provider.local && !provider.has_key;
  const chosen = models.find((model) => model.id === onboarding.model) || models.find((model) => model.free) || models[0];
  onboarding.model = chosen ? chosen.id : "";
  setStep(needsKey ? "key" : "test");

  const panel = el("div", { class: "panel-inset" });
  panel.append(el("h3", {}, needsKey ? "2. Cole a chave" : "2. Confirme o acesso"));
  if (needsKey) {
    panel.append(el("p", {}, provider.detail || "Cole a chave de API deste provedor."));
    if (provider.key_url) {
      panel.append(el("p", {}, el("a", { href: provider.key_url, target: "_blank", rel: "noopener noreferrer" }, "Abrir a página da chave ↗")));
    }
  } else {
    panel.append(el("p", { class: "muted" }, provider.local
      ? "Este modelo roda na sua máquina: não precisa de chave nem de internet."
      : "Este provedor já tem uma chave guardada. Se quiser trocá-la, cole a nova abaixo."));
  }
  const keyField = el("p", { class: "field" });
  keyField.append(el("label", { class: "field-label", for: "onboarding-key" }, needsKey ? "Chave de API" : "Trocar a chave (opcional)"));
  keyField.append(el("input", { id: "onboarding-key", type: "password", autocomplete: "off", spellcheck: "false", "aria-describedby": "onboarding-key-help" }));
  keyField.append(el("span", { class: "field-help", id: "onboarding-key-help" }, "A chave vai para o cofre do seu sistema operacional: ela não é mostrada de volta nem enviada a outro lugar."));
  panel.append(keyField);

  const modelList = el("div", { class: "model-list", role: "radiogroup", "aria-labelledby": "models-legend" });
  if (!models.length) {
    modelList.append(el("p", { class: "muted" }, "Este provedor não publicou modelos aqui. No terminal, rode ovc --setup para configurar um."));
  }
  for (const model of models) {
    const radio = el("input", { type: "radio", name: "model", value: model.id, id: `model-${model.id}` });
    radio.checked = model.id === onboarding.model;
    radio.addEventListener("change", () => {
      onboarding.model = model.id;
    });
    modelList.append(el("label", { class: "model-row", for: `model-${model.id}` }, radio,
      el("span", {},
        el("strong", {}, model.id),
        el("span", { class: "model-meta" }, model.context ? `contexto ${count(model.context)} tokens` : "contexto não informado"),
        model.free ? el("span", { class: "badge" }, "grátis") : el("span", { class: "badge badge-muted" }, "pago"))));
  }
  panel.append(el("fieldset", {}, el("legend", { id: "models-legend" }, "3. Modelo"), modelList));

  const status = el("p", { class: "muted", role: "status" });
  const details = el("details", { class: "evidence" }, el("summary", {}, "Detalhes técnicos do último erro"));
  const detailsBody = el("pre", { class: "tool-result" });
  details.append(detailsBody);
  details.hidden = true;

  const force = action("Ativar sem testar", "secondary-button", async () => {
    force.disabled = true;
    status.textContent = "Ativando sem o teste: se o modelo não responder, troque de novo em Modelo.";
    try {
      await finishOnboarding(await uiSetup({ provider: provider.id, model: onboarding.model, key: keyValue(), force: true }));
    } catch (error) {
      showSetupFailure(error);
    } finally {
      force.disabled = false;
    }
  });
  force.hidden = true;

  const connect = action("Testar e ativar", "primary-button", async () => {
    const key = keyValue();
    if (needsKey && !key) {
      status.classList.add("error-text");
      status.textContent = "Cole a chave para testar. Sem chave não é possível falar com este provedor.";
      return;
    }
    connect.disabled = true;
    status.classList.remove("error-text");
    status.textContent = "Enviando um pedido de teste real ao provedor…";
    try {
      await finishOnboarding(await uiSetup({ provider: provider.id, model: onboarding.model, key }));
    } catch (error) {
      showSetupFailure(error);
      connect.disabled = false;
    }
  });

  function keyValue() {
    const input = $("onboarding-key");
    return input ? input.value.trim() : "";
  }

  function showSetupFailure(error) {
    const info = explain(error, "setup");
    status.classList.add("error-text");
    status.textContent = `${info.title}. ${info.hint}`;
    detailsBody.textContent = info.detail;
    details.hidden = !info.detail;
    force.hidden = false;
  }

  panel.append(el("div", { class: "buttons" }, connect, force), status, details);
  box.append(panel);
}

async function finishOnboarding(result) {
  toast("ok", `Modelo ativo: ${result.provider} / ${result.model}.`);
  announce("Modelo conectado.");
  state.info = await uiInfo();
  composerMeta();
  view("chat");
  await refreshSessions();
  if (state.sessions.length) await openSession(state.sessions[0].ID);
  else renderWelcome();
}

// ---------- dialogs ----------

function showDialog(id) {
  const dialog = $(id);
  if (dialog && typeof dialog.showModal === "function" && !dialog.open) dialog.showModal();
}

function closeDialog(id) {
  const dialog = $(id);
  if (dialog && dialog.open) dialog.close();
}

function wireDialogs() {
  for (const button of document.querySelectorAll("[data-close]")) {
    button.addEventListener("click", () => {
      const dialog = button.closest("dialog");
      if (dialog) dialog.close();
    });
  }
}

// ---------- model dialog ----------

const modelChoice = { provider: "" };

function openModelDialog() {
  renderModelDialog();
  showDialog("dialog-model");
}

function renderModelDialog() {
  const body = clear($("dialog-model-body"));
  const providers = state.info ? state.info.providers || [] : [];
  if (!providers.length) {
    body.append(el("p", {}, "Nenhum provedor disponível. No terminal, rode ovc --setup para conectar um modelo."));
    return;
  }
  const active = state.info.active;
  const chosen = providers.find((provider) => provider.id === modelChoice.provider)
    || providers.find((provider) => provider.id === active.provider) || providers[0];
  modelChoice.provider = chosen.id;
  const usable = chosen.has_key || chosen.local;

  body.append(el("p", { class: "muted" }, `Em uso agora: ${active.provider} / ${active.model}`));

  const providerList = el("div", { class: "provider-list", role: "radiogroup", "aria-label": "Provedor" });
  for (const provider of providers) {
    const radio = el("input", { type: "radio", name: "dialog-provider", value: provider.id, id: `dialog-provider-${provider.id}` });
    radio.checked = provider.id === chosen.id;
    radio.addEventListener("change", () => {
      modelChoice.provider = provider.id;
      renderModelDialog();
    });
    const badges = el("span", {});
    if (provider.id === active.provider) badges.append(el("span", { class: "badge" }, "em uso"));
    if (!provider.has_key && !provider.local) badges.append(el("span", { class: "badge badge-muted" }, "precisa de chave"));
    providerList.append(el("label", { class: "provider-option", for: `dialog-provider-${provider.id}` }, radio,
      el("span", {}, el("strong", {}, provider.title || provider.id), el("small", {}, provider.detail || ""), badges)));
  }
  body.append(el("fieldset", {}, el("legend", {}, "Provedor"), providerList));

  const models = chosen.models || [];
  const selected = models.find((model) => model.id === active.model && chosen.id === active.provider)
    || models.find((model) => model.free) || models[0];
  const pick = { id: selected ? selected.id : "" };
  const modelList = el("div", { class: "model-list", role: "radiogroup", "aria-label": "Modelo" });
  if (!models.length) {
    modelList.append(el("p", { class: "muted" }, "Este provedor não publicou modelos aqui. No terminal, rode ovc --setup."));
  }
  for (const model of models) {
    const radio = el("input", { type: "radio", name: "dialog-model", value: model.id, id: `dialog-model-${chosen.id}-${model.id}` });
    radio.checked = model.id === pick.id;
    radio.addEventListener("change", () => {
      pick.id = model.id;
    });
    modelList.append(el("label", { class: "model-row", for: `dialog-model-${chosen.id}-${model.id}` }, radio,
      el("span", {},
        el("strong", {}, model.id),
        el("span", { class: "model-meta" }, model.context ? `contexto ${count(model.context)} tokens` : "contexto não informado"),
        model.free ? el("span", { class: "badge" }, "grátis") : el("span", { class: "badge badge-muted" }, "pago"))));
  }
  body.append(el("fieldset", {}, el("legend", {}, "Modelo"), modelList));

  let keyInput = null;
  if (!usable) {
    const field = el("p", { class: "field" });
    field.append(el("label", { class: "field-label", for: "dialog-key" }, "Chave de API"));
    keyInput = el("input", { id: "dialog-key", type: "password", autocomplete: "off", spellcheck: "false", "aria-describedby": "dialog-key-help" });
    const help = el("span", { class: "field-help", id: "dialog-key-help" }, chosen.key_url
      ? `Fica no cofre do sistema operacional. Você gera uma chave em ${chosen.key_url}.`
      : "Fica no cofre do sistema operacional.");
    field.append(keyInput, help);
    body.append(field);
  }

  const status = el("p", { class: "muted", role: "status" });
  const buttons = el("div", { class: "buttons" });
  const needKey = !usable;

  const test = action(needKey ? "Testar e usar" : "Testar antes de trocar", needKey ? "primary-button" : "secondary-button", async () => {
    const key = keyInput ? keyInput.value.trim() : "";
    if (needKey && !key) {
      status.classList.add("error-text");
      status.textContent = "Cole a chave deste provedor para testar.";
      return;
    }
    status.classList.remove("error-text");
    status.textContent = "Testando com um pedido real…";
    try {
      await adoptModel(await uiSetup({ provider: chosen.id, model: pick.id, key }));
    } catch (error) {
      const info = explain(error, "setup");
      status.classList.add("error-text");
      status.textContent = `${info.title}. ${info.hint}`;
    }
  });
  if (usable) {
    buttons.append(action("Trocar agora", "primary-button", async () => {
      try {
        await adoptModel(await uiSelect({ provider: chosen.id, model: pick.id }));
      } catch (error) {
        const info = explain(error, "setup");
        status.classList.add("error-text");
        status.textContent = `${info.title}. ${info.hint}`;
      }
    }));
  }
  buttons.append(test);
  body.append(buttons, status);
}

async function adoptModel(result) {
  state.info = await uiInfo();
  composerMeta();
  renderCounters();
  renderSettings();
  renderStatusPanel();
  closeDialog("dialog-model");
  toast("ok", `Agora falando com ${result.provider} / ${result.model}.`);
  applySearch();
}

// ---------- settings ----------

function renderSettings() {
  $("setting-density").value = prefs.density;
  $("setting-scale").value = prefs.scale;
  $("setting-motion").value = prefs.motion;
  $("setting-send-enter").checked = prefs.sendOnEnter;
  const supported = typeof Notification !== "undefined";
  $("setting-notify").checked = prefs.notify && supported;
  $("setting-notify").disabled = !supported;
  $("setting-notify-help").textContent = supported
    ? `Permissão de avisos no navegador: ${Notification.permission === "granted" ? "concedida" : Notification.permission === "denied" ? "negada" : "ainda não pedida"}.`
    : "Este navegador não oferece avisos.";
}

function renderStatusPanel() {
  const panel = clear($("setting-status"));
  if (!state.info) return;
  const active = state.info.active;
  const permissions = state.info.permissions || {};
  const automation = state.info.automation || {};
  const rows = [
    ["Serviço", location.origin],
    ["Protocolo", state.protocol ? String(state.protocol) : "não confirmado"],
    ["Modelo ativo", `${active.provider} / ${active.model}${active.harness ? ` · ${active.harness}` : ""}`],
    ["Pasta proposta para novas sessões", state.info.workspace || "não informada"],
    ["Sessões guardadas", String(state.sessions.length)],
    ["Rede nos comandos confinados", permissions.network ? "liberada" : "negada"],
    ["Instalação de pacotes", permissions.install ? "permitida" : "negada"],
    ["Navegador automatizado", automation.browser ? "permitido" : "desligado"],
    ["Área de trabalho", automation.desktop ? "permitida" : "desligada"],
    ["Área de transferência", automation.clipboard ? "permitida" : "desligada"],
  ];
  for (const [label, value] of rows) {
    panel.append(el("p", {}, el("strong", {}, `${label}: `), el("span", { class: "mono" }, value)));
  }
  panel.append(el("p", { class: "muted small" }, "A conversa, os usos e a auditoria ficam no computador que roda o ovc. Só o texto enviado ao modelo sai daqui."));
}

// ---------- text and confirm dialogs ----------

let textDialogSubmit = null;

function askText({ title, label, value = "", help = "", submitLabel = "Salvar", onSubmit }) {
  $("dialog-text-title").textContent = title;
  $("dialog-text-label").textContent = label;
  $("dialog-text-input").value = value;
  $("dialog-text-help").textContent = help;
  $("dialog-text-submit").textContent = submitLabel;
  $("dialog-text-error").hidden = true;
  textDialogSubmit = onSubmit;
  showDialog("dialog-text");
  const input = $("dialog-text-input");
  input.focus();
  input.select();
}

function confirmDialog({ title, body, note = "", accept = "Confirmar" }) {
  $("dialog-confirm-title").textContent = title;
  $("dialog-confirm-body").textContent = body;
  $("dialog-confirm-note").textContent = note;
  $("dialog-confirm-accept").textContent = accept;
  return new Promise((resolve) => {
    const dialog = $("dialog-confirm");
    const acceptButton = $("dialog-confirm-accept");
    const cancelButton = $("dialog-confirm-cancel");
    const finish = (value) => {
      acceptButton.removeEventListener("click", onAccept);
      cancelButton.removeEventListener("click", onCancel);
      dialog.removeEventListener("close", onClose);
      if (dialog.open) dialog.close();
      resolve(value);
    };
    const onAccept = () => finish(true);
    const onCancel = () => finish(false);
    const onClose = () => finish(false);
    acceptButton.addEventListener("click", onAccept);
    cancelButton.addEventListener("click", onCancel);
    dialog.addEventListener("close", onClose);
    showDialog("dialog-confirm");
    cancelButton.focus();
  });
}

// ---------- session dialog ----------

function openSessionDialog(mode) {
  const body = clear($("dialog-session-body"));
  const creating = mode === "create" || !state.session;
  $("dialog-session-title").textContent = creating ? "Começar uma sessão" : "Detalhes da sessão";

  const folderField = el("p", { class: "field" });
  folderField.append(el("label", { class: "field-label", for: "dialog-folder" }, creating ? "Pasta do projeto" : "Nova pasta do projeto"));
  const input = el("input", {
    id: "dialog-folder",
    type: "text",
    spellcheck: "false",
    autocomplete: "off",
    value: (state.info && state.info.workspace) || "",
  });
  folderField.append(input, el("span", { class: "field-help" }, "Caminho completo da pasta onde o agente pode trabalhar. Ex.: C:\\projetos\\meu-app"));
  body.append(el("h3", {}, "Nova sessão"), folderField);
  const recent = (state.info && state.info.workspaces) || [];
  if (recent.length) {
    const chips = el("div", { class: "buttons" });
    for (const workspace of recent) {
      chips.append(action(basename(workspace), "secondary-button", () => {
        input.value = workspace;
      }, { title: workspace }));
    }
    body.append(el("p", { class: "field-label" }, "Pastas usadas recentemente"), chips);
  }
  body.append(el("div", { class: "buttons" }, action("Criar sessão", "primary-button", async () => {
    await createSession(input.value);
    closeDialog("dialog-session");
  })));

  const session = state.session;
  if (creating || !session) {
    showDialog("dialog-session");
    input.focus();
    return;
  }

  body.append(el("h3", {}, "Sessão aberta"));
  const usage = state.usage;
  const rows = [
    ["Nome", session.Title || "Sem título"],
    ["Pasta", session.Workspace],
    ["Identificador", session.ID],
    ["Criada", `${when(session.Created)} (${clock(session.Created)})`],
    ["Tokens", `${count(usage.input)} entrada · ${count(usage.output)} saída`],
  ];
  if (usage.hasCost) rows.push(["Custo estimado", `US$ ${usage.cost.toFixed(4)}`]);
  const table = el("div", { class: "panel-inset" });
  for (const [label, value] of rows) {
    table.append(el("p", {}, el("strong", {}, `${label}: `), el("span", { class: "mono" }, String(value))));
  }
  body.append(table);

  const buttons = el("div", { class: "buttons" });
  buttons.append(action("Renomear", "secondary-button", () => {
    closeDialog("dialog-session");
    askText({
      title: "Renomear a sessão",
      label: "Nome da sessão",
      value: session.Title || "",
      help: "Um nome curto ajuda a achar a sessão depois.",
      onSubmit: (value) => renameSession(session.ID, value),
    });
  }));
  buttons.append(action("Salvar uma cópia", "secondary-button", () => exportSession(session.ID, session.Title)));
  buttons.append(action("Desfazer a última alteração", "secondary-button", async () => {
    const ok = await confirmDialog({
      title: "Desfazer a última alteração?",
      body: "O painel devolve o último arquivo alterado por esta sessão ao conteúdo anterior.",
      note: "Só a alteração mais recente é revertida, e o arquivo precisa estar igual ao que foi aplicado.",
      accept: "Desfazer",
    });
    if (ok) await undoSession(session.ID);
  }));
  buttons.append(action("Retomar sessão interrompida", "secondary-button", () => recoverSession(session.ID)));
  buttons.append(action("Apagar sessão", "danger-button", async () => {
    const ok = await confirmDialog({
      title: "Apagar esta sessão?",
      body: "A conversa e a auditoria desta sessão saem do banco local.",
      note: "Os arquivos do projeto não são tocados.",
      accept: "Apagar",
    });
    if (ok) {
      closeDialog("dialog-session");
      await deleteSession(session.ID);
    }
  }));
  body.append(el("h3", {}, "Ações"), buttons);
  showDialog("dialog-session");
}

// ---------- command palette ----------

function paletteActions() {
  return [
    { label: "Nova sessão", description: "Escolher a pasta e começar um trabalho", run: () => openSessionDialog("create") },
    { label: "Trocar de modelo", description: "Provedor, modelo e teste de conexão", run: openModelDialog },
    { label: "Buscar nesta conversa", description: "Procurar um texto no histórico da sessão aberta", keys: "Ctrl+F", run: showSearch },
    { label: "Detalhes da sessão", description: "Pasta, tokens, desfazer, apagar, salvar cópia", run: () => openSessionDialog("details") },
    { label: "Parar o trabalho atual", description: "Encerra o pedido em andamento e libera a sessão", keys: "Esc", run: stopTurn, enabled: () => Boolean(state.running) || pendingApprovals().length > 0 },
    { label: "Ajustes", description: "Densidade, movimento, avisos e estado do serviço", run: () => {
      renderSettings();
      renderStatusPanel();
      showDialog("dialog-settings");
    } },
    { label: "Ajuda e atalhos", description: "Como o agente trabalha, aprovações e limites", run: () => showDialog("dialog-help") },
    { label: "Recarregar dados do serviço", description: "Reler modelo ativo, sessões e permissões", run: refreshAll },
  ];
}

function renderPalette(filter = "") {
  const list = clear($("palette-list"));
  const term = filter.trim().toLowerCase();
  const items = paletteActions().filter((item) => !term || `${item.label} ${item.description}`.toLowerCase().includes(term));
  $("palette-empty").hidden = items.length > 0;
  for (const item of items) {
    const allowed = item.enabled ? item.enabled() : true;
    const button = el("button", { class: "palette-item", type: "button" },
      el("span", {}, el("strong", {}, item.label), el("small", {}, item.description)),
      item.keys ? el("kbd", {}, item.keys) : el("span"));
    button.disabled = !allowed;
    button.addEventListener("click", () => {
      closeDialog("dialog-palette");
      item.run();
    });
    list.append(el("li", {}, button));
  }
}

function openPalette() {
  $("palette-input").value = "";
  renderPalette("");
  showDialog("dialog-palette");
  $("palette-input").focus();
}

// ---------- keyboard ----------

function typing(target) {
  if (!target || !target.tagName) return false;
  const tag = target.tagName;
  return tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT" || target.isContentEditable;
}

function wireKeyboard() {
  document.addEventListener("keydown", (event) => {
    const meta = event.ctrlKey || event.metaKey;
    if (meta && event.key.toLowerCase() === "k") {
      event.preventDefault();
      openPalette();
      return;
    }
    if (meta && event.key.toLowerCase() === "f") {
      if (state.session) {
        event.preventDefault();
        showSearch();
      }
      return;
    }
    if (event.key === "Escape") {
      if (document.querySelector("dialog[open]")) return;
      if (event.target.closest && event.target.closest(".turn-approval") && rejectPending()) {
        event.preventDefault();
        return;
      }
      if (state.running) {
        event.preventDefault();
        stopTurn();
      }
      return;
    }
    if (event.key === "?" && !typing(event.target)) {
      event.preventDefault();
      showDialog("dialog-help");
      return;
    }
    if (event.key === "Enter" && event.target && event.target.id === "transcript-search") {
      event.preventDefault();
      focusHit(state.search.current + 1);
    }
  });

  $("palette-input").addEventListener("input", (event) => renderPalette(event.target.value));
  $("palette-input").addEventListener("keydown", (event) => {
    if (event.key === "Enter") {
      event.preventDefault();
      const first = $("palette-list").querySelector("button:not(:disabled)");
      if (first) first.click();
    }
  });
}

// ---------- wiring ----------

function wire() {
  $("menu-button").addEventListener("click", toggleSidebar);
  $("transcript-search-button").addEventListener("click", toggleSearch);
  $("rail-search").addEventListener("click", () => {
    const row = $("rail-filter-row");
    row.hidden = !row.hidden;
    $("rail-search").setAttribute("aria-expanded", String(!row.hidden));
    if (!row.hidden) $("session-filter").focus();
  });
  $("new-session-button").addEventListener("click", () => openSessionDialog("create"));
  $("session-filter").addEventListener("input", renderSessions);
  $("model-button").addEventListener("click", openModelDialog);
  $("palette-button").addEventListener("click", openPalette);
  $("help-button").addEventListener("click", () => showDialog("dialog-help"));
  $("help-button-2").addEventListener("click", () => showDialog("dialog-help"));
  const settings = () => {
    renderSettings();
    renderStatusPanel();
    showDialog("dialog-settings");
  };
  $("settings-button").addEventListener("click", settings);
  $("status-button").addEventListener("click", settings);
  $("rename-button").addEventListener("click", () => {
    if (!state.session) return;
    askText({
      title: "Renomear a sessão",
      label: "Nome da sessão",
      value: state.session.Title || "",
      help: "Um nome curto ajuda a achar a sessão depois.",
      onSubmit: (value) => renameSession(state.session.ID, value),
    });
  });
  $("details-button").addEventListener("click", () => openSessionDialog("details"));
  $("recover-button").addEventListener("click", () => {
    if (state.session) recoverSession(state.session.ID);
  });
  $("stop-button").addEventListener("click", stopTurn);
  $("transcript").addEventListener("scroll", () => {
    state.following = atBottom(transcript());
  });
  $("transcript-search").addEventListener("input", () => applySearch());
  $("search-next").addEventListener("click", () => focusHit(state.search.current + 1));
  $("search-prev").addEventListener("click", () => focusHit(state.search.current - 1));
  $("copy-command").addEventListener("click", async () => {
    const ok = await copyText("ovc --ui");
    toast(ok ? "ok" : "info", ok ? "Comando copiado." : "Copie o comando direto da linha acima.");
  });
  $("gate-retry").addEventListener("click", () => refreshAll());
  $("gate-help").addEventListener("click", () => {
    const target = $("gate-help-body");
    target.hidden = !target.hidden;
    $("gate-help").setAttribute("aria-expanded", String(!target.hidden));
  });

  $("composer").addEventListener("submit", (event) => {
    event.preventDefault();
    const area = $("prompt");
    submit(area.value);
    area.value = "";
    area.focus();
  });
  $("prompt").addEventListener("keydown", (event) => {
    if (event.key !== "Enter" || event.shiftKey) return;
    const meta = event.ctrlKey || event.metaKey;
    if (!meta && !prefs.sendOnEnter) return;
    event.preventDefault();
    const area = $("prompt");
    submit(area.value);
    area.value = "";
  });
  $("prompt").addEventListener("input", () => saveDraft($("prompt").value));

  $("dialog-text-form").addEventListener("submit", async (event) => {
    event.preventDefault();
    const value = $("dialog-text-input").value.trim();
    if (!value) {
      const error = $("dialog-text-error");
      error.hidden = false;
      error.textContent = "Escreva um valor para continuar.";
      return;
    }
    closeDialog("dialog-text");
    if (textDialogSubmit) await textDialogSubmit(value);
  });

  const onPref = (id, apply) => $(id).addEventListener("change", apply);
  onPref("setting-density", (event) => {
    prefs.density = event.target.value;
    savePrefs();
    applyPrefs();
  });
  onPref("setting-scale", (event) => {
    prefs.scale = event.target.value;
    savePrefs();
    applyPrefs();
  });
  onPref("setting-motion", (event) => {
    prefs.motion = event.target.value;
    savePrefs();
    applyPrefs();
  });
  onPref("setting-send-enter", (event) => {
    prefs.sendOnEnter = event.target.checked;
    savePrefs();
  });
  onPref("setting-notify", async (event) => {
    if (!event.target.checked) {
      prefs.notify = false;
      savePrefs();
      return;
    }
    try {
      const permission = await Notification.requestPermission();
      prefs.notify = permission === "granted";
      if (!prefs.notify) toast("info", "O navegador não autorizou avisos; o estado continua visível na aba.");
    } catch {
      prefs.notify = false;
    }
    savePrefs();
    renderSettings();
  });
  $("settings-reset").addEventListener("click", () => {
    Object.assign(prefs, PREFERENCES);
    savePrefs();
    applyPrefs();
    renderSettings();
    toast("ok", "Ajustes restaurados.");
  });

  wireDialogs();
  wireKeyboard();
  renderQueue();
  fillBase();
}

/** Base do composer: ▀ de meia célula na cor do fill, uma vez por boot (§8.1). */
function fillBase() {
  const run = $("composer-base-fill");
  if (run) run.textContent = "▀".repeat(240);
}

// ---------- refresh and boot ----------

async function refreshAll() {
  try {
    state.info = await uiInfo();
  } catch (error) {
    showGate(error instanceof ApiError && error.kind === "unpaired" ? "unpaired" : "offline");
    return;
  }
  composerMeta();
  await refreshHealth();
  if (!state.info.configured) {
    view("onboarding");
    renderOnboarding();
    return;
  }
  view("chat");
  await refreshSessions();
  if (state.sessions.length) await openSession(state.sessions[0].ID);
  else renderWelcome();
  renderStatus();
}

/** Estado do serviço: protocolo e tarefas vivas (D1 lê active_tasks). */
async function refreshHealth() {
  const summary = await health().catch(() => null);
  if (summary) {
    state.protocol = summary.protocol;
    state.activeTasks = Number(summary.active_tasks) || 0;
  }
  renderStatus();
}

function boot() {
  applyPrefs();
  wire();
  if (!window.matchMedia("(max-width: 980px)").matches) {
    $("sidebar").hidden = false;
    $("menu-button").setAttribute("aria-expanded", "true");
  }
  refreshAll();
}

boot();
