set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
output_dir="$script_dir/export"
mkdir -p "$output_dir"

resvg --width 512 --height 512 "$script_dir/logo/grepquest-mark.svg" "$output_dir/grepquest-mark-512.png"
resvg --width 128 --height 128 "$script_dir/logo/grepquest-mark.svg" "$output_dir/grepquest-mark-128.png"
resvg --width 64 --height 64 "$script_dir/logo/grepquest-mark.svg" "$output_dir/grepquest-mark-64.png"
resvg --width 1280 --height 320 "$script_dir/logo/grepquest-lockup.svg" "$output_dir/grepquest-lockup-1280.png"
