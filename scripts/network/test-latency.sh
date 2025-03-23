#!/usr/bin/env bash

source "$(dirname "$(dirname "$(dirname "$0")")")/lib/common.sh"

check_dependency "ping"

# Parâmetros
host="$1"
count="${2:-5}"

# Validações
if [ -z "$host" ]; then
    log_error "Uso: opsmaster network test-latency <host> [contagem]"
    exit 1
fi

log_info "Testando latência para $host ($count pings)..."
ping -c "$count" "$host" 