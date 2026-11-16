#!/bin/sh

cp /app/conf.d/${VPN_CONFIG_FILE} /vpn/conf/config.ovpn

set -eu

echo "[entrypoint] Installing firewall"

/bin/bash /usr/local/bin/openvpn.sh --setup-firewall

echo "[entrypoint] Starting Go application"

/vpn/srv &
APP_PID=$!

sleep 2

if ! kill -0 "$APP_PID" 2>/dev/null; then
    echo "[entrypoint] ERROR: Go application exited"
    wait "$APP_PID"
    exit 1
fi

echo "[entrypoint] Go application PID: $APP_PID"

echo "[entrypoint] Starting OpenVPN"

exec /bin/bash /usr/local/bin/openvpn.sh start-no-firewall

