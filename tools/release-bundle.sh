#!/usr/bin/env bash

set -euo pipefail

output_root="${1:-dist}"
version="$(tr -d '\n' < internal/buildinfo/VERSION)"
dirty_suffix=""

if [[ -n "$(git status --porcelain --untracked-files=normal)" ]]; then
  if [[ "${GREPQUEST_ALLOW_DIRTY_RELEASE:-0}" != "1" ]]; then
    echo "release bundle requires a clean working tree" >&2
    exit 1
  fi
  dirty_suffix="-dirty"
  echo "warning: building an explicitly marked dirty development bundle" >&2
fi

platform="$(go env GOOS)-$(go env GOARCH)"
bundle_name="grepquest-${version}-${platform}${dirty_suffix}"
archive_name="${bundle_name}.tar.gz"
mkdir -p "$output_root"

if [[ -e "$output_root/$bundle_name" || -e "$output_root/$archive_name" || -e "$output_root/$archive_name.sha256" ]]; then
  echo "release output already exists for $bundle_name" >&2
  exit 1
fi

work_dir="$(mktemp -d "$output_root/.grepquest-release.XXXXXX")"
trap 'rm -rf "$work_dir"' EXIT
bundle_dir="$work_dir/$bundle_name"
mkdir -p "$bundle_dir/bin" "$bundle_dir/licenses/go-modules"

CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o "$bundle_dir/bin/grepquest" ./cmd/grepquest
cp LICENSE THIRD_PARTY_NOTICES.md "$bundle_dir/"
cp packs/core/v1/LICENSE "$bundle_dir/CORE_PACK_LICENSE"

dependency_manifest="$bundle_dir/licenses/go-modules/DEPENDENCIES.tsv"
printf 'module\tversion\n' > "$dependency_manifest"
go list -m -f '{{if .Dir}}{{printf "%s\t%s\t%s" .Path .Version .Dir}}{{end}}' all |
  while IFS=$'\t' read -r module_name module_version module_dir; do
    [[ -n "$module_name" && "$module_name" != "github.com/timfewi/grepquest" ]] || continue
    printf '%s\t%s\n' "$module_name" "$module_version" >> "$dependency_manifest"
    safe_name="$(printf '%s@%s' "$module_name" "$module_version" | tr '/:' '--')"
    module_license_dir="$bundle_dir/licenses/go-modules/$safe_name"
    while IFS= read -r -d '' license_file; do
      mkdir -p "$module_license_dir"
      cp "$license_file" "$module_license_dir/$(basename "$license_file")"
    done < <(find "$module_dir" -maxdepth 1 -type f \( -iname 'LICENSE*' -o -iname 'COPYING*' -o -iname 'NOTICE*' \) -print0)
    if [[ ! -d "$module_license_dir" ]]; then
      echo "missing top-level license text for $module_name $module_version" >&2
      exit 1
    fi
  done

tar --sort=name --mtime='@0' --owner=0 --group=0 --numeric-owner -C "$work_dir" -cf - "$bundle_name" |
  gzip -n > "$output_root/$archive_name"
(
  cd "$output_root"
  sha256sum "$archive_name" > "$archive_name.sha256"
)

printf 'Created %s\n' "$output_root/$archive_name"
printf 'Checksum %s\n' "$output_root/$archive_name.sha256"
