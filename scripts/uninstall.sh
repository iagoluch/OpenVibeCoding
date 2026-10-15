#!/usr/bin/env bash
set -euo pipefail
install_dir="${INTERPRETER_BIN_DIR:-$HOME/.local/bin}"
for name in i interpreter; do
  if [ -f "$install_dir/$name" ] && "$install_dir/$name" --version | grep -Eq '^(OpenVibeCoding|Open Interpreter) '; then rm "$install_dir/$name"; fi
done
printf 'Removed launchers from %s. Configuration and sessions were preserved.\n' "$install_dir"
