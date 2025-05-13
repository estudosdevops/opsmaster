#!/usr/bin/env bash
# Description: Gerencia backup e restore de bancos MongoDB

# shellcheck disable=SC1091
source "/usr/local/lib/opsmaster/common.sh"

# Configurações padrão
BACKUP_DIR="${BACKUP_DIR:-/tmp/mongodb_backups}"
DATE_FORMAT=$(date +%Y%m%d_%H%M%S)
MONGODB_SOURCE_URI="${MONGODB_SOURCE_URI:-}"
MONGODB_DEST_URI="${MONGODB_DEST_URI:-}"

# Função para verificar dependências necessárias
check_mongodb_deps() {
    check_dependency "mongodump" "mongorestore" "mongosh" "mongoexport" "mongoimport"
    
    # Verificar se o dnspython está instalado (necessário para mongodb+srv)
    if ! python3 -c "import dns" &>/dev/null; then
        log_error "dnspython não está instalado. Este pacote é necessário para conexões mongodb+srv"
        log_info "Execute 'opsmaster backup mongodb install-deps' para instalar"
        exit 1
    fi
}

# Função para verificar e estabelecer conexão
verify_mongodb_connection() {
    local uri="$1"
    local operation="$2"
    
    # Verificar dependências antes de prosseguir
    check_mongodb_deps
    
    # Testar conexão antes de prosseguir
    if ! test_mongodb_connection "$uri"; then
        exit 1
    fi
    
    # Extrair credenciais da URI
    local host port username password auth_db
    parse_mongodb_uri "$uri" host port username password auth_db
    
    log_info "Conectando ao MongoDB em $host:$port para $operation"
    
    echo "$host:$port:$username:$password:$auth_db"
}

# Função para criar e verificar diretório
setup_backup_directory() {
    local base_dir="$1"
    local prefix="$2"
    local specific_db="$3"
    
    local output_dir="$base_dir/${specific_db:-$prefix}_${DATE_FORMAT}"
    
    # Criar diretório de saída
    if ! mkdir -p "$output_dir"; then
        log_error "Não foi possível criar o diretório: $output_dir"
        exit 1
    fi
    
    log_info "Diretório de destino: $output_dir"
    echo "$output_dir"
}

# Função para obter lista de bancos
get_databases_list() {
    local uri="$1"
    local specific_db="$2"
    local host port username password auth_db
    
    IFS=':' read -r host port username password auth_db <<< "$(verify_mongodb_connection "$uri" "listar bancos")"
    
    if [ -n "$specific_db" ]; then
        check_database_exists "$host" "$port" "$username" "$password" "$specific_db"
        echo "$specific_db"
    else
        get_user_databases "$uri"
    fi
}

# Função para construir comando base MongoDB
build_mongo_command() {
    local cmd="$1"
    local host="$2"
    local port="$3"
    local username="$4"
    local password="$5"
    local auth_db="$6"
    
    # Se a URI contém mongodb+srv, construímos a URI completa
    if [[ "$host" == *".mongodb.net" ]]; then
        if [ -n "$username" ] && [ -n "$password" ]; then
            echo "$cmd --uri mongodb+srv://$username:$password@$host"
        else
            echo "$cmd --uri mongodb+srv://$host"
        fi
    else
        local base_cmd="$cmd --host $host --port $port"
        
        if [ -n "$username" ] && [ -n "$password" ]; then
            base_cmd="$base_cmd --username $username --password $password --authenticationDatabase $auth_db"
        fi
        
        echo "$base_cmd"
    fi
}

# Função para realizar dump
do_dump() {
    local uri="$1"
    local specific_db="$2"
    local custom_dir="$3"
    
    local host port username password auth_db
    IFS=':' read -r host port username password auth_db <<< "$(verify_mongodb_connection "$uri" "dump")"
    
    # Usar diretório personalizado se fornecido, senão usar BACKUP_DIR padrão
    local base_dir="${custom_dir:-$BACKUP_DIR}"
    local output_dir
    output_dir=$(setup_backup_directory "$base_dir" "all" "$specific_db")
    
    # Construir comando base usando função
    local cmd_base
    cmd_base=$(build_mongo_command "mongodump" "$host" "$port" "$username" "$password" "$auth_db")
    
    # Obter lista de bancos
    log_info "Obtendo lista de bancos de dados..."
    local databases
    databases=$(get_databases_list "$uri" "$specific_db")
    
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
            log_info "Dump do banco $db concluído com sucesso"
        else
            log_error "Falha no dump do banco $db"
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
    local db_mapping="$4"  # Novo parâmetro para mapeamento de bancos
    
    local host port username password auth_db
    IFS=':' read -r host port username password auth_db <<< "$(verify_mongodb_connection "$uri" "restore")"
    
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
    
    # Construir comando base usando função
    local cmd_base
    cmd_base=$(build_mongo_command "mongorestore" "$host" "$port" "$username" "$password" "$auth_db")
    
    log_info "Iniciando restore MongoDB..."
    log_info "Diretório de origem: $backup_path"
    
    # Se um banco específico foi solicitado
    if [ -n "$specific_db" ]; then
        # Verificar se há mapeamento para este banco
        local target_db="$specific_db"
        if [ -n "$db_mapping" ]; then
            # Procurar o mapeamento no formato "origem:destino"
            while IFS=: read -r source_db dest_db; do
                if [ "$source_db" = "$specific_db" ]; then
                    target_db="$dest_db"
                    log_info "Mapeando banco de origem '$source_db' para destino '$dest_db'"
                    break
                fi
            done <<< "$db_mapping"
        fi
        
        local db_path="$backup_path/$specific_db"
        if [ ! -d "$db_path" ]; then
            log_error "Banco $specific_db não encontrado no backup: $backup_path"
            log_info "Bancos disponíveis neste backup:"
            ls -1 "$backup_path"
            exit 1
        fi
        
        log_info "Restaurando banco específico: $specific_db -> $target_db"
        if eval "$cmd_base --db=$target_db $db_path"; then
            log_info "Restore do banco $target_db concluído com sucesso"
        else
            log_error "Falha no restore do banco $target_db"
            exit 1
        fi
    else
        # Restore de todos os bancos
        log_info "Restaurando todos os bancos do backup..."
        
        # Encontrar todos os diretórios de bancos de dados no backup
        for db_path in "$backup_path"/*; do
            if [ -d "$db_path" ]; then
                local source_db
                source_db=$(basename "$db_path")
                
                # Pular bancos do sistema
                if [[ "$source_db" =~ ^(admin|local|config)$ ]]; then
                    log_warn "Pulando banco do sistema: $source_db"
                    continue
                fi
                
                # Verificar se há mapeamento para este banco
                local target_db="$source_db"
                if [ -n "$db_mapping" ]; then
                    # Procurar o mapeamento no formato "origem:destino"
                    while IFS=: read -r map_source map_dest; do
                        if [ "$map_source" = "$source_db" ]; then
                            target_db="$map_dest"
                            log_info "Mapeando banco de origem '$map_source' para destino '$map_dest'"
                            break
                        fi
                    done <<< "$db_mapping"
                fi
                
                log_info "Restaurando banco: $source_db -> $target_db"
                if eval "$cmd_base --db=$target_db $db_path"; then
                    log_info "Restore do banco $target_db concluído com sucesso"
                else
                    log_error "Falha no restore do banco $target_db"
                    exit 1
                fi
            fi
        done
    fi
    
    log_info "Restore completo!"
}

# Função para realizar export
do_export() {
    local uri="$1"
    local specific_db="$2"
    local custom_dir="$3"
    
    local host port username password auth_db
    IFS=':' read -r host port username password auth_db <<< "$(verify_mongodb_connection "$uri" "export")"
    
    # Usar diretório personalizado se fornecido, senão usar BACKUP_DIR padrão
    local base_dir="${custom_dir:-$BACKUP_DIR}"
    local output_dir
    output_dir=$(setup_backup_directory "$base_dir" "all_export" "$specific_db")
    
    # Construir comando base usando função
    local cmd_base
    cmd_base=$(build_mongo_command "mongoexport" "$host" "$port" "$username" "$password" "$auth_db")
    
    # Obter lista de bancos
    log_info "Obtendo lista de bancos de dados..."
    local databases
    databases=$(get_databases_list "$uri" "$specific_db")
    
    if [ -z "$databases" ]; then
        log_error "Nenhum banco de dados encontrado para export"
        exit 1
    fi
    
    # Realizar export de cada banco
    for db in $databases; do
        log_info "Realizando export do banco: $db"
        
        # Criar diretório para o banco
        local db_dir="$output_dir/$db"
        mkdir -p "$db_dir"
        
        # Obter lista de collections
        local collections
        collections=$(execute_mongo_command "$host" "$port" "$username" "$password" "db.getSiblingDB('$db').getCollectionNames()" | tr -d '[],"')
        
        # Exportar cada collection
        for collection in $collections; do
            log_info "Exportando collection: $collection"
            
            # Construir comando para esta collection
            local cmd="$cmd_base --db=$db --collection=$collection --out=$db_dir/${collection}.json"
            
            log_info "Executando: $cmd"
            if eval "$cmd"; then
                log_info "Export da collection $collection concluído com sucesso"
            else
                log_error "Falha no export da collection $collection"
                exit 1
            fi
        done
    done
    
    # Mostrar informações do export completo
    log_info "Export completo!"
    log_info "Diretório: $output_dir"
    
    # Mostrar tamanho de cada banco exportado
    show_backup_sizes "$output_dir"
}

# Função para realizar import
do_import() {
    local uri="$1"
    local backup_path="$2"
    local specific_db="$3"
    local db_mapping="$4"  # Novo parâmetro para mapeamento de bancos
    
    local host port username password auth_db
    IFS=':' read -r host port username password auth_db <<< "$(verify_mongodb_connection "$uri" "import")"
    
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
    
    # Construir comando base usando função
    local cmd_base
    cmd_base=$(build_mongo_command "mongoimport" "$host" "$port" "$username" "$password" "$auth_db")
    
    log_info "Iniciando import MongoDB..."
    log_info "Diretório de origem: $backup_path"
    
    # Se um banco específico foi solicitado
    if [ -n "$specific_db" ]; then
        # Verificar se há mapeamento para este banco
        local target_db="$specific_db"
        if [ -n "$db_mapping" ]; then
            # Procurar o mapeamento no formato "origem:destino"
            while IFS=: read -r source_db dest_db; do
                if [ "$source_db" = "$specific_db" ]; then
                    target_db="$dest_db"
                    log_info "Mapeando banco de origem '$source_db' para destino '$dest_db'"
                    break
                fi
            done <<< "$db_mapping"
        fi
        
        local db_path="$backup_path/$specific_db"
        if [ ! -d "$db_path" ]; then
            log_error "Banco $specific_db não encontrado no backup: $backup_path"
            log_info "Bancos disponíveis neste backup:"
            ls -1 "$backup_path"
            exit 1
        fi
        
        log_info "Importando banco específico: $specific_db -> $target_db"
        
        # Importar cada arquivo JSON no diretório do banco
        for json_file in "$db_path"/*.json; do
            if [ -f "$json_file" ]; then
                local collection
                collection=$(basename "$json_file" .json)
                log_info "Importando collection: $collection"
                
                if eval "$cmd_base --db=$target_db --collection=$collection --file=$json_file"; then
                    log_info "Import da collection $collection concluído com sucesso"
                else
                    log_error "Falha no import da collection $collection"
                    exit 1
                fi
            fi
        done
    else
        # Import de todos os bancos
        log_info "Importando todos os bancos do backup..."
        
        # Encontrar todos os diretórios de bancos de dados no backup
        for db_path in "$backup_path"/*; do
            if [ -d "$db_path" ]; then
                local source_db
                source_db=$(basename "$db_path")
                
                # Pular bancos do sistema
                if [[ "$source_db" =~ ^(admin|local|config)$ ]]; then
                    log_warn "Pulando banco do sistema: $source_db"
                    continue
                fi
                
                # Verificar se há mapeamento para este banco
                local target_db="$source_db"
                if [ -n "$db_mapping" ]; then
                    # Procurar o mapeamento no formato "origem:destino"
                    while IFS=: read -r map_source map_dest; do
                        if [ "$map_source" = "$source_db" ]; then
                            target_db="$map_dest"
                            log_info "Mapeando banco de origem '$map_source' para destino '$map_dest'"
                            break
                        fi
                    done <<< "$db_mapping"
                fi
                
                log_info "Importando banco: $source_db -> $target_db"
                
                # Importar cada arquivo JSON no diretório do banco
                for json_file in "$db_path"/*.json; do
                    if [ -f "$json_file" ]; then
                        local collection
                        collection=$(basename "$json_file" .json)
                        log_info "Importando collection: $collection"
                        
                        if eval "$cmd_base --db=$target_db --collection=$collection --file=$json_file"; then
                            log_info "Import da collection $collection concluído com sucesso"
                        else
                            log_error "Falha no import da collection $collection"
                            exit 1
                        fi
                    fi
                done
            fi
        done
    fi
    
    log_info "Import completo!"
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

# Função para executar comandos MongoDB
execute_mongo_command() {
    local host="$1"
    local port="$2"
    local username="$3"
    local password="$4"
    local command="$5"
    
    # Se a URI contém mongodb+srv, não usamos host e port separadamente
    if [[ "$host" == *".mongodb.net" ]]; then
        if [ -n "$username" ] && [ -n "$password" ]; then
            mongosh "mongodb+srv://$username:$password@$host" \
                    --quiet --eval "$command"
        else
            mongosh "mongodb+srv://$host" \
                    --quiet --eval "$command"
        fi
    else
        if [ -n "$username" ] && [ -n "$password" ]; then
            mongosh --host "$host" --port "$port" \
                    --username "$username" --password "$password" \
                    --quiet --eval "$command"
        else
            mongosh --host "$host" --port "$port" \
                    --quiet --eval "$command"
        fi
    fi
}

# Função para testar conexão com MongoDB
test_mongodb_connection() {
    local uri="$1"
    local host port username password auth_db
    parse_mongodb_uri "$uri" host port username password auth_db
    
    # Se a URI contém mongodb+srv, mostramos mensagem sem porta
    if [[ "$uri" == *"mongodb+srv://"* ]]; then
        log_info "Testando conexão com MongoDB Atlas em $host..."
    else
        log_info "Testando conexão com MongoDB em $host:$port..."
    fi
    
    # Comando para testar a conexão
    local test_cmd="db.runCommand({ping: 1})"
    
    # Se a URI contém mongodb+srv, usamos a URI completa
    if [[ "$uri" == *"mongodb+srv://"* ]]; then
        if ! mongosh "$uri" --quiet --eval "$test_cmd" >/dev/null 2>&1; then
            log_error "Não foi possível conectar ao MongoDB Atlas"
            log_error "Verifique:"
            log_error "- Se o servidor está acessível"
            log_error "- Se as credenciais estão corretas"
            log_error "- Se o usuário tem as permissões necessárias"
            log_error "- Se o IP do seu servidor está liberado no MongoDB Atlas"
            return 1
        fi
    else
        if ! execute_mongo_command "$host" "$port" "$username" "$password" "$test_cmd" >/dev/null 2>&1; then
            log_error "Não foi possível conectar ao MongoDB"
            log_error "Verifique:"
            log_error "- Se o servidor está acessível"
            log_error "- Se as credenciais estão corretas"
            log_error "- Se o usuário tem as permissões necessárias"
            return 1
        fi
    fi
    
    log_info "Conexão estabelecida com sucesso"
    return 0
}

# Função para verificar existência do banco
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
    # ou mongodb+srv://[username:password@]host[/database][?options]
    if [[ "$uri" =~ mongodb(\+srv)?://([^:@]+):([^@]+)@([^:/]+)(:([0-9]+))?/?(.*) ]]; then
        ref_username="${BASH_REMATCH[2]}"
        ref_password="${BASH_REMATCH[3]}"
        ref_host="${BASH_REMATCH[4]}"
        ref_port="${BASH_REMATCH[6]:-27017}"
        ref_auth_db="admin"
    else
        ref_host="localhost"
        ref_port="27017"
        ref_username=""
        ref_password=""
        ref_auth_db=""
    fi
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
            sudo apt-get install -y wget gnupg python3-pip

            # Adicionar chave GPG do MongoDB
            wget -qO - https://www.mongodb.org/static/pgp/server-6.0.asc | sudo apt-key add -
            
            # Adicionar repositório
            echo "deb [ arch=amd64,arm64 ] https://repo.mongodb.org/apt/ubuntu $(lsb_release -cs)/mongodb-org/6.0 multiverse" | \
                sudo tee /etc/apt/sources.list.d/mongodb-org-6.0.list
            
            # Atualizar e instalar
            log_info "Instalando ferramentas MongoDB..."
            sudo apt-get update
            sudo apt-get install -y mongodb-mongosh mongodb-database-tools
            sudo pip3 install dnspython
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
            sudo yum install -y mongodb-mongosh mongodb-database-tools python3-pip
            sudo pip3 install dnspython
            ;;
            
        *)
            log_error "Sistema operacional não suportado para instalação automática: $OS"
            log_info "Por favor, instale manualmente:"
            log_info "- mongosh (MongoDB Shell)"
            log_info "- mongodb-database-tools (para mongodump e mongorestore)"
            log_info "- dnspython (para suporte a mongodb+srv)"
            exit 1
            ;;
    esac
    
    # Verificar se a instalação foi bem sucedida
    if command -v mongosh >/dev/null 2>&1 && \
       command -v mongodump >/dev/null 2>&1 && \
       command -v mongorestore >/dev/null 2>&1 && \
       python3 -c "import dns" &>/dev/null; then
        log_info "Todas as dependências foram instaladas com sucesso!"
    else
        log_error "Falha na instalação das dependências"
        log_error "Por favor, verifique os erros acima e tente novamente"
        exit 1
    fi
}

# Função para realizar backup/restore em um único comando
do_backup_restore() {
    local mode="$1"  # dump ou export
    local source_uri="$2"
    local dest_uri="$3"
    local specific_db="$4"
    local custom_dir="$5"
    local db_mapping="$6"  # Novo parâmetro para mapeamento de bancos
    
    # Verificar URIs
    if [ -z "$source_uri" ]; then
        source_uri="$MONGODB_SOURCE_URI"
        if [ -z "$source_uri" ]; then
            log_error "URI de origem não especificada"
            log_error "Use --source ou defina MONGODB_SOURCE_URI"
            exit 1
        fi
    fi
    
    if [ -z "$dest_uri" ]; then
        dest_uri="$MONGODB_DEST_URI"
        if [ -z "$dest_uri" ]; then
            log_error "URI de destino não especificada"
            log_error "Use --dest ou defina MONGODB_DEST_URI"
            exit 1
        fi
    fi
    
    # Criar diretório temporário para o backup
    local temp_dir
    temp_dir=$(mktemp -d)
    
    # Realizar backup
    log_info "Realizando backup do banco de origem..."
    if [ "$mode" = "dump" ]; then
        do_dump "$source_uri" "$specific_db" "$temp_dir"
    else
        do_export "$source_uri" "$specific_db" "$temp_dir"
    fi
    
    # Encontrar o diretório de backup criado
    local backup_path
    backup_path=$(find "$temp_dir" -maxdepth 1 -type d | sort -r | head -n1)
    
    if [ -z "$backup_path" ]; then
        log_error "Falha ao localizar diretório de backup"
        rm -rf "$temp_dir"
        exit 1
    fi
    
    # Realizar restore
    log_info "Realizando restore no banco de destino..."
    if [ "$mode" = "dump" ]; then
        do_restore "$dest_uri" "$backup_path" "$specific_db" "$db_mapping"
    else
        do_import "$dest_uri" "$backup_path" "$specific_db" "$db_mapping"
    fi
    
    # Limpar diretório temporário
    rm -rf "$temp_dir"
    
    log_info "Operação concluída com sucesso!"
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
    echo "  export <uri> [banco] [dir]    Realiza export JSON (de um banco específico ou todos)"
    echo "  import <uri> <path> [banco]   Importa arquivos JSON (de um banco específico ou todos)"
    echo "  backup [opções]               Realiza backup e restore em um único comando"
    echo "  list                           Lista backups disponíveis com seus tamanhos"
    echo "  install-deps                   Instala dependências necessárias"
    echo
    echo "Argumentos:"
    echo "  uri                            URI de conexão do MongoDB"
    echo "  banco                          Nome do banco de dados (opcional)"
    echo "  dir                            Diretório de destino para o dump/export (opcional)"
    echo "  path                           Caminho do backup para restore/import"
    echo
    echo "Opções para o comando backup:"
    echo "  --source <uri>                 URI do banco de origem (opcional se MONGODB_SOURCE_URI estiver definido)"
    echo "  --dest <uri>                   URI do banco de destino (opcional se MONGODB_DEST_URI estiver definido)"
    echo "  --mode <dump|export>           Modo de backup (dump ou export, padrão: dump)"
    echo "  --db <banco>                   Banco específico para backup/restore"
    echo "  --db-mapping <map>             Mapeamento de nomes de bancos (formato: origem:destino,origem2:destino2)"
    echo
    echo "Variáveis de ambiente:"
    echo "  BACKUP_DIR                     Diretório para armazenar backups (padrão: /tmp/mongodb_backups)"
    echo "  MONGODB_SOURCE_URI             URI do banco de origem para o comando backup"
    echo "  MONGODB_DEST_URI               URI do banco de destino para o comando backup"
    echo
    echo "Exemplos para MongoDB Atlas:"
    echo "  1. Dump de todos os bancos:"
    echo "     opsmaster backup mongodb dump 'mongodb+srv://usuario:senha@seu-cluster.mongodb.net'"
    echo
    echo "  2. Dump de um banco específico:"
    echo "     opsmaster backup mongodb dump 'mongodb+srv://usuario:senha@seu-cluster.mongodb.net' meudb"
    echo
    echo "  3. Restore de todos os bancos:"
    echo "     opsmaster backup mongodb restore 'mongodb+srv://usuario:senha@seu-cluster.mongodb.net' all_20240225_150226"
    echo
    echo "  4. Restore de um banco específico:"
    echo "     opsmaster backup mongodb restore 'mongodb+srv://usuario:senha@seu-cluster.mongodb.net' all_20240225_150226 meudb"
    echo
    echo "  5. Export de todos os bancos:"
    echo "     opsmaster backup mongodb export 'mongodb+srv://usuario:senha@seu-cluster.mongodb.net'"
    echo
    echo "  6. Export de um banco específico:"
    echo "     opsmaster backup mongodb export 'mongodb+srv://usuario:senha@seu-cluster.mongodb.net' meudb"
    echo
    echo "  7. Import de todos os bancos:"
    echo "     opsmaster backup mongodb import 'mongodb+srv://usuario:senha@seu-cluster.mongodb.net' all_export_20240225_150226"
    echo
    echo "  8. Import de um banco específico:"
    echo "     opsmaster backup mongodb import 'mongodb+srv://usuario:senha@seu-cluster.mongodb.net' all_export_20240225_150226 meudb"
    echo
    echo "Exemplos para MongoDB Local/Remoto:"
    echo "  1. Dump de todos os bancos (localhost):"
    echo "     opsmaster backup mongodb dump 'mongodb://root:senha@localhost:27017'"
    echo
    echo "  2. Dump de um banco específico (servidor remoto):"
    echo "     opsmaster backup mongodb dump 'mongodb://usuario:senha@servidor:27017' meudb"
    echo
    echo "  3. Restore de todos os bancos:"
    echo "     opsmaster backup mongodb restore 'mongodb://root:senha@localhost:27017' all_20240225_150226"
    echo
    echo "  4. Restore de um banco específico:"
    echo "     opsmaster backup mongodb restore 'mongodb://usuario:senha@servidor:27017' all_20240225_150226 meudb"
    echo
    echo "  5. Export de todos os bancos (localhost):"
    echo "     opsmaster backup mongodb export 'mongodb://root:senha@localhost:27017'"
    echo
    echo "  6. Export de um banco específico (servidor remoto):"
    echo "     opsmaster backup mongodb export 'mongodb://usuario:senha@servidor:27017' meudb"
    echo
    echo "  7. Import de todos os bancos:"
    echo "     opsmaster backup mongodb import 'mongodb://root:senha@localhost:27017' all_export_20240225_150226"
    echo
    echo "  8. Import de um banco específico:"
    echo "     opsmaster backup mongodb import 'mongodb://usuario:senha@servidor:27017' all_export_20240225_150226 meudb"
    echo
    echo "  9. Listar backups disponíveis:"
    echo "     opsmaster backup mongodb list"
    echo
    echo "  10. Instalar dependências:"
    echo "     opsmaster backup mongodb install-deps"
    echo
    echo "Exemplos para backup/restore em um único comando:"
    echo "  1. Usando variáveis de ambiente:"
    echo "     export MONGODB_SOURCE_URI='mongodb+srv://usuario:senha@origem.mongodb.net'"
    echo "     export MONGODB_DEST_URI='mongodb+srv://usuario:senha@destino.mongodb.net'"
    echo "     opsmaster backup mongodb backup"
    echo
    echo "  2. Especificando URIs diretamente:"
    echo "     opsmaster backup mongodb backup --source 'mongodb+srv://usuario:senha@origem.mongodb.net' \\"
    echo "                                     --dest 'mongodb+srv://usuario:senha@destino.mongodb.net'"
    echo
    echo "  3. Backup de banco específico usando export:"
    echo "     opsmaster backup mongodb backup --source 'mongodb://origem:27017' \\"
    echo "                                     --dest 'mongodb://destino:27017' \\"
    echo "                                     --mode export \\"
    echo "                                     --db meudb"
    echo
    echo "  4. Backup com mapeamento de nomes de bancos:"
    echo "     opsmaster backup mongodb backup --source 'mongodb://origem:27017' \\"
    echo "                                     --dest 'mongodb://destino:27017' \\"
    echo "                                     --db-mapping 'fabio:fabio-coelho,teste:teste-prod'"
    echo
    echo "Notas sobre diretórios e caminhos:"
    echo "  1. Diretório de backup (dir):"
    echo "     - Se não especificado, usa o valor de BACKUP_DIR"
    echo "     - Pode ser um caminho absoluto ou relativo"
    echo "     - Exemplo: '/caminho/para/backup' ou './backups'"
    echo
    echo "  2. Caminho do backup (path):"
    echo "     - Para restore/import, pode ser:"
    echo "       a) Nome do diretório de backup (ex: all_20240225_150226)"
    echo "       b) Caminho absoluto (ex: /caminho/para/backup/all_20240225_150226)"
    echo "     - Se for apenas o nome do diretório, será procurado em BACKUP_DIR"
    echo "     - Se for caminho absoluto, será usado diretamente"
    echo
    echo "Formatos de URI suportados:"
    echo "  MongoDB Atlas:"
    echo "    - mongodb+srv://usuario:senha@seu-cluster.mongodb.net"
    echo "    - mongodb+srv://usuario:senha@seu-cluster.mongodb.net/admin"
    echo
    echo "  MongoDB Local/Remoto:"
    echo "    - mongodb://localhost:27017                     (sem autenticação)"
    echo "    - mongodb://usuario:senha@localhost:27017       (com autenticação básica)"
    echo "    - mongodb://usuario:senha@servidor:27017/admin  (com banco específico)"
    echo
    echo "Notas:"
    echo "  - Os backups são salvos em diretórios com o formato: <banco>_<data>_<hora>"
    echo "  - Para backups completos, usa-se o prefixo 'all' no nome do diretório"
    echo "  - O comando list mostra o nome e tamanho de cada backup disponível"
    echo "  - Para MongoDB Atlas, é necessário ter o IP do servidor liberado no firewall"
    echo "  - Para MongoDB Atlas, o pacote dnspython é instalado automaticamente"
    echo "  - As dependências podem ser instaladas automaticamente em sistemas Debian, Ubuntu e RHEL"
    echo "  - O export/import usa formato JSON para cada collection"
    echo
}

# Função principal
main() {
    local action="$1"
    local uri="$2"
    local param3="$3"    # Banco ou caminho do backup
    local param4="$4"    # Banco para restore/import ou diretório para dump/export
    local param5="$5"    # Mapeamento de bancos
    
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
            do_restore "$uri" "$param3" "$param4" "$param5"  # param4: banco (opcional), param5: mapeamento (opcional)
            ;;
        export)
            if [ -z "$uri" ]; then
                log_error "URI do MongoDB não especificada"
                show_help
                exit 1
            fi
            do_export "$uri" "$param3" "$param4"  # param3: banco (opcional), param4: diretório (opcional)
            ;;
        import)
            if [ -z "$uri" ] || [ -z "$param3" ]; then
                log_error "URI do MongoDB ou caminho do backup não especificado"
                show_help
                exit 1
            fi
            do_import "$uri" "$param3" "$param4" "$param5"  # param4: banco (opcional), param5: mapeamento (opcional)
            ;;
        backup)
            local source_uri=""
            local dest_uri=""
            local mode="dump"
            local db=""
            local db_mapping=""
            
            # Processar argumentos
            shift
            while [ $# -gt 0 ]; do
                case "$1" in
                    --source)
                        source_uri="$2"
                        shift 2
                        ;;
                    --dest)
                        dest_uri="$2"
                        shift 2
                        ;;
                    --mode)
                        if [ "$2" != "dump" ] && [ "$2" != "export" ]; then
                            log_error "Modo inválido: $2"
                            log_error "Use 'dump' ou 'export'"
                            exit 1
                        fi
                        mode="$2"
                        shift 2
                        ;;
                    --db)
                        db="$2"
                        shift 2
                        ;;
                    --db-mapping)
                        db_mapping="$2"
                        shift 2
                        ;;
                    *)
                        log_error "Argumento desconhecido: $1"
                        show_help
                        exit 1
                        ;;
                esac
            done
            
            do_backup_restore "$mode" "$source_uri" "$dest_uri" "$db" "$db_mapping"
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