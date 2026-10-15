#!/usr/bin/env bash
# Install an already downloaded/built binary. No sudo, shell-profile edits, or
# unverified remote execution. Existing user configuration is never touched.
set -euo pipefail
if [ "$#" -gt 1 ]; then echo "Usage: bash install.sh [local-binary]" >&2; exit 2; fi
case "$(uname -s)" in Linux) target_os=linux ;; Darwin) target_os=darwin ;; *) echo "Use install.ps1 on Windows" >&2; exit 1 ;; esac
case "$(uname -m)" in x86_64|amd64) target_arch=amd64 ;; arm64|aarch64) target_arch=arm64 ;; *) echo "Unsupported CPU architecture" >&2; exit 1 ;; esac
if [ "$#" = 1 ]; then source_binary=$1
else
  staging="$(mktemp -d)"
  trap 'rm -rf "$staging"' EXIT
  release="${INTERPRETER_VERSION:-latest}"
  if [ "$release" = latest ]; then base="https://github.com/iagoluch/OpenVibeCoding/releases/latest/download"
  else
    case "$release" in *[!a-zA-Z0-9._-]*) echo "Invalid release version" >&2; exit 1 ;; esac
    base="https://github.com/iagoluch/OpenVibeCoding/releases/download/$release"
  fi
  asset="interpreter-$target_os-$target_arch.tar.gz"
  curl --fail --location --proto '=https' --tlsv1.2 "$base/$asset" -o "$staging/$asset" || { echo "No downloadable release available. Build from source with Go 1.26.2+ and pass the local binary." >&2; exit 1; }
  curl --fail --location --proto '=https' --tlsv1.2 "$base/SHA256SUMS" -o "$staging/SHA256SUMS"
  expected="$(awk -v name="$asset" '$2==name {print $1}' "$staging/SHA256SUMS")"
  if command -v sha256sum >/dev/null; then actual="$(sha256sum "$staging/$asset" | awk '{print $1}')"; else actual="$(shasum -a 256 "$staging/$asset" | awk '{print $1}')"; fi
  [ -n "$expected" ] && [ "$expected" = "$actual" ] || { echo "Checksum mismatch" >&2; exit 1; }
  tar -xzf "$staging/$asset" -C "$staging" interpreter i
  source_binary="$staging/interpreter"
fi
if ! "$source_binary" --version | grep -Eq '^(OpenVibeCoding|Open Interpreter) '; then
  echo "Invalid interpreter binary" >&2; exit 1
fi
install_dir="${INTERPRETER_BIN_DIR:-$HOME/.local/bin}"
mkdir -p "$install_dir"
for name in interpreter i; do
  if [ -e "$install_dir/$name" ] && ! "$install_dir/$name" --version 2>/dev/null | grep -Eq '^(OpenVibeCoding|Open Interpreter) '; then
    echo "Refusing to replace unrelated command: $install_dir/$name" >&2; exit 1
  fi
done
for name in interpreter i; do
  install -m 755 "$source_binary" "$install_dir/$name.new"
  mv -f "$install_dir/$name.new" "$install_dir/$name"
done
"$install_dir/interpreter" --version
"$install_dir/i" --version
printf 'Installed to %s. Add this directory to PATH if necessary.\n' "$install_dir"
