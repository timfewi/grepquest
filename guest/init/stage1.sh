#!/bin/sh
set -eu

export PATH=/bin
mount -t devtmpfs devtmpfs /dev
mount -t proc proc /proc
mount -t sysfs sysfs /sys
mkdir -p /newroot
mount -t ext4 -o ro /dev/vda /newroot
exec switch_root /newroot /sbin/init
