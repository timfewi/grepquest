#!/bin/sh
set -eu

export PATH=/bin
mount -t proc -o nosuid,nodev,noexec proc /proc
mount -t sysfs -o nosuid,nodev,noexec,ro sysfs /sys
mount -t devtmpfs -o nosuid,noexec devtmpfs /dev
mount -t tmpfs -o nosuid,nodev,noexec,size=64m,mode=0755 tmpfs /run
mkdir -p /run/grepquest/player
chown 1000:1000 /run/grepquest/player
# Inherited by the agent and its UID-1000 grep child. The single-command
# parser already prevents process-spawning shell syntax; these are defense in
# depth for file descriptors and per-user processes.
ulimit -n 64
ulimit -u 32
exec /bin/grepquest-agent
