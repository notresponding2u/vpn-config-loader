
#!/usr/bin/env bash

set -Eeuo pipefail

# ============================================================
# Configuration
# ============================================================

VPN_DIR="/vpn"

VPN_CONFIG="${VPN_DIR}/config.ovpn"

[[ -n "$VPN_CONFIG_USERNAME" ]] || die "VPN username not present"

[[ -n "$VPN_CONFIG_PASSWORD" ]] || die "VPN password not present"

VPN_AUTH="$(mktemp)"

chmod 600 "$VPN_AUTH"

printf '%s\n' "$VPN_CONFIG_USERNAME" "$VPN_CONFIG_PASSWORD" > "$VPN_AUTH"

VPN_GROUP="vpn"

LAN_CIDRS="${LAN_CIDRS:-192.168.0.0/16 172.16.0.0/12}"
DOCKER_SUBNET="${DOCKER_SUBNET:-172.30.0.0/16}"

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

# ============================================================
# Dependencies
# ============================================================

for cmd in iptables ip ip6tables openvpn su-exec; do
    command -v "$cmd" >/dev/null 2>&1 ||
        die "Required command not found: $cmd"
done

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
# IPv4 firewall
# ============================================================

configure_ipv4_firewall() {

    log "Configuring IPv4 firewall"

    # Flush filter rules.
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
    iptables -w -A INPUT -i lo -j ACCEPT

    # Established and related connections.
    iptables -w -A INPUT \
        -m conntrack \
        --ctstate ESTABLISHED,RELATED \
        -j ACCEPT

    # Permit LAN access to application ports.
    for network in $LAN_CIDRS "$DOCKER_SUBNET"; do

        iptables -w -A INPUT \
            -i eth0 \
            -s "$network" \
            -p tcp \
            -m multiport \
            --dports "$WEB_PORT" \
            -j ACCEPT

    done

    # --------------------------------------------------------
    # OUTPUT
    # --------------------------------------------------------

    # Loopback: Go application <-> OpenVPN management.
    iptables -w -A OUTPUT -o lo -j ACCEPT

    # All services may use VPN tunnel interfaces.
    iptables -w -A OUTPUT -o tun+ -j ACCEPT
    iptables -w -A OUTPUT -o tap+ -j ACCEPT

    # OpenVPN is the ONLY process group allowed to
    # establish unrestricted external connections.
    iptables -w -A OUTPUT \
        -m owner \
        --gid-owner "$VPN_GID" \
        -j ACCEPT

    # Permit established replies to LAN/Docker connections.
    # Do not permit applications to initiate arbitrary
    # outbound connections through eth0.
    for network in $LAN_CIDRS "$DOCKER_SUBNET"; do

        iptables -w -A OUTPUT \
            -o eth0 \
            -d "$network" \
            -m conntrack \
            --ctstate ESTABLISHED,RELATED \
            -j ACCEPT

    done

    # Docker embedded DNS resolver.
    # DNS requests to 127.0.0.11 are local to the namespace.
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
    ip6tables -w -A INPUT -i lo -j ACCEPT
    ip6tables -w -A OUTPUT -o lo -j ACCEPT

    # Established connections.
    ip6tables -w -A INPUT \
        -m conntrack \
        --ctstate ESTABLISHED,RELATED \
        -j ACCEPT

    # VPN tunnel interfaces.
    ip6tables -w -A OUTPUT -o tun+ -j ACCEPT
    ip6tables -w -A OUTPUT -o tap+ -j ACCEPT

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

    [[ -f "$VPN_AUTH" ]] ||
        die "VPN authentication file not found: $VPN_AUTH"

    log "Starting OpenVPN"
    log "Configuration: $VPN_CONFIG"
    log "Management interface: 127.0.0.1:6061"

    # OpenVPN runs as root with VPN as its primary group.
    # This makes the firewall owner rule match its packets.
    #
    # su-exec replaces itself with OpenVPN.
    exec su-exec "root:${VPN_GROUP}" \
        openvpn \
            --cd "$VPN_DIR" \
            --config "$VPN_CONFIG" \
            --auth-user-pass $VPN_AUTH \
            --auth-nocache \
            --script-security 2 \
            --management 127.0.0.1 6061 \
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

