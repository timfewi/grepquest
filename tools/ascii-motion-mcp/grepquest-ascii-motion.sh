set -euo pipefail

usage() {
  cat <<'EOF'
Usage: grepquest-ascii-motion [--live] [--port PORT] [--help] [--version]

Runs the pinned Ascii Motion MCP server with file access restricted to
assets/ascii-motion. Network access is disabled unless --live is explicit.
EOF
}

live=0
declare -a tool_args=()

while (($# > 0)); do
  case "$1" in
    --live)
      live=1
      tool_args+=("--live")
      shift
      ;;
    --port)
      if (($# < 2)) || [[ ! "$2" =~ ^[0-9]+$ ]] || ((10#$2 < 1 || 10#$2 > 65535)); then
        echo "--port requires an integer from 1 through 65535" >&2
        exit 2
      fi
      tool_args+=("--port" "$2")
      shift 2
      ;;
    --help)
      usage
      exit 0
      ;;
    --version)
      tool_args+=("--version")
      shift
      ;;
    *)
      printf 'unsupported argument: %s\n' "$1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

repo_root="$(git rev-parse --show-toplevel 2>/dev/null)" || {
  echo "run grepquest-ascii-motion from inside the GrepQuest repository" >&2
  exit 1
}
repo_root="$(realpath -e "$repo_root")"

if [[ ! -f "$repo_root/go.mod" ]] || ! grep -qx 'module github.com/timfewi/grepquest' "$repo_root/go.mod"; then
  echo "the detected repository is not GrepQuest" >&2
  exit 1
fi

asset_root="$repo_root/assets/ascii-motion"
if [[ -L "$asset_root" ]]; then
  echo "refusing a symlinked Ascii Motion asset directory" >&2
  exit 1
fi
mkdir -p "$asset_root"
asset_root="$(realpath -e "$asset_root")"
if [[ "$asset_root" != "$repo_root/assets/ascii-motion" ]]; then
  echo "Ascii Motion asset directory escaped the repository" >&2
  exit 1
fi

ascii_motion_bin="$(command -v ascii-motion-mcp)" || {
  echo "the pinned ascii-motion-mcp executable is unavailable" >&2
  exit 1
}
ascii_motion_bin="$(realpath -e "$ascii_motion_bin")"
case "$ascii_motion_bin" in
  /nix/store/*/bin/ascii-motion-mcp) ;;
  *)
    echo "refusing an ascii-motion-mcp executable outside the Nix store" >&2
    exit 1
    ;;
esac

declare -a sandbox_args=(
  --die-with-parent
  --new-session
  --unshare-all
  --cap-drop ALL
  --clearenv
  --ro-bind /nix/store /nix/store
  --dev /dev
  --proc /proc
  --tmpfs /tmp
  --dir /tmp/home
  --bind "$asset_root" /project
  --chdir /project
  --setenv HOME /tmp/home
  --setenv XDG_CACHE_HOME /tmp/cache
  --setenv XDG_CONFIG_HOME /tmp/config
  --setenv TMPDIR /tmp
  --setenv NODE_ENV production
  --setenv PATH "$(dirname "$ascii_motion_bin")"
)

if ((live)); then
  sandbox_args+=(--share-net)
fi

exec bwrap "${sandbox_args[@]}" \
  "$ascii_motion_bin" --project-dir /project "${tool_args[@]}"
