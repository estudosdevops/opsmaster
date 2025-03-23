#!/usr/bin/env bash
# Description: Realiza backup de arquivos para Amazon S3

source "/usr/local/lib/opsmaster/common.sh"

check_dependency "aws"

# Parâmetros
source_dir="$1"
bucket_name="$2"

# Validações
if [ -z "$source_dir" ] || [ -z "$bucket_name" ]; then
    log_error "Uso: opsmaster backup s3-backup <diretório-origem> <nome-bucket>"
    exit 1
fi

log_info "Iniciando backup do diretório $source_dir para S3://$bucket_name"
aws s3 sync "$source_dir" "s3://$bucket_name/$(date +%Y-%m-%d)" 