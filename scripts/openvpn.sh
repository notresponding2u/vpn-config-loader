#!/usr/bin/env bash

set -Eeuo pipefail

# ============================================================
# Configuration
# ============================================================

VPN_DIR="${VPN_DIR:-/vpn}"
VPN_CONFIG="${VPN_CONFIG:-/vpn/conf/config.ovpn}"

VPN_GROUP="${VPN_GROUP:-vpn}"

# LAN networks allowed to access published application ports.
# Supports multiple CIDRs separated by spaces.
LAN_CIDRS="${LAN_CIDRS:-192.168.0.0/16}"

# Docker network subnet.
DOCKER_SUBNET="${DOCKER_SUBNET:-172.30.0.0/16}"

# Application ports inside the shared network namespace.
# Supports comma-separated ports (iptables multiport syntax).

ALL_FIREWALL_ALLOWED_PORTS="${WEB_PORT:-}"

if [[ -n "${FIREWALL_ALLOWED_PORTS:-}" ]]; then

    if [[ -n "$ALL_FIREWALL_ALLOWED_PORTS" ]]; then
        ALL_FIREWALL_ALLOWED_PORTS+=","
    fi

    ALL_FIREWALL_ALLOWED_PORTS+="${FIREWALL_ALLOWED_PORTS}"
fi

FIREWALL_ALLOWED_PORTS="$ALL_FIREWALL_ALLOWED_PORTS"

[[ -n "$FIREWALL_ALLOWED_PORTS" ]] ||
    die "No firewall application ports configured"

IFS=',' read -ra PORTS <<< "$FIREWALL_ALLOWED_PORTS"

for port in "${PORTS[@]}"; do
    [[ "$port" =~ ^[0-9]+$ ]] ||
        die "Invalid firewall port: $port"

    (( port >= 1 && port <= 65535 )) ||
        die "Firewall port out of range: $port"
done

((${#PORTS[@]} <= 15)) ||
    die "iptables multiport supports at most 15 ports"

# OpenVPN management interface.
VPN_MANAGEMENT_HOST="${VPN_MANAGEMENT_HOST:-127.0.0.1}"
VPN_MANAGEMENT_PORT="${VPN_MANAGEMENT_PORT:-6061}"

VPN_AUTH=""

# ============================================================
# Logging
# ============================================================

log() {
    echo "[openvpn] $*"
}

die() {
    echo "[openvpn] ERROR: $*" >&2
    exit 1
}

cleanup() {
    if [[ -n "${VPN_AUTH:-}" ]]; then
        rm -f "$VPN_AUTH"
    fi
}

trap cleanup EXIT

# ============================================================
# Validate configuration
# ============================================================

: "${VPN_CONFIG_USERNAME:?VPN username not present}"
: "${VPN_CONFIG_PASSWORD:?VPN password not present}"

[[ -f "$VPN_CONFIG" ]] ||
    die "VPN configuration not found: $VPN_CONFIG"

[[ -n "$FIREWALL_ALLOWED_PORTS" ]] ||
    die "FIREWALL_ALLOWED_PORTS cannot be empty"

# ============================================================
# Dependencies
# ============================================================

for cmd in iptables ip ip6tables openvpn su-exec; do
    command -v "$cmd" >/dev/null 2>&1 ||
        die "Required command not found: $cmd"
done

# ============================================================
# VPN authentication
# ============================================================

create_vpn_auth() {

    VPN_AUTH="$(mktemp)"

    chmod 600 "$VPN_AUTH"

    printf '%s\n' \
        "$VPN_CONFIG_USERNAME" \
        "$VPN_CONFIG_PASSWORD" > "$VPN_AUTH"

    log "VPN authentication file created"
}

# ============================================================
# VPN group
# ============================================================

ensure_vpn_group() {

    if ! grep -q "^${VPN_GROUP}:" /etc/group; then
        addgroup -S "$VPN_GROUP"
    fi

    VPN_GID="$(awk -F: -v group="$VPN_GROUP" \
        '$1 == group { print $3 }' /etc/group)"

    [[ -n "$VPN_GID" ]] ||
        die "Unable to determine VPN group ID"

    log "VPN group: ${VPN_GROUP} (GID ${VPN_GID})"
}

# ============================================================
# LAN routing
# ============================================================

configure_lan_routes() {

    log "Configuring LAN routes"

    LAN_GATEWAY="$(ip -4 route show default dev eth0 |
        awk 'NR == 1 { print $3 }')"

    [[ -n "$LAN_GATEWAY" ]] ||
        die "Unable to determine eth0 gateway"

    log "Docker gateway: ${LAN_GATEWAY}"

    for network in $LAN_CIDRS; do

        ip route replace "$network" \
            via "$LAN_GATEWAY" \
            dev eth0

        log "LAN route: ${network} via ${LAN_GATEWAY}"

    done
}

# ============================================================
# IPv4 firewall
# ============================================================

configure_ipv4_firewall() {

    log "Configuring IPv4 firewall"

    # Flush existing filter rules.
    iptables -w -F
    iptables -w -X

    # Default deny.
    iptables -w -P INPUT DROP
    iptables -w -P FORWARD DROP
    iptables -w -P OUTPUT DROP

    # --------------------------------------------------------
    # INPUT
    # --------------------------------------------------------

    # Loopback.
    iptables -w -A INPUT \
        -i lo \
        -j ACCEPT

    # Established and related connections.
    iptables -w -A INPUT \
        -m conntrack \
        --ctstate ESTABLISHED,RELATED \
        -j ACCEPT

    # Allow LAN and Docker networks to access application ports.
    for network in $LAN_CIDRS "$DOCKER_SUBNET"; do

        iptables -w -A INPUT \
            -i eth0 \
            -s "$network" \
            -p tcp \
            -m multiport \
            --dports "$FIREWALL_ALLOWED_PORTS" \
            -m conntrack \
            --ctstate NEW \
            -j ACCEPT

    done

    # --------------------------------------------------------
    # OUTPUT
    # --------------------------------------------------------

    # Loopback.
    iptables -w -A OUTPUT \
        -o lo \
        -j ACCEPT

    # Applications can communicate through VPN interfaces.
    iptables -w -A OUTPUT \
        -o tun+ \
        -j ACCEPT

    iptables -w -A OUTPUT \
        -o tap+ \
        -j ACCEPT

    # OpenVPN process is allowed to establish its external
    # connections through the physical Docker interface.
    iptables -w -A OUTPUT \
        -m owner \
        --gid-owner "$VPN_GID" \
        -j ACCEPT

    # Allow established responses to LAN and Docker clients.
    for network in $LAN_CIDRS "$DOCKER_SUBNET"; do

        iptables -w -A OUTPUT \
            -o eth0 \
            -d "$network" \
            -m conntrack \
            --ctstate ESTABLISHED,RELATED \
            -j ACCEPT

    done

    # Docker embedded DNS resolver.
    iptables -w -A OUTPUT \
        -d 127.0.0.11 \
        -p udp \
        --dport 53 \
        -j ACCEPT

    iptables -w -A OUTPUT \
        -d 127.0.0.11 \
        -p tcp \
        --dport 53 \
        -j ACCEPT

    log "IPv4 firewall configured"
}

# ============================================================
# IPv6 firewall
# ============================================================

configure_ipv6_firewall() {

    log "Configuring IPv6 firewall"

    # Block IPv6 leaks outside the VPN.
    ip6tables -w -F
    ip6tables -w -X

    ip6tables -w -P INPUT DROP
    ip6tables -w -P FORWARD DROP
    ip6tables -w -P OUTPUT DROP

    # Loopback.
    ip6tables -w -A INPUT \
        -i lo \
        -j ACCEPT

    ip6tables -w -A OUTPUT \
        -o lo \
        -j ACCEPT

    # Established connections.
    ip6tables -w -A INPUT \
        -m conntrack \
        --ctstate ESTABLISHED,RELATED \
        -j ACCEPT

    # VPN tunnel interfaces.
    ip6tables -w -A OUTPUT \
        -o tun+ \
        -j ACCEPT

    ip6tables -w -A OUTPUT \
        -o tap+ \
        -j ACCEPT

    # OpenVPN process.
    ip6tables -w -A OUTPUT \
        -m owner \
        --gid-owner "$VPN_GID" \
        -j ACCEPT

    log "IPv6 firewall configured"
}

# ============================================================
# Firewall initialization
# ============================================================

configure_firewall() {

    rm -f /run/vpnloader-firewall-ready

    ensure_vpn_group

    # Preserve LAN routing before OpenVPN changes routes.
    configure_lan_routes

    configure_ipv4_firewall
    configure_ipv6_firewall

    touch /run/vpnloader-firewall-ready

    log "Firewall initialization completed"
}

# ============================================================
# OpenVPN startup
# ============================================================

start_openvpn() {

    [[ -f "$VPN_CONFIG" ]] ||
        die "VPN configuration not found: $VPN_CONFIG"

    create_vpn_auth

    log "Starting OpenVPN"
    log "Configuration: $VPN_CONFIG"
    log "Management interface: ${VPN_MANAGEMENT_HOST}:${VPN_MANAGEMENT_PORT}"

    log "LAN networks: ${LAN_CIDRS}"
    log "Docker subnet: ${DOCKER_SUBNET}"
    log "Application ports: ${FIREWALL_ALLOWED_PORTS}"

    # OpenVPN runs as root with VPN as its primary group.
    # The firewall owner rule allows its packets through eth0.

    exec su-exec "root:${VPN_GROUP}" \
        openvpn \
            --cd "$VPN_DIR" \
            --config "$VPN_CONFIG" \
            --auth-user-pass "$VPN_AUTH" \
            --auth-nocache \
            --script-security 2 \
            --management "$VPN_MANAGEMENT_HOST" "$VPN_MANAGEMENT_PORT" \
            --verb 3
}

# ============================================================
# Main
# ============================================================

case "${1:-start}" in

    --setup-firewall)
        configure_firewall
        exit 0
        ;;

    start)
        configure_firewall
        start_openvpn
        ;;

    start-no-firewall)
        ensure_vpn_group
        start_openvpn
        ;;

    *)
        die "Unknown argument: $1"
        ;;

esac

