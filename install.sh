#!/usr/bin/env bash

# Diretório base do projeto
BASE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Importa funções comuns
source "${BASE_DIR}/lib/common.sh"

# Verifica se está rodando como root
if [ "$EUID" -ne 0 ]; then 
    log_error "Por favor, execute o script como root (sudo ./install.sh)"
    exit 1
fi

# Configura permissões de execução
log_info "Configurando permissões de execução..."
chmod +x "${BASE_DIR}/bin/opsmaster"
find "${BASE_DIR}/scripts" -type f -name "*.sh" -exec chmod +x {} \;

# Cria diretórios necessários
log_info "Criando diretórios de sistema..."
mkdir -p /usr/local/lib/opsmaster
mkdir -p /etc/opsmaster
mkdir -p /var/log/opsmaster

# Copia executável para o path do sistema
log_info "Instalando OPSMaster..."
cp "${BASE_DIR}/bin/opsmaster" /usr/local/bin/
chmod 755 /usr/local/bin/opsmaster

# Copia biblioteca comum
log_info "Instalando biblioteca comum..."
cp "${BASE_DIR}/lib/common.sh" /usr/local/lib/opsmaster/
chmod 644 /usr/local/lib/opsmaster/common.sh

# Copia scripts
log_info "Copiando scripts..."
cp -r "${BASE_DIR}/scripts" /etc/opsmaster/

# Configura permissões dos diretórios
log_info "Configurando permissões..."
chown -R root:root /etc/opsmaster /usr/local/lib/opsmaster
chmod -R 755 /etc/opsmaster
chmod -R 755 /usr/local/lib/opsmaster

log_info "OPSMaster foi instalado com sucesso!"
log_info "Execute 'opsmaster --version' para verificar a instalação." 