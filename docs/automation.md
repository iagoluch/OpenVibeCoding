# Automation contract

Browser and native desktop are distinct tools and permissions. Shell execution does not imply either grant. The inspect harness rejects both even if their grants are enabled.

## Browser request

```json
{
  "origins": ["http://127.0.0.1:3000"],
  "steps": [
    {"action":"open","url":"http://127.0.0.1:3000"},
    {"action":"fill","selector":"#name","text":"Synthetic user"},
    {"action":"click","selector":"button[type=submit]"},
    {"action":"expect_text","selector":"#result","text":"Saved"},
    {"action":"screenshot"}
  ]
}
```

The model passes the serialized object in the browser tool's `workflow` string. The UI shows the whole workflow before approval. At most 30 steps run, each with a ten-second locator timeout and a two-minute overall deadline. Origins are exact scheme/host/port matches; routes outside them are denied, including redirects and subresources. This is a browser routing policy, not an OS network sandbox. Use trusted test sites; browser engine activity outside routed page requests is not claimed to be isolated. Failures return an explicit failed result, never a fabricated assertion pass.

A screenshot is a PNG in transient memory. The agent strips it from persisted tool output, sends the latest images to an image-capable model, and retains only a placeholder in the session. Resuming requires fresh capture. An endpoint/model without image support can reject such a request; select a model supporting images for visual workflows.

## Native request

```json
{"action":"click","x":400,"y":220,"button":1}
```

The model passes the serialized object as the desktop tool's `action` string. Mouse, keyboard, clipboard and screenshots use platform APIs/tools; they do not run through the shell tool. A single action has a fifteen-second deadline. Users can stop with Esc/Ctrl+C, which cancels the active worker. Completed input cannot be undone by cancellation. OS-blocked or unavailable actions return errors without granting additional privileges.

Linux key names use xdotool syntax (`ctrl+l`, `Return`); Windows keys use Windows Forms SendKeys syntax; macOS keys use virtual key codes 0–127. Linux close requires a numeric window ID; Windows/macOS close targets the currently foreground window. These differences are intentional and must be checked before acting. Launch/focus requires the exact configured allowlist entry. Global pointer/keyboard actions have desktop-wide effects and are individually approved.

On Unix, ongoing automation subprocess groups are terminated on cancellation; detached applications launched intentionally are separate persistent user state. On Windows, cancellation invokes taskkill for the action process tree. These mechanisms cannot undo completed actions or guarantee termination of deliberately detached host processes. The tool never interacts with OS privilege prompts or grants Accessibility rights to itself.

## External desktop backend

Configure a trusted local executable:

```json
{"automation":{"desktop":false,"desktop_backend":["/absolute/path/to/my-adapter"],"applications":["my-app"]}}
```

After desktop permission and action approval, the executable receives exactly one JSON action object on stdin. It must return observed results on stdout, exit nonzero on failure, honor termination, and enforce any additional OS/app policy. Output is bounded to 8 MiB. Return screenshots as `{"image":"BASE64_PNG","mime":"image/png"}`; the same transient-image handling applies. Never emit credentials or private clipboard content unless the approved action specifically requires it. Additional arguments are fixed user configuration, never model-generated command text.

This interface supports platform-specific integrations (for example UI Automation, macOS accessibility frameworks or compositor-approved Wayland tooling) without modifying the agent core. Availability of an executable is only a prerequisite check; actual actions still verify their results and permissions.
