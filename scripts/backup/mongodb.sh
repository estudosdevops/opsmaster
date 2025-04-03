#!/usr/bin/env bash
# Description: Gerencia backup e restore de bancos MongoDB

# shellcheck disable=SC1091
source "/usr/local/lib/opsmaster/common.sh"

# Configurações padrão
BACKUP_DIR="${BACKUP_DIR:-/tmp/mongodb_backups}"
DATE_FORMAT=$(date +%Y%m%d_%H%M%S)

# Função para verificar dependências necessárias
check_mongodb_deps() {
    check_dependency "mongodump" "mongorestore" "mongosh"
}

# Função para obter lista de bancos excluindo os do sistema
get_user_databases() {
    local uri="$1"
    mongosh "$uri" --quiet --eval '
        db.getMongo().getDBs().databases
        .filter(db => !["admin", "local", "config"].includes(db.name))
        .filter(db => db.sizeOnDisk > 0)
        .map(db => db.name)
    ' | tr -d '[],"'
}

# Função para verificar permissões do usuário
check_mongodb_permissions() {
    local uri="$1"
    log_info "Verificando permissões do usuário..."
    
    if ! mongosh "$uri" --quiet --eval 'db.adminCommand({listDatabases: 1})' &>/dev/null; then
        log_error "Usuário sem permissões suficientes. É necessário:"
        log_error "- Ser usuário root ou"
        log_error "- Ter role 'backup' ou 'readAnyDatabase' para backup"
        log_error "- Ter role 'restore' ou 'readWriteAnyDatabase' para restore"
        exit 1
    fi
}

# Função para processar a URI do MongoDB
process_mongodb_uri() {
    local uri="$1"
    local base_uri

    # Se a URI termina com /, retorna ela mesma
    if [[ "$uri" =~ /$ ]]; then
        echo "$uri"
        return
    fi

    # Se a URI contém parâmetros de query (?), adiciona / antes do ?
    if [[ "$uri" =~ \? ]]; then
        base_uri=$(echo "$uri" | sed -E 's/\/[^/?]+(\?.+)$/\/\1/')
    else
        # Se não tem query, apenas adiciona / no final
        base_uri="${uri%/*}/"
    fi

    echo "$base_uri"
}

# Função para extrair credenciais da URI MongoDB
parse_mongodb_uri() {
    local uri="$1"
    local -n ref_host=$2
    local -n ref_port=$3
    local -n ref_username=$4
    local -n ref_password=$5
    local -n ref_auth_db=$6
    
    # Padrão para parsing da URI mongodb://[username:password@]host[:port][/database][?options]
    if [[ "$uri" =~ mongodb://([^:@]+):([^@]+)@([^:/]+):?([0-9]*)/?(.*) ]]; then
        ref_username="${BASH_REMATCH[1]}"
        ref_password="${BASH_REMATCH[2]}"
        ref_host="${BASH_REMATCH[3]}"
        ref_port="${BASH_REMATCH[4]:-27017}"
        ref_auth_db="admin"
    else
        ref_host="localhost"
        ref_port="27017"
        ref_username=""
        ref_password=""
        ref_auth_db=""
    fi
}

# Função para mostrar tamanho dos bancos backupeados
show_backup_sizes() {
    local backup_dir="$1"
    
    echo
    echo "Detalhes do backup:"
    echo "----------------------------------------"
    echo "Banco                           Tamanho"
    echo "----------------------------------------"
    
    # Usar find para localizar todos os diretórios de primeiro nível
    while IFS= read -r db_path; do
        if [ -d "$db_path" ]; then
            local db_name
            db_name=$(basename "$db_path")
            
            # Pular diretórios que não são bancos
            if [[ "$db_name" =~ ^(admin|local|config)$ ]]; then
                continue
            fi
            
            local size
            size=$(du -sh "$db_path" 2>/dev/null | cut -f1)
            printf "%-30s %10s\n" "$db_name" "${size:-0B}"
        fi
    done < <(find "$backup_dir" -mindepth 1 -maxdepth 1 -type d)
    
    echo "----------------------------------------"
    local total_size
    total_size=$(du -sh "$backup_dir" 2>/dev/null | cut -f1)
    echo "Tamanho total do backup: ${total_size:-0B}"
    echo
}

# Nova função para executar comandos MongoDB
execute_mongo_command() {
    local host="$1"
    local port="$2"
    local username="$3"
    local password="$4"
    local command="$5"
    
    if [ -n "$username" ] && [ -n "$password" ]; then
        mongosh --host "$host" --port "$port" \
                --username "$username" --password "$password" \
                --quiet --eval "$command"
    else
        mongosh --host "$host" --port "$port" \
                --quiet --eval "$command"
    fi
}

# Nova função para construir comando base MongoDB
build_mongo_command() {
    local cmd="$1"
    local host="$2"
    local port="$3"
    local username="$4"
    local password="$5"
    local auth_db="$6"
    
    local base_cmd="$cmd --host $host --port $port"
    
    if [ -n "$username" ] && [ -n "$password" ]; then
        base_cmd="$base_cmd --username $username --password $password --authenticationDatabase $auth_db"
    fi
    
    echo "$base_cmd"
}

# Nova função para verificar existência do banco
check_database_exists() {
    local host="$1"
    local port="$2"
    local username="$3"
    local password="$4"
    local db_name="$5"
    
    local check_cmd="db.getMongo().getDBs().databases.some(db => db.name === '$db_name')"
    
    if ! execute_mongo_command "$host" "$port" "$username" "$password" "$check_cmd"; then
        log_error "Banco de dados '$db_name' não encontrado"
        exit 1
    fi
}

# Função para realizar dump
do_dump() {
    local uri="$1"
    local specific_db="$2"
    local custom_dir="$3"  # Novo parâmetro para diretório personalizado
    
    # Extrair credenciais da URI
    local host port username password auth_db
    parse_mongodb_uri "$uri" host port username password auth_db
    
    log_info "Conectando ao MongoDB em $host:$port"
    
    # Usar diretório personalizado se fornecido, senão usar BACKUP_DIR padrão
    local base_dir="${custom_dir:-$BACKUP_DIR}"
    local output_dir="$base_dir/${specific_db:-all}_${DATE_FORMAT}"
    
    # Criar diretório de saída
    if ! mkdir -p "$output_dir"; then
        log_error "Não foi possível criar o diretório: $output_dir"
        exit 1
    fi
    
    log_info "Diretório de destino: $output_dir"
    
    # Construir comando base usando a nova função
    local cmd_base
    cmd_base=$(build_mongo_command "mongodump" "$host" "$port" "$username" "$password" "$auth_db")
    
    # Obter lista de bancos
    log_info "Obtendo lista de bancos de dados..."
    local databases
    
    if [ -n "$specific_db" ]; then
        check_database_exists "$host" "$port" "$username" "$password" "$specific_db"
        databases="$specific_db"
        log_info "Realizando dump do banco específico: $specific_db"
    else
        local mongo_cmd="db.adminCommand('listDatabases').databases"
        mongo_cmd="$mongo_cmd.filter(db => !['admin', 'local', 'config'].includes(db.name))"
        mongo_cmd="$mongo_cmd.map(db => db.name)"
        
        databases=$(execute_mongo_command "$host" "$port" "$username" "$password" "$mongo_cmd" | tr -d '[],"')
        log_info "Realizando dump de todos os bancos"
    fi
    
    if [ -z "$databases" ]; then
        log_error "Nenhum banco de dados encontrado para dump"
        exit 1
    fi
    
    # Realizar dump de cada banco
    for db in $databases; do
        log_info "Realizando dump do banco: $db"
        
        # Construir comando completo para este banco
        local cmd="$cmd_base --db=$db --out=$output_dir"
        
        log_info "Executando: $cmd"
        if eval "$cmd"; then
            log_info "✅ Dump do banco $db concluído com sucesso"
        else
            log_error "❌ Falha no dump do banco $db"
            exit 1
        fi
    done
    
    # Mostrar informações do dump completo
    log_info "Dump completo!"
    log_info "Diretório: $output_dir"
    
    # Mostrar tamanho de cada banco backupeado
    show_backup_sizes "$output_dir"
}

# Função para realizar restore
do_restore() {
    local uri="$1"
    local backup_path="$2"
    local specific_db="$3"
    
    # Verificar dependências antes de prosseguir
    check_mongodb_deps
    
    # Extrair credenciais da URI
    local host port username password auth_db
    parse_mongodb_uri "$uri" host port username password auth_db
    
    log_info "Conectando ao MongoDB em $host:$port"
    
    # Ajustar caminho do backup se necessário
    if [[ ! "$backup_path" = /* ]]; then
        backup_path="$BACKUP_DIR/$backup_path"
    fi
    
    if [ ! -d "$backup_path" ]; then
        log_error "Diretório de backup não encontrado: $backup_path"
        log_error "Backups disponíveis:"
        list_backups
        exit 1
    fi
    
    # Construir comando base usando a nova função
    local cmd_base
    cmd_base=$(build_mongo_command "mongorestore" "$host" "$port" "$username" "$password" "$auth_db")
    
    log_info "Iniciando restore MongoDB..."
    log_info "Diretório de origem: $backup_path"
    
    # Se um banco específico foi solicitado
    if [ -n "$specific_db" ]; then
        local db_path="$backup_path/$specific_db"
        if [ ! -d "$db_path" ]; then
            log_error "Banco $specific_db não encontrado no backup: $backup_path"
            log_info "Bancos disponíveis neste backup:"
            ls -1 "$backup_path"
            exit 1
        fi
        
        log_info "Restaurando banco específico: $specific_db"
        if eval "$cmd_base --db=$specific_db $db_path"; then
            log_info "✅ Restore do banco $specific_db concluído com sucesso"
        else
            log_error "❌ Falha no restore do banco $specific_db"
            exit 1
        fi
    else
        # Restore de todos os bancos
        log_info "Restaurando todos os bancos do backup..."
        
        # Encontrar todos os diretórios de bancos de dados no backup
        for db_path in "$backup_path"/*; do
            if [ -d "$db_path" ]; then
                db_name=$(basename "$db_path")
                
                # Pular bancos do sistema
                if [[ "$db_name" =~ ^(admin|local|config)$ ]]; then
                    log_warn "Pulando banco do sistema: $db_name"
                    continue
                fi
                
                log_info "Restaurando banco: $db_name"
                if eval "$cmd_base --db=$db_name $db_path"; then
                    log_info "✅ Restore do banco $db_name concluído com sucesso"
                else
                    log_error "❌ Falha no restore do banco $db_name"
                    exit 1
                fi
            fi
        done
    fi
    
    log_info "Restore completo!"
}

# Função para listar backups disponíveis
list_backups() {
    log_info "Backups disponíveis em $BACKUP_DIR:"
    if [ -d "$BACKUP_DIR" ]; then
        ls -lh "$BACKUP_DIR" | awk 'NR>1 {printf "%-50s %10s\n", $9, $5}'
    else
        log_warn "Diretório de backup não encontrado"
    fi
}

# Função para instalar dependências
install_deps() {
    log_info "Verificando e instalando dependências necessárias..."
    
    # Detectar o sistema operacional
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        OS=$ID
    else
        OS=$(uname -s)
    fi
    
    case "$OS" in
        ubuntu|debian)
            log_info "Detectado sistema Debian/Ubuntu"
            log_info "Adicionando repositório MongoDB..."
            
            # Instalar pré-requisitos
            sudo apt-get update
            sudo apt-get install -y wget gnupg

            # Adicionar chave GPG do MongoDB
            wget -qO - https://www.mongodb.org/static/pgp/server-6.0.asc | sudo apt-key add -
            
            # Adicionar repositório
            echo "deb [ arch=amd64,arm64 ] https://repo.mongodb.org/apt/ubuntu $(lsb_release -cs)/mongodb-org/6.0 multiverse" | \
                sudo tee /etc/apt/sources.list.d/mongodb-org-6.0.list
            
            # Atualizar e instalar
            log_info "Instalando ferramentas MongoDB..."
            sudo apt-get update
            sudo apt-get install -y mongodb-mongosh mongodb-database-tools
            ;;
            
        rhel|centos|amzn)
            log_info "Detectado sistema RHEL/CentOS/Amazon Linux"
            log_info "Adicionando repositório MongoDB..."
            
            # Criar arquivo do repositório
            cat << EOF | sudo tee /etc/yum.repos.d/mongodb-org-6.0.repo
[mongodb-org-6.0]
name=MongoDB Repository
baseurl=https://repo.mongodb.org/yum/redhat/\$releasever/mongodb-org/6.0/x86_64/
gpgcheck=1
enabled=1
gpgkey=https://www.mongodb.org/static/pgp/server-6.0.asc
EOF
            
            # Instalar
            log_info "Instalando ferramentas MongoDB..."
            sudo yum install -y mongodb-mongosh mongodb-database-tools
            ;;
            
        *)
            log_error "Sistema operacional não suportado para instalação automática: $OS"
            log_info "Por favor, instale manualmente:"
            log_info "- mongosh (MongoDB Shell)"
            log_info "- mongodb-database-tools (para mongodump e mongorestore)"
            exit 1
            ;;
    esac
    
    # Verificar se a instalação foi bem sucedida
    if command -v mongosh >/dev/null 2>&1 && \
       command -v mongodump >/dev/null 2>&1 && \
       command -v mongorestore >/dev/null 2>&1; then
        log_info "✅ Todas as dependências foram instaladas com sucesso!"
    else
        log_error "❌ Falha na instalação das dependências"
        log_error "Por favor, verifique os erros acima e tente novamente"
        exit 1
    fi
}

# Função de ajuda
show_help() {
    echo "Gerenciamento de Backup/Restore MongoDB"
    echo
    echo "Uso: opsmaster backup mongodb <ação> [argumentos]"
    echo
    echo "Ações:"
    echo "  dump <uri> [banco] [dir]      Realiza dump (de um banco específico ou todos)"
    echo "  restore <uri> <path> [banco]   Restaura backup (de um banco específico ou todos)"
    echo "  list                           Lista backups disponíveis com seus tamanhos"
    echo "  install-deps                   Instala dependências necessárias"
    echo
    echo "Argumentos:"
    echo "  uri                            URI de conexão do MongoDB"
    echo "  banco                          Nome do banco de dados (opcional)"
    echo "  dir                            Diretório de destino para o dump (opcional)"
    echo "  path                           Caminho do backup para restore"
    echo
    echo "Variáveis de ambiente:"
    echo "  BACKUP_DIR                     Diretório para armazenar backups (padrão: /tmp/mongodb_backups)"
    echo
    echo "Exemplos:"
    echo "  1. Dump de todos os bancos no diretório padrão:"
    echo "     opsmaster backup mongodb dump 'mongodb://root:senha@localhost:27017'"
    echo
    echo "  2. Dump de um banco específico em diretório personalizado:"
    echo "     opsmaster backup mongodb dump 'mongodb://root:senha@localhost:27017' meudb /path/to/backup"
    echo
    echo "  3. Restore de todos os bancos:"
    echo "     opsmaster backup mongodb restore 'mongodb://root:senha@localhost:27017' all_20240225_150226"
    echo
    echo "  4. Restore de um banco específico:"
    echo "     opsmaster backup mongodb restore 'mongodb://root:senha@localhost:27017' all_20240225_150226 meudb"
    echo
    echo "  5. Listar backups disponíveis:"
    echo "     opsmaster backup mongodb list"
    echo
    echo "  6. Instalar dependências:"
    echo "     opsmaster backup mongodb install-deps"
    echo
    echo "Formatos de URI suportados:"
    echo "  - mongodb://localhost:27017                     (sem autenticação)"
    echo "  - mongodb://usuario:senha@localhost:27017       (com autenticação básica)"
    echo "  - mongodb://usuario:senha@localhost:27017/admin (com banco específico)"
    echo
    echo "Notas:"
    echo "  - Os backups são salvos em diretórios com o formato: <banco>_<data>_<hora>"
    echo "  - Para backups completos, usa-se o prefixo 'all' no nome do diretório"
    echo "  - O comando list mostra o nome e tamanho de cada backup disponível"
    echo "  - As dependências podem ser instaladas automaticamente em sistemas Debian, Ubuntu e RHEL"
    echo
}

# Função principal
main() {
    local action="$1"
    local uri="$2"
    local param3="$3"    # Banco ou caminho do backup
    local param4="$4"    # Banco para restore ou diretório para dump
    
    case "$action" in
        dump)
            if [ -z "$uri" ]; then
                log_error "URI do MongoDB não especificada"
                show_help
                exit 1
            fi
            do_dump "$uri" "$param3" "$param4"  # param3: banco (opcional), param4: diretório (opcional)
            ;;
        restore)
            if [ -z "$uri" ] || [ -z "$param3" ]; then
                log_error "URI do MongoDB ou caminho do backup não especificado"
                show_help
                exit 1
            fi
            do_restore "$uri" "$param3" "$param4"  # param4 é opcional (nome do banco)
            ;;
        list)
            list_backups
            ;;
        install-deps)
            install_deps
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            log_error "Ação desconhecida: $action"
            show_help
            exit 1
            ;;
    esac
}

main "$@" 