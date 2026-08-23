#!/usr/bin/env bash

set -euo pipefail

failed=0

pass() {
  printf '%-24s %s\n' "$1" "ok"
}

fail() {
  printf '%-24s %s\n' "$1" "$2" >&2
  failed=1
}

check_root_owned_path() {
  label="$1"
  candidate="$2"
  expected_type="${3:-path}"
  if [[ ! -e "$candidate" ]]; then
    fail "$label" "missing: $candidate"
    return
  fi
  resolved="$(realpath -e -- "$candidate")"
  if [[ "$expected_type" == "executable" && ( ! -f "$resolved" || ! -x "$resolved" ) ]]; then
    fail "$label" "must resolve to an executable regular file: $candidate"
    return
  fi
  if [[ "$expected_type" == "directory" && ! -d "$resolved" ]]; then
    fail "$label" "must resolve to a directory: $candidate"
    return
  fi

  current="$resolved"
  while true; do
    owner="$(stat -c '%u' "$current")"
    mode="$(stat -c '%a' "$current")"
    permissions="$(stat -c '%A' "$current")"
    if [[ "$owner" != "0" ]]; then
      fail "$label" "path component must be root-owned: $current"
      return
    fi
    if [[ ( "$permissions" == ?????w???? || "$permissions" == ????????w? ) && "$permissions" != *t && "$permissions" != *T ]]; then
      fail "$label" "path component must not be group/world writable: $current (mode $mode)"
      return
    fi
    [[ "$current" == "/" ]] && break
    current="$(dirname -- "$current")"
  done
  pass "$label"
}

if [[ "$(uname -s)" != "Linux" || "$(uname -m)" != "x86_64" ]]; then
  fail "platform" "requires x86_64 Linux"
else
  pass "platform"
fi

firecracker_binary="${GREPQUEST_FIRECRACKER:-$(command -v firecracker || true)}"
jailer_binary="${GREPQUEST_JAILER:-$(command -v jailer || true)}"
jail_base="${GREPQUEST_JAIL_BASE:-}"
cgroup_relative="${GREPQUEST_CGROUP:-}"
run_uid="${GREPQUEST_FIRECRACKER_UID:-}"
run_gid="${GREPQUEST_FIRECRACKER_GID:-}"

check_root_owned_path "firecracker binary" "$firecracker_binary" executable
check_root_owned_path "jailer binary" "$jailer_binary" executable

if [[ -n "$firecracker_binary" && -n "$jailer_binary" ]]; then
  firecracker_version="$("$firecracker_binary" --version 2>/dev/null | grep -Eo '[0-9]+\.[0-9]+\.[0-9]+' | head -n1 || true)"
  jailer_version="$("$jailer_binary" --version 2>/dev/null | grep -Eo '[0-9]+\.[0-9]+\.[0-9]+' | head -n1 || true)"
  if [[ -n "$firecracker_version" && "$firecracker_version" == "$jailer_version" ]]; then
    pass "matching binaries"
  else
    fail "matching binaries" "Firecracker and jailer versions differ or are unreadable"
  fi
fi

if [[ -c /dev/kvm && -r /dev/kvm && -w /dev/kvm ]]; then
  pass "/dev/kvm"
else
  fail "/dev/kvm" "must be readable and writable"
fi

if [[ -z "$jail_base" ]]; then
  fail "jail base" "set GREPQUEST_JAIL_BASE"
else
  check_root_owned_path "jail base" "$jail_base" directory
fi

if [[ -z "$run_uid" || ! "$run_uid" =~ ^[0-9]+$ || "$run_uid" == "0" ]] || ! getent passwd "$run_uid" >/dev/null; then
  fail "dedicated uid" "set GREPQUEST_FIRECRACKER_UID to an existing non-root UID"
else
  pass "dedicated uid"
fi

if [[ -z "$run_gid" || ! "$run_gid" =~ ^[0-9]+$ || "$run_gid" == "0" ]] || ! getent group "$run_gid" >/dev/null; then
  fail "dedicated gid" "set GREPQUEST_FIRECRACKER_GID to an existing non-root GID"
else
  pass "dedicated gid"
fi

if [[ ! -f /sys/fs/cgroup/cgroup.controllers ]]; then
  fail "cgroup version" "cgroup v2 unified hierarchy is required"
elif [[ -z "$cgroup_relative" || "$cgroup_relative" == /* || "$cgroup_relative" == *..* ]]; then
  fail "cgroup path" "set GREPQUEST_CGROUP to a safe relative cgroup-v2 path"
else
  cgroup_path="/sys/fs/cgroup/$cgroup_relative"
  if [[ ! -d "$cgroup_path" ]]; then
    fail "cgroup path" "missing: $cgroup_path"
  else
    check_root_owned_path "cgroup path" "$cgroup_path" directory
    for limit_file in memory.max pids.max cpu.max; do
      if [[ ! -r "$cgroup_path/$limit_file" ]]; then
        fail "$limit_file" "missing or unreadable"
        continue
      fi
      limit_value="$(<"$cgroup_path/$limit_file")"
      if [[ "$limit_value" == "max" || "$limit_value" == "max "* ]]; then
        fail "$limit_file" "must be explicitly bounded"
      else
        pass "$limit_file"
      fi
    done
  fi
fi

if [[ "$(wc -l < /proc/swaps)" -le 1 ]]; then
  pass "swap"
else
  fail "swap" "disable swap to avoid guest-memory remanence"
fi

if [[ "$failed" != "0" ]]; then
  printf '\nProduction Firecracker host audit failed. No host settings were changed.\n' >&2
  exit 1
fi

printf '\nProduction Firecracker host prerequisites passed. This audit does not launch a VM.\n'
