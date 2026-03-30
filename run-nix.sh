#!/usr/bin/env bash
set -euo pipefail

if ! command -v nix >/dev/null 2>&1; then
  echo "Error: nix is not installed or not in PATH." >&2
  echo "Install Nix in WSL first, then run this script again." >&2
  exit 1
fi

if [[ ! -f "flake.nix" ]]; then
  echo "Error: flake.nix was not found in the current directory." >&2
  exit 1
fi

if ! command -v git >/dev/null 2>&1; then
  echo "Error: git is not installed or not in PATH." >&2
  exit 1
fi

required_files=("main.go" "go.mod" "flake.nix")
untracked=()

for file in "${required_files[@]}"; do
  if [[ ! -f "$file" ]]; then
    echo "Error: required file '$file' is missing." >&2
    exit 1
  fi

  if ! git ls-files --error-unmatch "$file" >/dev/null 2>&1; then
    untracked+=("$file")
  fi
done

if (( ${#untracked[@]} > 0 )); then
  echo "Error: Nix flakes only include Git-tracked files in local builds." >&2
  echo "Track these files first:" >&2
  printf '  - %s\n' "${untracked[@]}" >&2
  echo "Then run: git add ${untracked[*]}" >&2
  exit 1
fi

echo "Building flake package..."
nix --extra-experimental-features "nix-command flakes" build .#default

echo "Running app..."
nix --extra-experimental-features "nix-command flakes" run .#default -- "$@"