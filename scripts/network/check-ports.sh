#!/usr/bin/env bash

source "$(dirname "$(dirname "$(dirname "$0")")")/lib/common.sh"

check_dependency "nc"

# Parâmetros
host="$1"
port="$2"

# Validações
if [ -z "$host" ] || [ -z "$port" ]; then
    log_error "Uso: opsmaster network check-ports <host> <porta>"
    exit 1
fi

log_info "Verificando conectividade com $host:$port..."
if nc -zv "$host" "$port" 2>&1; then
    log_info "Porta $port está aberta em $host"
else
    log_error "Porta $port está fechada em $host"
fi 