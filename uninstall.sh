#!/bin/bash

# Diretório base do projeto
BASE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Importa funções comuns
source "${BASE_DIR}/lib/common.sh"

# Verifica se está rodando como root
if [ "$EUID" -ne 0 ]; then 
    log_error "Por favor, execute o script como root (sudo ./uninstall.sh)"
    exit 1
fi

# Remove o executável
if [ -f "/usr/local/bin/opsmaster" ]; then
    log_info "Removendo executável do OPSMaster..."
    rm -f /usr/local/bin/opsmaster
fi

# Remove diretório de configuração
if [ -d "/etc/opsmaster" ]; then
    log_info "Removendo diretório de configuração..."
    rm -rf /etc/opsmaster
fi

# Remove logs (se existirem)
if [ -d "/var/log/opsmaster" ]; then
    log_info "Removendo logs..."
    rm -rf /var/log/opsmaster
fi

log_info "OPSMaster foi desinstalado com sucesso!"
log_warn "Os arquivos do diretório atual não foram removidos."
log_info "Para remover completamente, você pode deletar este diretório." 