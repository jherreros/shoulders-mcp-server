#!/usr/bin/env bash
set -euo pipefail

# Install the Shoulders agent skill for VS Code Copilot.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/jherreros/shoulders/main/scripts/install-skill.sh | bash
#   curl -fsSL ... | bash -s -- --workspace   # Install into current workspace
#   curl -fsSL ... | bash -s -- --global      # Install globally (default)

SKILL_URL="https://raw.githubusercontent.com/jherreros/shoulders/main/.github/skills/shoulders/SKILL.md"
SKILL_NAME="shoulders"

MODE="global"
for arg in "$@"; do
  case "$arg" in
    --workspace) MODE="workspace" ;;
    --global)    MODE="global" ;;
    --help|-h)
      echo "Usage: install-skill.sh [--global|--workspace]"
      echo ""
      echo "  --global     Install to ~/.copilot/skills/shoulders/ (default, works across all workspaces)"
      echo "  --workspace  Install to .github/skills/shoulders/ in the current directory"
      exit 0
      ;;
    *)
      echo "Unknown option: $arg" >&2
      exit 1
      ;;
  esac
done

if [ "$MODE" = "workspace" ]; then
  DEST_DIR=".github/skills/${SKILL_NAME}"
else
  DEST_DIR="${HOME}/.copilot/skills/${SKILL_NAME}"
fi

mkdir -p "$DEST_DIR"

echo "Downloading Shoulders skill..."
if command -v curl &> /dev/null; then
  curl -fsSL "$SKILL_URL" -o "${DEST_DIR}/SKILL.md"
elif command -v wget &> /dev/null; then
  wget -qO "${DEST_DIR}/SKILL.md" "$SKILL_URL"
else
  echo "Error: curl or wget is required." >&2
  exit 1
fi

echo "Shoulders skill installed to ${DEST_DIR}/SKILL.md"

if [ "$MODE" = "global" ]; then
  echo "The skill is available across all workspaces. Invoke it with /shoulders in VS Code Copilot chat."
else
  echo "The skill is available in this workspace. Invoke it with /shoulders in VS Code Copilot chat."
fi
