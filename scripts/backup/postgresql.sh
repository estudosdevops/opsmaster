#!/usr/bin/env bash
# Description: Gerencia backup e restore de bancos PostgreSQL

# shellcheck disable=SC1091
source "/usr/local/lib/opsmaster/common.sh"

# Constantes
readonly DEFAULT_BACKUP_DIR="/tmp/postgresql_backups"
readonly DEFAULT_CONFIG_FILE="$HOME/.config/backups/postgresql.yaml"
readonly DEFAULT_PORT="5432"
readonly DEFAULT_HOST="localhost"
readonly DATE_FORMAT=$(date +%Y%m%d_%H%M%S)

# Configurações
BACKUP_DIR="${BACKUP_DIR:-$DEFAULT_BACKUP_DIR}"
CONFIG_FILE="${CONFIG_FILE:-$DEFAULT_CONFIG_FILE}"

# Funções de utilidade
validate_port() {
    local port="$1"
    if ! [[ "$port" =~ ^[0-9]+$ ]] || [ "$port" -lt 1 ] || [ "$port" -gt 65535 ]; then
        log_error "Porta inválida: $port"
        return 1
    fi
    return 0
}

validate_required_field() {
    local field_name="$1"
    local field_value="$2"
    local config_type="$3"

    if [ -z "$field_value" ]; then
        log_error "$field_name não especificado para $config_type"
        return 1
    fi
    return 0
}

# Função para validar configurações
validate_config() {
    local config_type="$1"  # source ou target
    local host="$2"
    local port="$3"
    local database="$4"
    local username="$5"
    local password="$6"

    validate_required_field "Host" "$host" "$config_type" || return 1
    validate_port "$port" || return 1
    validate_required_field "Banco de dados" "$database" "$config_type" || return 1
    validate_required_field "Usuário" "$username" "$config_type" || return 1
    validate_required_field "Senha" "$password" "$config_type" || return 1

    return 0
}

# Função para criar arquivo de configuração inicial
create_config() {
    local config_file="$1"
    local config_dir
    config_dir=$(dirname "$config_file")

    # Verificar se o diretório existe
    if [ ! -d "$config_dir" ]; then
        log_info "Criando diretório de configuração: $config_dir"
        mkdir -p "$config_dir"
    fi

    # Verificar se o arquivo já existe
    if [ -f "$config_file" ]; then
        log_warn "Arquivo de configuração já existe: $config_file"
        read -p "Deseja sobrescrever? (s/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Ss]$ ]]; then
            return 1
        fi
    fi

    # Criar arquivo de configuração
    log_info "Criando arquivo de configuração: $config_file"
    cat > "$config_file" << EOF
# Configurações do PostgreSQL
# Este arquivo contém as configurações para backup e restore de bancos PostgreSQL

# Configurações do banco de origem
source:
  host: $DEFAULT_HOST
  port: $DEFAULT_PORT
  database: sourcedb
  username: postgres
  password: sourcepass123

# Configurações do banco de destino
target:
  host: $DEFAULT_HOST
  port: $DEFAULT_PORT
  database: targetdb
  username: postgres
  password: targetpass123

# Diretório para armazenar os backups
backup_dir: $HOME/backups/postgresql
EOF

    # Ajustar permissões
    chmod 600 "$config_file"

    log_info "Arquivo de configuração criado com sucesso"
    log_info "Edite o arquivo $config_file para configurar suas conexões"
}

# Função para ler configurações do arquivo YAML
read_yaml_config() {
    local config_file="$1"
    
    # Verificar se o arquivo existe
    if [ ! -f "$config_file" ]; then
        log_warn "Arquivo de configuração não encontrado: $config_file"
        read -p "Deseja criar um novo arquivo de configuração? (s/N): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Ss]$ ]]; then
            create_config "$config_file"
        else
            return 1
        fi
    fi

    # Verificar se o yq está instalado
    if ! command -v yq &> /dev/null; then
        log_error "yq não está instalado. Execute 'opsmaster backup postgresql install-deps' para instalar"
        return 1
    fi

    # Ler configurações do arquivo YAML
    SOURCE_HOST=$(yq eval '.source.host // "'$DEFAULT_HOST'"' "$config_file")
    SOURCE_PORT=$(yq eval '.source.port // "'$DEFAULT_PORT'"' "$config_file")
    SOURCE_DB=$(yq eval '.source.database // ""' "$config_file")
    SOURCE_USER=$(yq eval '.source.username // ""' "$config_file")
    SOURCE_PASS=$(yq eval '.source.password // ""' "$config_file")

    TARGET_HOST=$(yq eval '.target.host // "'$DEFAULT_HOST'"' "$config_file")
    TARGET_PORT=$(yq eval '.target.port // "'$DEFAULT_PORT'"' "$config_file")
    TARGET_DB=$(yq eval '.target.database // ""' "$config_file")
    TARGET_USER=$(yq eval '.target.username // ""' "$config_file")
    TARGET_PASS=$(yq eval '.target.password // ""' "$config_file")

    # Configurar diretório de backup se especificado
    local backup_dir
    backup_dir=$(yq eval '.backup_dir // ""' "$config_file")
    if [ -n "$backup_dir" ]; then
        BACKUP_DIR="$backup_dir"
    fi

    # Validar configurações
    if ! validate_config "source" "$SOURCE_HOST" "$SOURCE_PORT" "$SOURCE_DB" "$SOURCE_USER" "$SOURCE_PASS"; then
        return 1
    fi

    if ! validate_config "target" "$TARGET_HOST" "$TARGET_PORT" "$TARGET_DB" "$TARGET_USER" "$TARGET_PASS"; then
        return 1
    fi

    log_info "Configurações carregadas de: $config_file"
}

# Função para executar comando PostgreSQL
execute_postgresql_command() {
    local host="$1"
    local port="$2"
    local database="$3"
    local username="$4"
    local password="$5"
    local command="$6"

    export PGPASSWORD="$password"
    if psql -h "$host" -p "$port" -U "$username" -d "$database" -c "$command" >/dev/null 2>&1; then
        return 0
    else
        return 1
    fi
}

# Função para realizar dump
dump_postgresql() {
    local host="${SOURCE_HOST:-$DEFAULT_HOST}"
    local port="${SOURCE_PORT:-$DEFAULT_PORT}"
    local database="$SOURCE_DB"
    local username="$SOURCE_USER"
    local password="$SOURCE_PASS"
    local backup_file="${BACKUP_FILE:-$BACKUP_DIR/${database}_${DATE_FORMAT}.dump}"

    # Validar parâmetros
    if [ -z "$database" ] || [ -z "$username" ]; then
        log_error "Parâmetros obrigatórios não fornecidos"
        show_help
        return 1
    fi

    # Criar diretório de backup se não existir
    mkdir -p "$(dirname "$backup_file")"

    log_info "Iniciando dump do banco $database de $host:$port"

    # Configurar variável de ambiente para senha
    export PGPASSWORD="$password"

    # Realizar dump
    if pg_dump -h "$host" -p "$port" -U "$username" -d "$database" -F c -f "$backup_file"; then
        log_success "Dump concluído com sucesso: $backup_file"
        show_backup_info "$backup_file"
        return 0
    else
        log_error "Falha ao realizar dump"
        return 1
    fi
}

# Função para restaurar backup
restore_postgresql() {
    local host="${TARGET_HOST:-$DEFAULT_HOST}"
    local port="${TARGET_PORT:-$DEFAULT_PORT}"
    local database="$TARGET_DB"
    local username="$TARGET_USER"
    local password="$TARGET_PASS"
    local backup_file="$BACKUP_FILE"

    # Validar parâmetros
    if [ -z "$database" ] || [ -z "$username" ] || [ -z "$backup_file" ]; then
        log_error "Parâmetros obrigatórios não fornecidos"
        show_help
        return 1
    fi

    # Verificar se arquivo de backup existe
    if [ ! -f "$backup_file" ]; then
        log_error "Arquivo de backup não encontrado: $backup_file"
        return 1
    fi

    log_info "Iniciando restore do banco $database em $host:$port"

    # Configurar variável de ambiente para senha
    export PGPASSWORD="$password"

    # Encerrar conexões existentes
    execute_postgresql_command "$host" "$port" "postgres" "$username" "$password" \
        "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '$database';"

    # Dropar e recriar banco
    execute_postgresql_command "$host" "$port" "postgres" "$username" "$password" \
        "DROP DATABASE IF EXISTS $database;"
    execute_postgresql_command "$host" "$port" "postgres" "$username" "$password" \
        "CREATE DATABASE $database;"

    # Realizar restore
    if pg_restore -h "$host" -p "$port" -U "$username" -d "$database" "$backup_file"; then
        log_success "Restore concluído com sucesso"
        return 0
    else
        log_error "Falha ao realizar restore"
        return 1
    fi
}

# Função para listar backups
list_backups() {
    if [ ! -d "$BACKUP_DIR" ]; then
        log_info "Nenhum backup encontrado"
        return 0
    fi

    echo
    echo "Backups disponíveis:"
    echo "----------------------------------------"
    echo "Arquivo                           Tamanho    Data"
    echo "----------------------------------------"

    find "$BACKUP_DIR" -name "*.dump" -type f -printf "%f\t%s\t%Td/%Tm/%TY %TH:%TM\n" | while IFS=$'\t' read -r file size date; do
        size=$(numfmt --to=iec-i --suffix=B --format="%.1f" "$size")
        printf "%-30s %10s %s\n" "$file" "$size" "$date"
    done

    echo "----------------------------------------"
    local total_size
    total_size=$(du -sh "$BACKUP_DIR" 2>/dev/null | cut -f1)
    echo "Tamanho total dos backups: ${total_size:-0B}"
    echo
}

# Função para mostrar informações do backup
show_backup_info() {
    local backup_file="$1"
    local size
    size=$(du -h "$backup_file" 2>/dev/null | cut -f1)
    
    echo
    echo "Informações do backup:"
    echo "----------------------------------------"
    echo "Arquivo: $(basename "$backup_file")"
    echo "Tamanho: ${size:-0B}"
    echo "Data: $(date -r "$backup_file" "+%d/%m/%Y %H:%M:%S")"
    echo "----------------------------------------"
    echo
}

# Função para sincronizar banco de dados (dump + restore)
sync_postgresql() {
    log_info "Iniciando sincronização do banco $SOURCE_DB"
    log_info "Origem: $SOURCE_HOST:$SOURCE_PORT"
    log_info "Destino: $TARGET_HOST:$TARGET_PORT"

    # Realizar dump
    if ! dump_postgresql; then
        log_error "Falha ao realizar dump. Abortando sincronização."
        return 1
    fi

    # Realizar restore
    if ! restore_postgresql; then
        log_error "Falha ao realizar restore. Abortando sincronização."
        return 1
    fi

    log_success "Sincronização concluída com sucesso"
}

# Função para mostrar o menu de ajuda
show_help() {
    cat << EOF
Uso: opsmaster backup postgresql [opções] [comando]

Comandos:
  dump                    Realiza dump de um banco PostgreSQL
  restore                 Restaura um backup em um banco PostgreSQL
  sync                    Realiza dump e restore em uma única operação
  list                    Lista todos os backups disponíveis
  install-deps           Instala dependências necessárias
  init-config            Cria arquivo de configuração inicial

Opções:
  -h, --help             Mostra esta mensagem de ajuda
  -c, --config FILE      Arquivo de configuração YAML (padrão: $DEFAULT_CONFIG_FILE)
  --source-host HOST     Host do banco de origem (sobrescreve configuração)
  --source-port PORT     Porta do banco de origem (sobrescreve configuração)
  --source-db DB         Nome do banco de dados de origem (sobrescreve configuração)
  --source-user USER     Usuário do banco de origem (sobrescreve configuração)
  --source-pass PASS     Senha do banco de origem (sobrescreve configuração)
  --target-host HOST     Host do banco de destino (sobrescreve configuração)
  --target-port PORT     Porta do banco de destino (sobrescreve configuração)
  --target-db DB         Nome do banco de dados de destino (sobrescreve configuração)
  --target-user USER     Usuário do banco de destino (sobrescreve configuração)
  --target-pass PASS     Senha do banco de destino (sobrescreve configuração)
  --backup-file FILE     Caminho do arquivo de backup

Exemplos:
  # Criar arquivo de configuração inicial
  opsmaster backup postgresql init-config

  # Sincronizar banco usando configuração do arquivo
  opsmaster backup postgresql sync

  # Sincronizar banco com configuração personalizada
  opsmaster backup postgresql sync --source-host servidor1 --target-host servidor2

  # Dump de um banco usando configuração do arquivo
  opsmaster backup postgresql dump

  # Dump de um banco com configuração personalizada
  opsmaster backup postgresql dump --source-db mydb --source-user postgres --source-pass mypass

  # Restore de um backup
  opsmaster backup postgresql restore --target-db mydb --target-user postgres --target-pass mypass --backup-file /path/to/backup.dump

  # Listar backups disponíveis
  opsmaster backup postgresql list
EOF
}

# Função para verificar dependências necessárias
check_postgresql_deps() {
    check_dependency "pg_dump" "pg_restore" "psql"
}

# Função para instalar dependências
install_dependencies() {
    log_info "Instalando dependências do PostgreSQL..."
    
    if command -v apt-get &> /dev/null; then
        sudo apt-get update
        sudo apt-get install -y postgresql-client yq
    elif command -v yum &> /dev/null; then
        sudo yum install -y postgresql yq
    else
        log_error "Gerenciador de pacotes não suportado"
        return 1
    fi
    
    log_info "Dependências instaladas com sucesso"
}

# Função para processar argumentos
process_arguments() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_help
                exit 0
                ;;
            -c|--config)
                CONFIG_FILE="$2"
                shift 2
                ;;
            --source-host)
                SOURCE_HOST="$2"
                shift 2
                ;;
            --source-port)
                SOURCE_PORT="$2"
                shift 2
                ;;
            --source-db)
                SOURCE_DB="$2"
                shift 2
                ;;
            --source-user)
                SOURCE_USER="$2"
                shift 2
                ;;
            --source-pass)
                SOURCE_PASS="$2"
                shift 2
                ;;
            --target-host)
                TARGET_HOST="$2"
                shift 2
                ;;
            --target-port)
                TARGET_PORT="$2"
                shift 2
                ;;
            --target-db)
                TARGET_DB="$2"
                shift 2
                ;;
            --target-user)
                TARGET_USER="$2"
                shift 2
                ;;
            --target-pass)
                TARGET_PASS="$2"
                shift 2
                ;;
            --backup-file)
                BACKUP_FILE="$2"
                shift 2
                ;;
            dump)
                read_yaml_config "$CONFIG_FILE" && dump_postgresql
                exit $?
                ;;
            restore)
                read_yaml_config "$CONFIG_FILE" && restore_postgresql
                exit $?
                ;;
            sync)
                read_yaml_config "$CONFIG_FILE" && sync_postgresql
                exit $?
                ;;
            list)
                list_backups
                exit 0
                ;;
            install-deps)
                install_dependencies
                exit $?
                ;;
            init-config)
                create_config "$CONFIG_FILE"
                exit $?
                ;;
            *)
                log_error "Comando inválido: $1"
                show_help
                exit 1
                ;;
        esac
    done
}

# Função principal
main() {
    # Verificar dependências
    check_postgresql_deps

    # Processar argumentos
    process_arguments "$@"

    # Se nenhum comando foi especificado, mostrar ajuda
    show_help
    exit 1
}

# Executar função principal se o script for executado diretamente
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi 