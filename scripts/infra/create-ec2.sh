#!/usr/bin/env bash

source "$(dirname "$(dirname "$(dirname "$0")")")/lib/common.sh"

check_dependency "aws"

# Parâmetros
instance_type="$1"
region="$2"

# Validações
if [ -z "$instance_type" ] || [ -z "$region" ]; then
    log_error "Uso: opsmaster infra create-ec2 <instance-type> <region>"
    exit 1
fi

log_info "Criando instância EC2 do tipo $instance_type na região $region..."
# aws ec2 run-instances ... (comando aws cli aqui) 