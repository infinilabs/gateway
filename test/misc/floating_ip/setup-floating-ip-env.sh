#!/bin/bash
# setup-floating-ip-env.sh
# Detect physical network interface and output floating IP to stdout.
# Usage: FLOAT_IP=$(./setup-floating-ip-env.sh)

set -e

log() { echo "$@" >&2; }

detect_os() {
    case "$(uname -s)" in
        Darwin*) echo "macos" ;;
        Linux*)  echo "linux" ;;
        *)       echo "unknown" ;;
    esac
}

OS=$(detect_os)

collect_ifaces_macos() {
    ifconfig | awk '
        /^[a-z][a-z0-9]+:/ {
            iface = $1
            sub(/:$/, "", iface)
            next
        }
        /inet [0-9]/ && iface {
            ip = $2
            if (ip !~ /^127\./ && ip !~ /^0\./) {
                printf "%s %s\n", iface, ip
                iface = ""
            }
        }
    ' | sort -u
}

collect_ifaces_linux() {
    ip -4 -o addr show | awk '{print $2, $4}' | cut -d/ -f1 | awk '$2 !~ /^127\./'
}

# Pick best: en* (mac) / eth*|ens*|enp* (linux), then anything else
pick_primary_iface() {
    echo "$1" | awk '{
        if ($1 ~ /^en[0-9]/)        print "1 " $0
        else if ($1 ~ /^eth[0-9]/)  print "2 " $0
        else if ($1 ~ /^ens[0-9]/)  print "2 " $0
        else if ($1 ~ /^enp[0-9]s/) print "2 " $0
        else                         print "9 " $0
    }' | sort | head -1 | cut -d' ' -f2-
}

if [[ "$OS" == "macos" ]]; then
    CANDIDATES=$(collect_ifaces_macos)
else
    CANDIDATES=$(collect_ifaces_linux)
fi

if [ -z "$CANDIDATES" ]; then
    log "ERROR: No non-loopback IPv4 interface found"
    if [[ "$OS" == "macos" ]]; then
        ifconfig >&2
    else
        ip -4 addr show >&2
    fi
    exit 1
fi

PRIMARY_LINE=$(pick_primary_iface "$CANDIDATES")
LOCAL_IP=$(echo "$PRIMARY_LINE" | awk '{print $2}')

FLOATING_IP=$(echo "$LOCAL_IP" | awk -F. '{print $1"."$2"."$3".234"}')

printf '%s' "$FLOATING_IP"