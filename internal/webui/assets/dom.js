// DOM helpers for the OpenVibeCoding web UI.
//
// Everything the page shows comes from the runtime, the model or the user's
// files, so this module builds nodes with createElement/textContent only.
// Markup is never assembled from strings and nothing is evaluated: output is
// data, never code.

/** Creates an element: el("p", { class: "muted" }, "texto"). */
export function el(tag, attributes = {}, ...children) {
  const node = document.createElement(tag);
  for (const [key, value] of Object.entries(attributes || {})) {
    if (value === undefined || value === null || value === false) continue;
    if (key === "class") node.className = value;
    else if (key === "text") node.textContent = value;
    else if (key === "dataset") Object.assign(node.dataset, value);
    else if (key === "for") node.htmlFor = value;
    else node.setAttribute(key, value === true ? "" : String(value));
  }
  append(node, children);
  return node;
}

/** Appends strings as text nodes and everything else as a node. */
export function append(parent, children) {
  for (const child of children.flat(Infinity)) {
    if (child === undefined || child === null || child === false) continue;
    parent.append(typeof child === "object" ? child : String(child));
  }
  return parent;
}

export function clear(node) {
  node.replaceChildren();
  return node;
}

export function byId(id) {
  return document.getElementById(id);
}

/** Wraps every occurrence of term in <mark>, splitting plain text only. */
export function highlight(text, term) {
  const fragment = document.createDocumentFragment();
  if (!term) {
    fragment.append(text);
    return fragment;
  }
  const haystack = text.toLowerCase();
  const needle = term.toLowerCase();
  let index = 0;
  for (;;) {
    const at = haystack.indexOf(needle, index);
    if (at < 0) break;
    if (at > index) fragment.append(text.slice(index, at));
    fragment.append(el("mark", {}, text.slice(at, at + needle.length)));
    index = at + needle.length;
  }
  fragment.append(text.slice(index));
  return fragment;
}

export function basename(path) {
  const parts = String(path || "").split(/[\\/]+/).filter(Boolean);
  return parts.length ? parts[parts.length - 1] : String(path || "");
}

export function shorten(path, limit = 44) {
  const value = String(path || "");
  if (value.length <= limit) return value;
  const parts = value.split(/[\\/]+/).filter(Boolean);
  const tail = parts.slice(-2).join("/");
  return "…/" + (tail.length <= limit - 2 ? tail : parts[parts.length - 1] || value.slice(-limit));
}

const numbers = new Intl.NumberFormat("pt-BR");
const relative = typeof Intl.RelativeTimeFormat === "function" ? new Intl.RelativeTimeFormat("pt-BR", { numeric: "auto" }) : null;

export function count(value) {
  return numbers.format(Number(value) || 0);
}

/** Formats an RFC3339 timestamp as "há 3 min", "ontem" or a date. */
export function when(iso) {
  const time = Date.parse(iso || "");
  if (!Number.isFinite(time)) return "";
  const seconds = (time - Date.now()) / 1000;
  const steps = [
    [60, "second"],
    [60, "minute"],
    [24, "hour"],
    [7, "day"],
    [4.35, "week"],
    [12, "month"],
  ];
  if (!relative) return new Date(time).toLocaleDateString("pt-BR");
  let value = seconds;
  for (const [size, unit] of steps) {
    if (Math.abs(value) < size) return relative.format(Math.round(value), unit);
    value /= size;
  }
  return relative.format(Math.round(value), "year");
}

export function clock(iso) {
  const time = Date.parse(iso || "");
  if (!Number.isFinite(time)) return "";
  return new Date(time).toLocaleTimeString("pt-BR", { hour: "2-digit", minute: "2-digit" });
}

/** One short line out of a possibly multi-line tool result. */
export function firstLine(text, limit = 160) {
  const line = String(text || "").split(/\r?\n/).find((value) => value.trim()) || "";
  const trimmed = line.trim();
  return trimmed.length > limit ? trimmed.slice(0, limit) + "…" : trimmed;
}

export async function copyText(value) {
  if (navigator.clipboard && window.isSecureContext) {
    try {
      await navigator.clipboard.writeText(value);
      return true;
    } catch {
      // fall through to the selection fallback
    }
  }
  const area = el("textarea", { class: "sr-only", "aria-hidden": "true" });
  area.value = value;
  document.body.append(area);
  area.select();
  let ok = false;
  try {
    ok = document.execCommand("copy");
  } catch {
    ok = false;
  }
  area.remove();
  return ok;
}

/** Announces a short update to screen readers without moving focus. */
export function announce(message) {
  const node = byId("announcer");
  if (!node) return;
  node.textContent = "";
  window.setTimeout(() => {
    node.textContent = message;
  }, 30);
}

/** Shows a transient message; the container is the polite live region. */
export function toast(kind, message, actionLabel, action, timeout = 7000) {
  const box = byId("toasts");
  if (!box) return null;
  const actions = el("div", { class: "buttons" });
  if (actionLabel && action) {
    const run = el("button", { class: "link-button", type: "button" }, actionLabel);
    run.addEventListener("click", () => {
      card.remove();
      action();
    });
    actions.append(run);
  }
  const close = el("button", { class: "link-button", type: "button", "aria-label": "Fechar aviso" }, "Fechar");
  close.addEventListener("click", () => card.remove());
  actions.append(close);
  const card = el("div", { class: `toast toast-${kind}` }, el("p", {}, message), actions);
  box.append(card);
  if (timeout > 0) window.setTimeout(() => card.remove(), timeout);
  return card;
}
