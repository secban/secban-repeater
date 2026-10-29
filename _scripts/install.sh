#!/bin/bash
set -ex

OWNER=$(whoami)
if [ "$OWNER" != "root" ]; then
    echo "This script must be run as root"
    exit 1
fi

OWN_DIR=$(dirname "$0")
cd "$OWN_DIR"

if ! getent group secban >/dev/null 2>&1; then
    groupadd --system secban
fi

if ! id -u secban >/dev/null 2>&1; then
    useradd --no-create-home --system --shell /sbin/nologin --gid secban secban
fi

cp "./secban-repeater" "/usr/sbin/"
chmod +x "/usr/sbin/secban-repeater"

# Install the example agent configuration.
cp -ar "./etc/secban/" "/etc/secban/"

mkdir -p /var/run/secban
chown secban:secban /var/run/secban
chmod 0770 /var/run/secban

# Systemd
if [ -d /etc/systemd/system ]; then
    if [ ! -f /etc/systemd/system/secban-repeater.service ]; then
        cp "./secban-repeater.service" "/etc/systemd/system/secban-repeater.service"
    fi
    if [ -n "$(which systemctl 2>/dev/null)" ]; then
        systemctl daemon-reload
        systemctl enable secban-repeater.service
    fi
fi

# OpenRC
if [ -d /etc/init.d ] && [ -n "$(which rc-update 2>/dev/null)" ]; then
    cp "./secban-repeater.rc" "/etc/init.d/secban-repeater"
    chmod +x "/etc/init.d/secban-repeater"
    rc-update add secban-repeater default || true
fi
