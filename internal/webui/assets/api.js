// Transport for the authenticated loopback runtime.
//
// Every call is same-origin POST, carries the mandatory client header and
// reuses the pairing cookie the browser already holds. This module never
// invents endpoints and never holds a credential: the cookie belongs to the
// browser, the page only speaks to the origin that served it.

let identity = "browser";

/** Sets the stable client id sent as X-OpenVibeCoding-Client on every call. */
export function identify(id) {
  if (id) identity = id;
}

export class ApiError extends Error {
  constructor({ status = 0, message = "", kind = "http", detail = "" }) {
    super(message || "falha na chamada");
    this.name = "ApiError";
    this.status = status;
    this.kind = kind;
    this.detail = detail;
  }
}

async function bodyText(response, limit = 4096) {
  try {
    const text = await response.text();
    return text.slice(0, limit).trim();
  } catch {
    return "";
  }
}

/** POSTs JSON and returns the raw response; throws ApiError on failure. */
export async function post(path, body, signal) {
  let response;
  try {
    response = await fetch(path, {
      method: "POST",
      credentials: "same-origin",
      headers: { "Content-Type": "application/json", "X-OpenVibeCoding-Client": identity },
      body: JSON.stringify(body === undefined ? {} : body),
      signal,
    });
  } catch (error) {
    if (error && error.name === "AbortError") throw error;
    throw new ApiError({
      kind: "network",
      message: "o serviço local não respondeu",
      detail: error && error.message ? error.message : "",
    });
  }
  if (response.status === 401) {
    throw new ApiError({ status: 401, kind: "unpaired", message: "esta aba não está autorizada" });
  }
  if (!response.ok) {
    throw new ApiError({ status: response.status, message: await bodyText(response), detail: "" });
  }
  return response;
}

async function json(path, body, signal) {
  return (await post(path, body, signal)).json();
}

/** One storage operation; the runtime owns every mutation. */
export async function store(operation) {
  const payload = await json("/store", operation);
  if (payload && payload.error) throw new ApiError({ status: 200, kind: "store", message: payload.error });
  return payload ? payload.value : undefined;
}

export function uiInfo() {
  return json("/ui/info", {});
}

export function health() {
  return json("/health", {});
}

async function uiAction(path, body) {
  const payload = await json(path, body);
  if (payload && payload.error) throw new ApiError({ status: 200, kind: "setup", message: payload.error });
  return payload;
}

/** Connects a provider: stores the optional key, activates and smoke-tests. */
export function uiSetup({ provider, model = "", key = "", force = false }) {
  return uiAction("/ui/setup", { provider, model, key, force });
}

/** Activates a provider/model pair that is already known to work. */
export function uiSelect({ provider, model }) {
  return uiAction("/ui/select", { provider, model });
}

export function approve(id, allow) {
  return json("/approve", { ID: id, Allow: allow });
}

// ---------- explaining failures to a beginner ----------

const patterns = [
  [/bubblewrap|sandbox|isolation/i, "Esta máquina não oferece isolamento de comandos", "Por segurança, comandos não rodam sem isolamento. No terminal, ovc --allow-unsandboxed-shell permite aprovar cada comando individualmente."],
  [/no such file|cannot find|system cannot find|não pode encontrar|does not exist/i, "A pasta não foi encontrada", "Confira o caminho da pasta do projeto: ele precisa existir e estar escrito por completo, por exemplo C:\\projetos\\meu-app."],
  [/unknown provider|unknown model|não é conhecido/i, "A escolha de modelo não existe mais", "Abra Trocar de modelo e conecte o provedor outra vez."],
  [/rate limit|too many requests|\b429\b/i, "O provedor pediu para esperar", "Você atingiu o limite gratuito por agora. Espere alguns minutos ou escolha outro modelo."],
  [/unauthorized|invalid.*(api)?.*key|incorrect api key|\b401\b|\b403\b/i, "A chave foi recusada", "Confira se a chave foi copiada inteira (sem espaços) e se ela pertence ao provedor escolhido."],
  [/quota|insufficient|billing|\b402\b/i, "A conta do provedor não tem saldo", "Escolha um provedor com plano gratuito ou libere créditos na conta."],
  [/timeout|deadline|slow|demorou|no response/i, "O provedor demorou demais para responder", "Costuma ser passageiro: tente de novo. Se repetir, escolha outro modelo."],
  [/connection refused|no route|network is unreachable|dial tcp|failed to connect|offline/i, "Sem conexão com o provedor", "Verifique a internet desta máquina — o modelo escolhido roda fora dela."],
  [/context|token/i, "A conversa ficou grande demais para este modelo", "Comece uma sessão nova para seguir com um histórico mais curto."],
  [/audit unavailable/i, "O banco de sessões não respondeu", "Feche e rode ovc --ui outra vez para reabrir o serviço."],
  [/permission|denied|forbidden/i, "A operação foi recusada pelo serviço", "Se você recusou uma aprovação, o agente já sabe. Caso contrário, tente de novo."],
];

/** Turns any failure into a title, a plain explanation and the raw detail. */
export function explain(error, context = "") {
  const detail = error && error.message ? error.message : String(error || "");
  const raw = (error && error.detail ? error.detail + " " : "") + detail;
  const base = { title: "Algo não saiu como esperado", hint: "", detail: raw.trim() };

  if (error instanceof ApiError && error.kind === "network") {
    return { ...base, title: "Não encontramos o serviço local", hint: "Rode ovc --ui no terminal: ele sobe o serviço e imprime um link novo. Se o painel já estava aberto, confira se o terminal continua rodando." };
  }
  if (error instanceof ApiError && error.kind === "unpaired") {
    return { ...base, title: "Esta aba não está autorizada", hint: "Cada aba entra por um link de uso único criado por ovc --ui. Rode o comando e abra o link aqui." };
  }
  if (error instanceof ApiError && error.status === 403) {
    return { ...base, title: "Quem pediu foi outra aba", hint: context === "approve"
      ? "Esta aprovação pertence ao painel que iniciou o pedido. Responda nele, ou interrompa aqui e envie a mensagem de novo."
      : "A ação foi recusada pela origem. Recarregue a página com o link novo do ovc --ui." };
  }
  if (error instanceof ApiError && error.status === 409) {
    return { ...base, title: "Já existe um trabalho em andamento", hint: "Esta sessão (ou a pasta dela) está ocupada — talvez em outra aba. Espere terminar, interrompa ou escolha outra sessão." };
  }
  if (error instanceof ApiError && error.status === 413) {
    return { ...base, title: "A mensagem é grande demais", hint: "Cole o texto em etapas ou aponte o arquivo pelo nome para o agente ler." };
  }
  if (error instanceof ApiError && error.status === 400 && context === "run") {
    return { ...base, title: "O pedido foi recusado antes de começar", hint: "Confira a pasta da sessão e o modelo ativo (Trocar de modelo). Depois envie de novo." };
  }
  for (const [match, title, hint] of patterns) {
    if (match.test(raw)) return { ...base, title, hint };
  }
  if (context === "setup" || (error instanceof ApiError && error.kind === "setup")) {
    return { ...base, title: "Não deu para ativar este modelo", hint: "Confira a chave e escolha outro modelo da mesma lista. O que já estava funcionando continua valendo." };
  }
  return base;
}
