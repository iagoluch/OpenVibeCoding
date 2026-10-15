#!/usr/bin/env bash
set -euo pipefail
install_dir="${OPENVIBECODING_BIN_DIR:-$HOME/.local/bin}"
for name in ovc; do
  if [ -f "$install_dir/$name" ] && "$install_dir/$name" --version | grep -Eq '^OpenVibeCoding '; then rm "$install_dir/$name"; fi
done
printf 'Removed launcher from %s. Configuration and sessions were preserved.\n' "$install_dir"
