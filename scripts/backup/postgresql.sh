#!/usr/bin/env bash
# Description: Gerencia backup e restore de bancos PostgreSQL

# shellcheck disable=SC1091
source "/usr/local/lib/opsmaster/common.sh"

# Constantes
readonly DEFAULT_BACKUP_DIR="/tmp/postgresql_backups"
readonly DEFAULT_CONFIG_FILE="$HOME/.config/backups/postgresql.yaml"
readonly DEFAULT_PORT="5432"
readonly DEFAULT_HOST="localhost"
# shellcheck disable=SC2155
readonly DATE_FORMAT=$(date +%Y%m%d_%H%M%S)

# Tipos de banco
readonly DB_TYPE_SOURCE="source"
readonly DB_TYPE_TARGET="target"

# Configurações
BACKUP_DIR="${BACKUP_DIR:-$DEFAULT_BACKUP_DIR}"
CONFIG_FILE="${CONFIG_FILE:-$DEFAULT_CONFIG_FILE}"

# Estrutura para armazenar configurações do banco
declare -A DB_CONFIG

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
validate_db_config() {
    local db_type="$1"
    local host="${DB_CONFIG[${db_type}_host]}"
    local port="${DB_CONFIG[${db_type}_port]}"
    local database="${DB_CONFIG[${db_type}_database]}"
    local username="${DB_CONFIG[${db_type}_username]}"
    local password="${DB_CONFIG[${db_type}_password]}"

    validate_required_field "Host" "$host" "$db_type" || return 1
    validate_port "$port" || return 1
    validate_required_field "Banco de dados" "$database" "$db_type" || return 1
    validate_required_field "Usuário" "$username" "$db_type" || return 1
    validate_required_field "Senha" "$password" "$db_type" || return 1

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
backup_dir: $DEFAULT_BACKUP_DIR
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

    # Função auxiliar para ler configuração do banco
    read_db_config() {
        local db_type="$1"
        DB_CONFIG["${db_type}_host"]=$(yq eval ".${db_type}.host // \"$DEFAULT_HOST\"" "$config_file")
        DB_CONFIG["${db_type}_port"]=$(yq eval ".${db_type}.port // \"$DEFAULT_PORT\"" "$config_file")
        DB_CONFIG["${db_type}_database"]=$(yq eval ".${db_type}.database // \"\"" "$config_file")
        DB_CONFIG["${db_type}_username"]=$(yq eval ".${db_type}.username // \"\"" "$config_file")
        DB_CONFIG["${db_type}_password"]=$(yq eval ".${db_type}.password // \"\"" "$config_file")
    }

    # Ler configurações dos bancos
    read_db_config "$DB_TYPE_SOURCE"
    read_db_config "$DB_TYPE_TARGET"

    # Configurar diretório de backup se especificado
    local backup_dir
    backup_dir=$(yq eval '.backup_dir // ""' "$config_file")
    if [ -n "$backup_dir" ]; then
        BACKUP_DIR="$backup_dir"
    fi

    # Debug das configurações
    log_info "Configurações carregadas:"
    log_info "Origem: ${DB_CONFIG[source_host]}:${DB_CONFIG[source_port]}/${DB_CONFIG[source_database]} (usuário: ${DB_CONFIG[source_username]})"
    log_info "Destino: ${DB_CONFIG[target_host]}:${DB_CONFIG[target_port]}/${DB_CONFIG[target_database]} (usuário: ${DB_CONFIG[target_username]})"

    # Validar configurações
    if ! validate_db_config "$DB_TYPE_SOURCE"; then
        return 1
    fi

    if ! validate_db_config "$DB_TYPE_TARGET"; then
        return 1
    fi

    log_info "Configurações carregadas de: $config_file"
}

# Função para executar comando PostgreSQL
execute_postgresql_command() {
    local db_type="$1"
    local command="$2"

    export PGPASSWORD="${DB_CONFIG[${db_type}_password]}"
    if psql -h "${DB_CONFIG[${db_type}_host]}" \
            -p "${DB_CONFIG[${db_type}_port]}" \
            -U "${DB_CONFIG[${db_type}_username]}" \
            -d "${DB_CONFIG[${db_type}_database]}" \
            -c "$command" >/dev/null 2>&1; then
        return 0
    else
        return 1
    fi
}

# Função para realizar dump
dump_postgresql() {
    local backup_file="${BACKUP_FILE:-$BACKUP_DIR/${DB_CONFIG[source_database]}_${DATE_FORMAT}.dump}"

    # Criar diretório de backup se não existir
    mkdir -p "$(dirname "$backup_file")"

    log_info "Iniciando dump do banco ${DB_CONFIG[source_database]} de ${DB_CONFIG[source_host]}:${DB_CONFIG[source_port]}"

    # Configurar variável de ambiente para senha
    export PGPASSWORD="${DB_CONFIG[source_password]}"

    # Realizar dump sem permissões de usuários
    if pg_dump -h "${DB_CONFIG[source_host]}" \
            -p "${DB_CONFIG[source_port]}" \
            -U "${DB_CONFIG[source_username]}" \
            -d "${DB_CONFIG[source_database]}" \
            -F c --no-owner --no-acl -f "$backup_file"; then
        log_info "Dump concluído com sucesso: $backup_file"
        show_backup_info "$backup_file"
        return 0
    else
        log_error "Falha ao realizar dump"
        return 1
    fi
}

# Função para restaurar backup
restore_postgresql() {
    local backup_file="$BACKUP_FILE"

    # Verificar se arquivo de backup existe
    if [ -z "$backup_file" ] || [ ! -f "$backup_file" ]; then
        log_error "Arquivo de backup não encontrado: $backup_file"
        return 1
    fi

    log_info "Iniciando restore do banco ${DB_CONFIG[target_database]} em ${DB_CONFIG[target_host]}:${DB_CONFIG[target_port]}"
    log_info "Usando arquivo de backup: $backup_file"

    # Configurar variável de ambiente para senha
    export PGPASSWORD="${DB_CONFIG[target_password]}"

    # Encerrar conexões existentes
    execute_postgresql_command "$DB_TYPE_TARGET" \
        "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '${DB_CONFIG[target_database]}';"

    # Dropar e recriar banco
    execute_postgresql_command "$DB_TYPE_TARGET" \
        "DROP DATABASE IF EXISTS ${DB_CONFIG[target_database]};"
    execute_postgresql_command "$DB_TYPE_TARGET" \
        "CREATE DATABASE ${DB_CONFIG[target_database]};"

    # Realizar restore com opções específicas
    if pg_restore -h "${DB_CONFIG[target_host]}" \
            -p "${DB_CONFIG[target_port]}" \
            -U "${DB_CONFIG[target_username]}" \
            -d "${DB_CONFIG[target_database]}" \
            --no-owner --no-acl --clean --if-exists "$backup_file"; then
        log_info "Restore concluído com sucesso"
        return 0
    else
        log_error "Falha ao realizar restore"
        return 1
    fi
}

# Função para listar backups
list_backups() {
    local backup_dir="${BACKUP_DIR:-$DEFAULT_BACKUP_DIR}"
    
    if [ ! -d "$backup_dir" ]; then
        log_info "Nenhum backup encontrado"
        return 0
    fi

    echo
    echo "Backups disponíveis em: $backup_dir"
    echo "----------------------------------------"
    echo "Arquivo                           Tamanho    Data"
    echo "----------------------------------------"

    # Ordenar por data de criação (mais recente primeiro)
    find "$backup_dir" -name "*.dump" -type f -printf "%T@\t%f\t%s\t%Td/%Tm/%TY %TH:%TM\n" | sort -nr | cut -f2- | while IFS=$'\t' read -r file size date; do
        size=$(numfmt --to=iec-i --suffix=B --format="%.1f" "$size")
        printf "%-30s %10s %s\n" "$file" "$size" "$date"
    done

    echo "----------------------------------------"
    local total_size
    total_size=$(du -sh "$backup_dir" 2>/dev/null | cut -f1)
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
    log_info "Iniciando sincronização do banco ${DB_CONFIG[source_database]}"
    log_info "Origem: ${DB_CONFIG[source_host]}:${DB_CONFIG[source_port]}"
    log_info "Destino: ${DB_CONFIG[target_host]}:${DB_CONFIG[target_port]}"

    # Definir nome do arquivo de backup
    local backup_file="${BACKUP_DIR}/${DB_CONFIG[source_database]}_${DATE_FORMAT}.dump"
    BACKUP_FILE="$backup_file"

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

    log_info "Sincronização concluída com sucesso"
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
  --backup-dir DIR       Diretório para listar backups (padrão: $DEFAULT_BACKUP_DIR)

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

  # Listar backups de um diretório específico
  opsmaster backup postgresql list --backup-dir /path/to/backups
EOF
}

# Função para verificar dependências necessárias
check_postgresql_deps() {
    check_dependency "pg_dump" "pg_restore" "psql" "yq"
}

# Função para instalar dependências
install_dependencies() {
    log_info "Instalando dependências do PostgreSQL..."
    
    # Instalar PostgreSQL client via gerenciador de pacotes do sistema
    if command -v apt-get &> /dev/null; then
        # Adicionar repositório oficial do PostgreSQL
        if ! grep -q "pgdg" /etc/apt/sources.list.d/pgdg.list 2>/dev/null; then
            log_info "Adicionando repositório oficial do PostgreSQL..."
            # Instalar dependências necessárias
            sudo apt-get update
            sudo apt-get install -y lsb-release wget
            
            # Adicionar repositório usando os comandos que funcionaram manualmente
            sudo sh -c 'echo "deb http://apt.postgresql.org/pub/repos/apt $(lsb_release -cs)-pgdg main" > /etc/apt/sources.list.d/pgdg.list'
            wget --quiet -O - https://www.postgresql.org/media/keys/ACCC4CF8.asc | sudo tee /etc/apt/trusted.gpg.d/postgresql.asc
            
            # Atualizar lista de pacotes
            sudo apt-get update
        fi
        
        # Remover versão antiga se existir
        sudo apt-get remove -y postgresql-client postgresql-client-14
        
        # Instalar PostgreSQL client 15
        sudo apt-get install -y postgresql-client-15
    elif command -v yum &> /dev/null; then
        # Adicionar repositório oficial do PostgreSQL para RHEL/CentOS
        if ! grep -q "pgdg" /etc/yum.repos.d/pgdg.repo 2>/dev/null; then
            log_info "Adicionando repositório oficial do PostgreSQL..."
            sudo yum install -y https://download.postgresql.org/pub/repos/yum/reporpms/EL-7-x86_64/pgdg-redhat-repo-latest.noarch.rpm
            sudo yum update
        fi
        sudo yum install -y postgresql15
    else
        log_error "Gerenciador de pacotes não suportado"
        return 1
    fi

    # Instalar yq via asdf se disponível
    if command -v asdf &> /dev/null; then
        log_info "Instalando yq via asdf..."
        if ! asdf plugin list | grep -q "yq"; then
            asdf plugin add yq
        fi
        asdf install yq latest
        asdf global yq latest
    else
        log_info "asdf não encontrado, instalando yq via wget..."
        sudo wget https://github.com/mikefarah/yq/releases/latest/download/yq_linux_amd64 -O /usr/local/bin/yq
        sudo chmod +x /usr/local/bin/yq
    fi
    
    log_info "Dependências instaladas com sucesso"
}

# Função para processar argumentos
process_arguments() {
    local command=""

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
                DB_CONFIG[source_host]="$2"
                shift 2
                ;;
            --source-port)
                DB_CONFIG[source_port]="$2"
                shift 2
                ;;
            --source-db)
                DB_CONFIG[source_database]="$2"
                shift 2
                ;;
            --source-user)
                DB_CONFIG[source_username]="$2"
                shift 2
                ;;
            --source-pass)
                DB_CONFIG[source_password]="$2"
                shift 2
                ;;
            --target-host)
                DB_CONFIG[target_host]="$2"
                shift 2
                ;;
            --target-port)
                DB_CONFIG[target_port]="$2"
                shift 2
                ;;
            --target-db)
                DB_CONFIG[target_database]="$2"
                shift 2
                ;;
            --target-user)
                DB_CONFIG[target_username]="$2"
                shift 2
                ;;
            --target-pass)
                DB_CONFIG[target_password]="$2"
                shift 2
                ;;
            --backup-file)
                BACKUP_FILE="$2"
                shift 2
                ;;
            --backup-dir)
                BACKUP_DIR="$2"
                shift 2
                ;;
            dump|restore|sync|list|install-deps|init-config)
                command="$1"
                shift
                ;;
            *)
                log_error "Comando inválido: $1"
                show_help
                exit 1
                ;;
        esac
    done

    # Executar comando se especificado
    case "$command" in
        dump)
            read_yaml_config "$CONFIG_FILE" && dump_postgresql
            ;;
        restore)
            read_yaml_config "$CONFIG_FILE" && restore_postgresql
            ;;
        sync)
            read_yaml_config "$CONFIG_FILE" && sync_postgresql
            ;;
        list)
            list_backups
            ;;
        install-deps)
            install_dependencies
            ;;
        init-config)
            create_config "$CONFIG_FILE"
            ;;
        *)
            show_help
            exit 1
            ;;
    esac
}

# Função principal
main() {
    # Verificar dependências
    check_postgresql_deps

    # Processar argumentos
    process_arguments "$@"
}

# Executar função principal se o script for executado diretamente
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi 