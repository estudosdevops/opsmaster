#!/usr/bin/env bash

# Cores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Função para logs
log_info() {
    echo -e "${GREEN}[INFO]${NC} ✓ $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} ⚠️ $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} ✗ $1"
}

# Função para validar dependências
check_dependency() {
    local missing_deps=()
    
    for dep in "$@"; do
        if ! command -v "$dep" &> /dev/null; then
            missing_deps+=("$dep")
        fi
    done
    
    if [ ${#missing_deps[@]} -ne 0 ]; then
        log_error "Dependências não encontradas: ${missing_deps[*]}"
        log_info "Por favor, instale as dependências necessárias e tente novamente"
        exit 1
    fi
    
    log_info "Todas as dependências encontradas: $*"
}

# Função para validar variáveis de ambiente
check_env_var() {
    if [ -z "${!1}" ]; then
        log_error "Variável de ambiente '$1' não definida"
        exit 1
    fi
} 