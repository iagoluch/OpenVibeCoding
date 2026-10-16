// The turn stream, mirroring internal/runtime/client.go.
//
// POST /run answers with newline-delimited JSON: one object per line with
// Kind (delta, status, response, tool, approval, done), Text, Approval and
// Usage. The request body is exactly the one the CLI client builds, so the
// browser and the TUI exercise the same contract.

import { post } from "./api.js";

/** Builds the /run body of internal/runtime/client.go (Client.Run). */
export function turnBody({ session, prompt, provider, model, harness, workspace, permissions, automation }) {
  return {
    Session: session,
    Prompt: prompt,
    Provider: provider,
    Model: model,
    Harness: harness,
    Workspace: workspace,
    Shell: false,
    Permissions: permissions,
    Automation: automation,
  };
}

/**
 * Reads the NDJSON stream and calls onEvent for each parsed event. A line that
 * is not valid JSON is reported through onBrokenLine and skipped: one bad
 * frame must not end the turn.
 */
export async function readEvents(response, onEvent, onBrokenLine) {
  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = "";
  const flush = (line) => {
    const text = line.trim();
    if (!text) return;
    let event;
    try {
      event = JSON.parse(text);
    } catch {
      if (onBrokenLine) onBrokenLine(text);
      return;
    }
    onEvent(event);
  };
  for (;;) {
    const { value, done } = await reader.read();
    if (done) break;
    buffer += decoder.decode(value, { stream: true });
    let end = buffer.indexOf("\n");
    while (end >= 0) {
      flush(buffer.slice(0, end));
      buffer = buffer.slice(end + 1);
      end = buffer.indexOf("\n");
    }
  }
  buffer += decoder.decode();
  flush(buffer);
}

/** Starts one turn and streams its events until the runtime closes the stream. */
export async function runTurn(options, onEvent, onBrokenLine) {
  const response = await post("/run", turnBody(options), options.signal);
  await readEvents(response, onEvent, onBrokenLine);
}
