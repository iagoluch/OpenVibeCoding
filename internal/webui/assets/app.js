// Placeholder client for the paired runtime: vanilla JS, no frameworks, no
// remote assets. Every call is same-origin and carries this tab's client id.
const client = crypto.randomUUID();
const el = (id) => document.getElementById(id);
const state = { info: null, session: null, workspace: "", running: null };

function say(text) {
  el("notice").textContent = text || "";
  el("notice").hidden = !text;
}

async function call(path, body, signal) {
  const response = await fetch(path, {
    method: "POST",
    credentials: "same-origin",
    headers: { "Content-Type": "application/json", "X-OpenVibeCoding-Client": client },
    body: JSON.stringify(body || {}),
    signal
  });
  if (response.status === 401) throw new Error("unpaired");
  if (!response.ok) throw new Error("runtime HTTP " + response.status);
  return response.json();
}

async function store(request) {
  const result = await call("/store", request);
  if (result.error) throw new Error(result.error);
  return result.value;
}

function line(kind, text) {
  const node = document.createElement("pre");
  node.className = "line " + kind;
  node.textContent = text || "";
  return node;
}

function scroll() {
  const box = el("transcript");
  box.scrollTop = box.scrollHeight;
}

function options(select, items, value, label) {
  select.textContent = "";
  for (const item of items) {
    const option = document.createElement("option");
    option.value = value(item);
    option.textContent = label(item);
    select.append(option);
  }
}

async function refresh() {
  state.info = await call("/ui/info", {});
  el("setup").hidden = state.info.configured;
  el("work").hidden = !state.info.configured;
  if (!state.info.configured) {
    options(el("providers"), state.info.providers, (p) => p.id,
      (p) => p.title + (p.free ? " · free" : "") + (p.local ? " · local" : "") + (p.has_key ? " · key stored" : ""));
    return;
  }
  const active = state.info.active;
  el("session-meta").textContent = active.provider + "/" + active.model +
    (state.info.workspace ? " · " + state.info.workspace : "");
  await loadSessions();
}

async function loadSessions() {
  const list = (await store({ Op: "list" })) || [];
  if (!list.some((session) => session.ID === state.session)) state.session = list.length ? list[0].ID : null;
  options(el("sessions"), list, (session) => session.ID, (session) => session.Title + " · " + session.Workspace);
  el("sessions").value = state.session || "";
  if (!state.session) return;
  const info = await store({ Op: "get", ID: state.session });
  state.workspace = info.Workspace;
  const history = (await store({ Op: "history", ID: state.session })) || [];
  const box = el("transcript");
  box.textContent = "";
  box.append(line("meta", info.ID + " · " + info.Workspace));
  for (const entry of history) {
    const message = entry.Message || {};
    if (message.content) box.append(line(message.role || "unknown", message.content));
  }
  scroll();
}

async function connect() {
  const provider = el("providers").value;
  el("setup-status").textContent = "Testing " + provider + " with a real request...";
  try {
    const result = await call("/ui/setup", { provider, key: el("key").value.trim() });
    if (!result.ok) { el("setup-status").textContent = result.error || "setup failed"; return; }
    el("key").value = "";
    await refresh();
  } catch (error) {
    el("setup-status").textContent = error.message;
  }
}

async function newSession() {
  const workspace = state.info.workspace || window.prompt("Absolute workspace folder for the new session:") || "";
  if (!workspace) return;
  state.session = await store({ Op: "new", Workspace: workspace });
  await loadSessions();
}

function approval(box, event) {
  const wrap = document.createElement("div");
  wrap.className = "line approval";
  const text = document.createElement("div");
  text.textContent = event.Text;
  const row = document.createElement("div");
  row.className = "row";
  const answer = (allow) => async (clicked) => {
    for (const button of row.children) button.disabled = true;
    try {
      await call("/approve", { ID: event.Approval, Allow: allow });
      text.textContent += allow ? " — allowed" : " — denied";
    } catch (error) {
      text.textContent += " — " + error.message;
    }
  };
  const allows = document.createElement("button");
  allows.type = "button";
  allows.textContent = "Allow";
  const denies = document.createElement("button");
  denies.type = "button";
  denies.textContent = "Deny";
  allows.addEventListener("click", answer(true));
  denies.addEventListener("click", answer(false));
  row.append(allows, denies);
  wrap.append(text, row);
  box.append(wrap);
}

function handleEvent(event, pending, tools) {
  if (event.Kind === "delta") pending.textContent += event.Text;
  else if (event.Kind === "response") pending.textContent = event.Text;
  else if (event.Kind === "approval") approval(tools, event);
  else if (event.Kind === "status" || event.Kind === "tool") tools.append(line(event.Kind, event.Text));
  else if (event.Kind === "done" && event.Text) pending.textContent = event.Text;
  scroll();
}

async function readStream(response, pending, tools) {
  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = "";
  for (;;) {
    const { value, done } = await reader.read();
    if (done) return;
    buffer += decoder.decode(value, { stream: true });
    let end;
    while ((end = buffer.indexOf("\n")) >= 0) {
      const text = buffer.slice(0, end).trim();
      buffer = buffer.slice(end + 1);
      if (text) handleEvent(JSON.parse(text), pending, tools);
    }
  }
}

async function send(event) {
  event.preventDefault();
  const prompt = el("prompt").value.trim();
  if (!state.session || !prompt || state.running) return;
  el("prompt").value = "";
  const pending = line("assistant", "");
  const tools = document.createElement("div");
  el("transcript").append(line("user", prompt), pending, tools);
  scroll();
  state.running = new AbortController();
  el("send").disabled = true;
  el("stop").disabled = false;
  const active = state.info.active;
  const body = {
    Session: state.session, Prompt: prompt, Provider: active.provider, Model: active.model,
    Harness: active.harness, Permissions: state.info.permissions, Automation: state.info.automation,
    Workspace: state.workspace, Shell: false
  };
  try {
    const response = await fetch("/run", {
      method: "POST",
      credentials: "same-origin",
      headers: { "Content-Type": "application/json", "X-OpenVibeCoding-Client": client },
      body: JSON.stringify(body),
      signal: state.running.signal
    });
    if (response.status === 401) say("Open the pairing link printed by ovc --ui.");
    else if (!response.ok) pending.textContent = "runtime HTTP " + response.status;
    else await readStream(response, pending, tools);
  } catch (error) {
    pending.textContent += error.name === "AbortError" ? "\n[stopped]" : "\n" + error.message;
  } finally {
    state.running = null;
    el("send").disabled = false;
    el("stop").disabled = true;
    await loadSessions();
  }
}

el("connect").addEventListener("click", connect);
el("new").addEventListener("click", newSession);
el("sessions").addEventListener("change", async (event) => { state.session = event.target.value; await loadSessions(); });
el("composer").addEventListener("submit", send);
el("stop").addEventListener("click", () => { if (state.running) state.running.abort(); });

(async () => {
  try {
    await refresh();
  } catch (error) {
    say(error.message === "unpaired"
      ? "This page is not paired with the running daemon. Open the pairing link printed by ovc --ui."
      : "Runtime unavailable: " + error.message);
  }
})();
