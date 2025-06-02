#!/usr/bin/env bash

# Funções utilitárias para os testes

# Verificar se um comando existe
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Verificar se um arquivo existe
file_exists() {
    [ -f "$1" ]
}

# Verificar se um diretório existe
dir_exists() {
    [ -d "$1" ]
}

# Verificar se um arquivo contém uma string
file_contains() {
    local file="$1"
    local string="$2"
    grep -q "$string" "$file"
}

# Verificar se um arquivo tem permissões corretas
file_has_permissions() {
    local file="$1"
    local permissions="$2"
    [ "$(stat -c "%a" "$file")" = "$permissions" ]
}

# Verificar se um arquivo tem o tamanho esperado
file_has_size() {
    local file="$1"
    local size="$2"
    [ "$(stat -c "%s" "$file")" = "$size" ]
}

# Verificar se um arquivo foi modificado recentemente
file_was_modified_recently() {
    local file="$1"
    local seconds="$2"
    local current_time=$(date +%s)
    local file_time=$(stat -c "%Y" "$file")
    local time_diff=$((current_time - file_time))
    [ "$time_diff" -le "$seconds" ]
}

# Verificar se um processo está rodando
process_is_running() {
    local process="$1"
    pgrep -f "$process" >/dev/null
}

# Verificar se uma porta está em uso
port_is_in_use() {
    local port="$1"
    netstat -tuln | grep -q ":$port "
}

# Verificar se um serviço está respondendo
service_is_responding() {
    local host="$1"
    local port="$2"
    nc -z "$host" "$port" >/dev/null 2>&1
}

# Verificar se um banco de dados existe
database_exists() {
    local host="$1"
    local port="$2"
    local user="$3"
    local password="$4"
    local database="$5"
    
    export PGPASSWORD="$password"
    psql -h "$host" -p "$port" -U "$user" -lqt | cut -d \| -f 1 | grep -qw "$database"
}

# Verificar se uma tabela existe
table_exists() {
    local host="$1"
    local port="$2"
    local user="$3"
    local password="$4"
    local database="$5"
    local table="$6"
    
    export PGPASSWORD="$password"
    psql -h "$host" -p "$port" -U "$user" -d "$database" -c "\dt" | grep -qw "$table"
}

# Verificar se um backup é válido
backup_is_valid() {
    local backup_file="$1"
    local format="$2"
    
    case "$format" in
        "postgresql")
            pg_restore -l "$backup_file" >/dev/null 2>&1
            ;;
        "mongodb")
            bsondump "$backup_file" >/dev/null 2>&1
            ;;
        *)
            return 1
            ;;
    esac
}

# Verificar se um arquivo de configuração é válido
config_is_valid() {
    local config_file="$1"
    yq eval "$config_file" >/dev/null 2>&1
}

# Verificar se um arquivo de log contém uma mensagem
log_contains() {
    local log_file="$1"
    local message="$2"
    grep -q "$message" "$log_file"
}

# Verificar se um arquivo de log não contém uma mensagem
log_not_contains() {
    local log_file="$1"
    local message="$2"
    ! grep -q "$message" "$log_file"
}

# Verificar se um arquivo de log contém um erro
log_contains_error() {
    local log_file="$1"
    grep -q "ERROR" "$log_file"
}

# Verificar se um arquivo de log contém um aviso
log_contains_warning() {
    local log_file="$1"
    grep -q "WARNING" "$log_file"
}

# Verificar se um arquivo de log contém uma informação
log_contains_info() {
    local log_file="$1"
    grep -q "INFO" "$log_file"
}

# Verificar se um arquivo de log contém um debug
log_contains_debug() {
    local log_file="$1"
    grep -q "DEBUG" "$log_file"
}

# Verificar se um arquivo de log contém um trace
log_contains_trace() {
    local log_file="$1"
    grep -q "TRACE" "$log_file"
}

# Verificar se um arquivo de log contém um fatal
log_contains_fatal() {
    local log_file="$1"
    grep -q "FATAL" "$log_file"
}

# Verificar se um arquivo de log contém um panic
log_contains_panic() {
    local log_file="$1"
    grep -q "PANIC" "$log_file"
}

# Verificar se um arquivo de log contém um critical
log_contains_critical() {
    local log_file="$1"
    grep -q "CRITICAL" "$log_file"
}

# Verificar se um arquivo de log contém um alert
log_contains_alert() {
    local log_file="$1"
    grep -q "ALERT" "$log_file"
}

# Verificar se um arquivo de log contém um emergency
log_contains_emergency() {
    local log_file="$1"
    grep -q "EMERGENCY" "$log_file"
}

# Verificar se um arquivo de log contém um notice
log_contains_notice() {
    local log_file="$1"
    grep -q "NOTICE" "$log_file"
} 