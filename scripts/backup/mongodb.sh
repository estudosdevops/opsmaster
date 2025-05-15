#!/usr/bin/env bash
# Description: Gerencia backup e restore de bancos MongoDB

# shellcheck disable=SC1091
source "/usr/local/lib/opsmaster/common.sh"

# Configurações padrão
BACKUP_DIR="${BACKUP_DIR:-/tmp/mongodb_backups}"
DATE_FORMAT=$(date +%Y%m%d_%H%M%S)

# Variáveis de ambiente para URIs do MongoDB
MONGODB_SOURCE_URI="${MONGODB_SOURCE_URI:-}"
MONGODB_TARGET_URI="${MONGODB_TARGET_URI:-}"

# Função para verificar dependências necessárias
check_mongodb_deps() {
    # Verificar ferramentas MongoDB
    check_dependency "mongodump" "mongorestore" "mongosh" "mongoexport" "mongoimport"
    
    # Verificar se o dnspython está instalado (necessário para mongodb+srv)
    if ! python3 -c "import dns" &>/dev/null; then
        log_error "dnspython não está instalado. Este pacote é necessário para conexões mongodb+srv"
        log_info "Execute 'opsmaster backup mongodb install-deps' para instalar"
        exit 1
    fi
}

# Função para obter lista de bancos excluindo os do sistema
get_user_databases() {
    local uri="$1"
    local host port username password auth_db
    parse_mongodb_uri "$uri" host port username password auth_db
    
    # Comando para listar bancos de dados
    local list_cmd="db.getMongo().getDBs().databases
        .filter(db => !['admin', 'local', 'config'].includes(db.name))
        .filter(db => db.sizeOnDisk > 0)
        .map(db => db.name)"
    
    execute_mongo_command "$host" "$port" "$username" "$password" "$list_cmd" | tr -d '[],"'
}

# Função para verificar permissões do usuário
check_mongodb_permissions() {
    local uri="$1"
    local host port username password auth_db
    parse_mongodb_uri "$uri" host port username password auth_db
    
    log_info "Verificando permissões do usuário..."
    
    # Comando para verificar permissões
    local check_cmd="db.adminCommand({listDatabases: 1})"
    
    if ! execute_mongo_command "$host" "$port" "$username" "$password" "$check_cmd" >/dev/null 2>&1; then
        log_error "Usuário sem permissões suficientes. É necessário:"
        log_error "- Ser usuário root ou"
        log_error "- Ter role 'backup' ou 'readAnyDatabase' para backup"
        log_error "- Ter role 'restore' ou 'readWriteAnyDatabase' para restore"
        return 1
    fi
    
    return 0
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
    
    # Construir URI baseada no tipo de conexão
    local uri
    if [[ "$host" == *".mongodb.net" ]]; then
        # MongoDB Atlas
        if [ -n "$username" ] && [ -n "$password" ]; then
            uri="mongodb+srv://$username:$password@$host"
        else
            uri="mongodb+srv://$host"
        fi
    else
        # MongoDB Local/Remoto
        if [ -n "$username" ] && [ -n "$password" ]; then
            uri="mongodb://$username:$password@$host:$port"
        else
            uri="mongodb://$host:$port"
        fi
    fi
    
    # Executar comando com a URI construída
    mongosh "$uri" --quiet --eval "$command"
}

# Função para testar conexão com MongoDB
test_mongodb_connection() {
    local uri="$1"
    local host port username password auth_db
    parse_mongodb_uri "$uri" host port username password auth_db
    
    # Mostrar mensagem apropriada baseada no tipo de conexão
    if [[ "$uri" == *"mongodb+srv://"* ]]; then
        log_info "Testando conexão com MongoDB Atlas em $host..."
    else
        log_info "Testando conexão com MongoDB em $host:$port..."
    fi
    
    # Comando para testar a conexão
    local test_cmd="db.runCommand({ping: 1})"
    
    # Testar conexão usando execute_mongo_command
    if ! execute_mongo_command "$host" "$port" "$username" "$password" "$test_cmd" >/dev/null 2>&1; then
        log_error "Não foi possível conectar ao MongoDB"
        log_error "Verifique:"
        log_error "- Se o servidor está acessível"
        log_error "- Se as credenciais estão corretas"
        log_error "- Se o usuário tem as permissões necessárias"
        if [[ "$uri" == *"mongodb+srv://"* ]]; then
            log_error "- Se o IP do seu servidor está liberado no MongoDB Atlas"
        fi
        return 1
    fi
    
    log_info "Conexão estabelecida com sucesso"
    return 0
}

# Função para construir comando base MongoDB
build_mongo_command() {
    local cmd="$1"
    local host="$2"
    local port="$3"
    local username="$4"
    local password="$5"
    local auth_db="$6"
    
    # Construir URI baseada no tipo de conexão
    local uri
    if [[ "$host" == *".mongodb.net" ]]; then
        # MongoDB Atlas
        if [ -n "$username" ] && [ -n "$password" ]; then
            uri="mongodb+srv://$username:$password@$host"
        else
            uri="mongodb+srv://$host"
        fi
    else
        # MongoDB Local/Remoto
        if [ -n "$username" ] && [ -n "$password" ]; then
            uri="mongodb://$username:$password@$host:$port"
        else
            uri="mongodb://$host:$port"
        fi
    fi
    
    # Retornar comando com a URI construída
    echo "$cmd --uri $uri"
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

# Função para realizar dump
do_dump() {
    local uri="${1:-$MONGODB_SOURCE_URI}"
    local specific_db="$2"
    local custom_dir="$3"
    
    if [ -z "$uri" ]; then
        log_error "URI do MongoDB não especificada. Use o parâmetro ou defina MONGODB_SOURCE_URI"
        show_help
        exit 1
    fi
    
    # Testar conexão antes de prosseguir
    if ! test_mongodb_connection "$uri"; then
        exit 1
    fi
    
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
        
        # Construir comando para este banco
        local cmd="mongodump"
        
        # Adicionar parâmetros de conexão
        if [[ "$host" == *".mongodb.net" ]]; then
            # MongoDB Atlas
            if [ -n "$username" ] && [ -n "$password" ]; then
                cmd="$cmd --uri mongodb+srv://$username:$password@$host"
            else
                cmd="$cmd --uri mongodb+srv://$host"
            fi
        else
            # MongoDB Local/Remoto
            cmd="$cmd --host $host --port $port"
            if [ -n "$username" ] && [ -n "$password" ]; then
                cmd="$cmd --username $username --password $password --authenticationDatabase $auth_db"
            fi
        fi
        
        # Adicionar parâmetros de dump
        cmd="$cmd --db=$db --out=$output_dir"
        
        if eval "$cmd" >/dev/null 2>&1; then
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
    local source_uri="${1:-$MONGODB_SOURCE_URI}"
    local backup_path="$2"
    local specific_db="$3"
    local dest_db="$4"
    local target_uri="${5:-$MONGODB_TARGET_URI}"
    
    # Se target_uri não for especificado, usar source_uri
    target_uri="${target_uri:-$source_uri}"
    
    if [ -z "$source_uri" ]; then
        log_error "URI do MongoDB de origem não especificada. Use o parâmetro ou defina MONGODB_SOURCE_URI"
        show_help
        exit 1
    fi
    
    # Verificar dependências antes de prosseguir
    check_mongodb_deps
    
    # Testar conexão com origem antes de prosseguir
    if ! test_mongodb_connection "$source_uri"; then
        exit 1
    fi
    
    # Testar conexão com destino antes de prosseguir
    if ! test_mongodb_connection "$target_uri"; then
        exit 1
    fi
    
    # Extrair credenciais da URI de destino
    local host port username password auth_db
    parse_mongodb_uri "$target_uri" host port username password auth_db
    
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
        
        # Usar o nome do banco de destino se fornecido, senão usar o nome original
        local target_db="${dest_db:-$specific_db}"
        log_info "Restaurando banco $specific_db para $target_db"
        
        # Construir comando para este banco
        local cmd="mongorestore"
        
        # Adicionar parâmetros de conexão
        if [[ "$host" == *".mongodb.net" ]]; then
            # MongoDB Atlas
            if [ -n "$username" ] && [ -n "$password" ]; then
                cmd="$cmd --uri mongodb+srv://$username:$password@$host"
            else
                cmd="$cmd --uri mongodb+srv://$host"
            fi
        else
            # MongoDB Local/Remoto
            cmd="$cmd --host $host --port $port"
            if [ -n "$username" ] && [ -n "$password" ]; then
                cmd="$cmd --username $username --password $password --authenticationDatabase $auth_db"
            fi
        fi
        
        # Adicionar parâmetros de restore
        cmd="$cmd --db=$target_db $db_path"
        
        if eval "$cmd" >/dev/null 2>&1; then
            log_info "Restore do banco $specific_db para $target_db concluído com sucesso"
        else
            log_error "Falha no restore do banco $specific_db para $target_db"
            exit 1
        fi
    else
        # Restore de todos os bancos
        log_info "Restaurando todos os bancos do backup..."
        
        # Encontrar todos os diretórios de bancos de dados no backup
        for db_path in "$backup_path"/*; do
            if [ -d "$db_path" ]; then
                local db_name
                db_name=$(basename "$db_path")
                
                # Pular bancos do sistema
                if [[ "$db_name" =~ ^(admin|local|config)$ ]]; then
                    log_warn "Pulando banco do sistema: $db_name"
                    continue
                fi
                
                log_info "Restaurando banco: $db_name"
                
                # Construir comando para este banco
                local cmd="mongorestore"
                
                # Adicionar parâmetros de conexão
                if [[ "$host" == *".mongodb.net" ]]; then
                    # MongoDB Atlas
                    if [ -n "$username" ] && [ -n "$password" ]; then
                        cmd="$cmd --uri mongodb+srv://$username:$password@$host"
                    else
                        cmd="$cmd --uri mongodb+srv://$host"
                    fi
                else
                    # MongoDB Local/Remoto
                    cmd="$cmd --host $host --port $port"
                    if [ -n "$username" ] && [ -n "$password" ]; then
                        cmd="$cmd --username $username --password $password --authenticationDatabase $auth_db"
                    fi
                fi
                
                # Adicionar parâmetros de restore
                cmd="$cmd --db=$db_name $db_path"
                
                if eval "$cmd" >/dev/null 2>&1; then
                    log_info "Restore do banco $db_name concluído com sucesso"
                else
                    log_error "Falha no restore do banco $db_name"
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

# Função para realizar export
do_export() {
    local uri="${1:-$MONGODB_SOURCE_URI}"
    local specific_db="$2"
    local custom_dir="$3"
    
    if [ -z "$uri" ]; then
        log_error "URI do MongoDB não especificada. Use o parâmetro ou defina MONGODB_SOURCE_URI"
        show_help
        exit 1
    fi
    
    # Testar conexão antes de prosseguir
    if ! test_mongodb_connection "$uri"; then
        exit 1
    fi
    
    # Extrair credenciais da URI
    local host port username password auth_db
    parse_mongodb_uri "$uri" host port username password auth_db
    
    log_info "Conectando ao MongoDB em $host:$port"
    
    # Usar diretório personalizado se fornecido, senão usar BACKUP_DIR padrão
    local base_dir="${custom_dir:-$BACKUP_DIR}"
    local output_dir="$base_dir/${specific_db:-all_export}_${DATE_FORMAT}"
    
    # Criar diretório de saída
    if ! mkdir -p "$output_dir"; then
        log_error "Não foi possível criar o diretório: $output_dir"
        exit 1
    fi
    
    log_info "Diretório de destino: $output_dir"
    
    # Obter lista de bancos
    log_info "Obtendo lista de bancos de dados..."
    local databases
    
    if [ -n "$specific_db" ]; then
        check_database_exists "$host" "$port" "$username" "$password" "$specific_db"
        databases="$specific_db"
        log_info "Realizando export do banco específico: $specific_db"
    else
        local mongo_cmd="db.adminCommand('listDatabases').databases"
        mongo_cmd="$mongo_cmd.filter(db => !['admin', 'local', 'config'].includes(db.name))"
        mongo_cmd="$mongo_cmd.map(db => db.name)"
        
        databases=$(execute_mongo_command "$host" "$port" "$username" "$password" "$mongo_cmd" | tr -d '[],"')
        log_info "Realizando export de todos os bancos"
    fi
    
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
        log_info "Exportando todas as collections do banco $db"
        
        # Exportar cada collection
        for collection in $collections; do
            log_info "Exportando collection: $collection"
            
            # Construir comando para esta collection
            local cmd="mongoexport"
            
            # Adicionar parâmetros de conexão
            if [[ "$host" == *".mongodb.net" ]]; then
                # MongoDB Atlas
                if [ -n "$username" ] && [ -n "$password" ]; then
                    cmd="$cmd --uri mongodb+srv://$username:$password@$host"
                else
                    cmd="$cmd --uri mongodb+srv://$host"
                fi
            else
                # MongoDB Local/Remoto
                cmd="$cmd --host $host --port $port"
                if [ -n "$username" ] && [ -n "$password" ]; then
                    cmd="$cmd --username $username --password $password --authenticationDatabase $auth_db"
                fi
            fi
            
            # Adicionar parâmetros de export
            cmd="$cmd --db=$db --collection=$collection --out=$db_dir/${collection}.json"
            
            if eval "$cmd" >/dev/null 2>&1; then
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
    local source_uri="${1:-$MONGODB_SOURCE_URI}"
    local backup_path="$2"
    local specific_db="$3"
    local dest_db="$4"
    local target_uri="${5:-$MONGODB_TARGET_URI}"
    
    # Se target_uri não for especificado, usar source_uri
    target_uri="${target_uri:-$source_uri}"
    
    if [ -z "$source_uri" ]; then
        log_error "URI do MongoDB de origem não especificada. Use o parâmetro ou defina MONGODB_SOURCE_URI"
        show_help
        exit 1
    fi
    
    # Testar conexão antes de prosseguir
    if ! test_mongodb_connection "$source_uri"; then
        exit 1
    fi
    
    # Extrair credenciais da URI de destino
    local host port username password auth_db
    parse_mongodb_uri "$target_uri" host port username password auth_db
    
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
    
    log_info "Iniciando import MongoDB..."
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
        
        # Usar o nome do banco de destino se fornecido, senão usar o nome original
        local target_db="${dest_db:-$specific_db}"
        log_info "Importando banco $specific_db para $target_db"
        
        # Importar todas as collections do banco
        log_info "Importando todas as collections do banco $specific_db"
        for json_file in "$db_path"/*.json; do
            if [ -f "$json_file" ]; then
                local collection
                collection=$(basename "$json_file" .json)
                log_info "Importando collection: $collection"
                
                # Construir comando para esta collection
                local cmd="mongoimport"
                
                # Adicionar parâmetros de conexão
                if [[ "$host" == *".mongodb.net" ]]; then
                    # MongoDB Atlas
                    if [ -n "$username" ] && [ -n "$password" ]; then
                        cmd="$cmd --uri mongodb+srv://$username:$password@$host"
                    else
                        cmd="$cmd --uri mongodb+srv://$host"
                    fi
                else
                    # MongoDB Local/Remoto
                    cmd="$cmd --host $host --port $port"
                    if [ -n "$username" ] && [ -n "$password" ]; then
                        cmd="$cmd --username $username --password $password --authenticationDatabase $auth_db"
                    fi
                fi
                
                # Adicionar parâmetros de import
                cmd="$cmd --db=$target_db --collection=$collection --file=$json_file"
                
                if eval "$cmd" >/dev/null 2>&1; then
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
                local db_name
                db_name=$(basename "$db_path")
                
                # Pular bancos do sistema
                if [[ "$db_name" =~ ^(admin|local|config)$ ]]; then
                    log_warn "Pulando banco do sistema: $db_name"
                    continue
                fi
                
                log_info "Importando banco: $db_name"
                
                # Importar todas as collections do banco
                log_info "Importando todas as collections do banco $db_name"
                for json_file in "$db_path"/*.json; do
                    if [ -f "$json_file" ]; then
                        local collection
                        collection=$(basename "$json_file" .json)
                        log_info "Importando collection: $collection"
                        
                        # Construir comando para esta collection
                        local cmd="mongoimport"
                        
                        # Adicionar parâmetros de conexão
                        if [[ "$host" == *".mongodb.net" ]]; then
                            # MongoDB Atlas
                            if [ -n "$username" ] && [ -n "$password" ]; then
                                cmd="$cmd --uri mongodb+srv://$username:$password@$host"
                            else
                                cmd="$cmd --uri mongodb+srv://$host"
                            fi
                        else
                            # MongoDB Local/Remoto
                            cmd="$cmd --host $host --port $port"
                            if [ -n "$username" ] && [ -n "$password" ]; then
                                cmd="$cmd --username $username --password $password --authenticationDatabase $auth_db"
                            fi
                        fi
                        
                        # Adicionar parâmetros de import
                        cmd="$cmd --db=$db_name --collection=$collection --file=$json_file"
                        
                        if eval "$cmd" >/dev/null 2>&1; then
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

# Função para validar URI do MongoDB
validate_mongodb_uri() {
    local uri="$1"
    local context="$2"
    
    if [ -z "$uri" ]; then
        log_error "URI do MongoDB não especificada para $context"
        return 1
    fi
    
    # Verificar se a URI começa com mongodb:// ou mongodb+srv://
    if [[ ! "$uri" =~ ^mongodb(\+srv)?:// ]]; then
        log_error "URI do MongoDB inválida para $context: $uri"
        log_error "A URI deve começar com mongodb:// ou mongodb+srv://"
            return 1
    fi
    
    return 0
}

# Função para realizar sincronização via dump/restore
do_sync_dump() {
    local specific_db="$1"
    local dest_db="$2"
    local source_uri="${MONGODB_SOURCE_URI:-}"
    local target_uri="${MONGODB_TARGET_URI:-}"
    
    # Validar URIs
    if ! validate_mongodb_uri "$source_uri" "origem"; then
        log_error "Use o parâmetro ou defina MONGODB_SOURCE_URI corretamente"
        show_help
        exit 1
    fi
    
    if ! validate_mongodb_uri "$target_uri" "destino"; then
        log_error "Use o parâmetro ou defina MONGODB_TARGET_URI corretamente"
        show_help
        exit 1
    fi
    
    # Verificar dependências antes de prosseguir
    check_mongodb_deps

    # Testar conexão com origem/destino
    log_info "Testando conexão com MongoDB de origem..."
    if ! test_mongodb_connection "$source_uri"; then
        log_error "Falha ao conectar com MongoDB de origem"
        exit 1
    elif ! (log_info "Testando conexão com MongoDB de destino..." && test_mongodb_connection "$target_uri"); then
        log_error "Falha ao conectar com MongoDB de destino"
        exit 1
    fi

    # O script continua aqui apenas se ambas as conexões foram bem-sucedidas.
    log_info "Ambas as conexões MongoDB foram bem-sucedidas."

    # Verificar permissões na origem
    log_info "Verificando permissões no MongoDB de origem..."
    if ! check_mongodb_permissions "$source_uri"; then
        log_error "Usuário sem permissões suficientes no MongoDB de origem"
        exit 1
    fi
    
    # Verificar permissões no destino
    log_info "Verificando permissões no MongoDB de destino..."
    if ! check_mongodb_permissions "$target_uri"; then
        log_error "Usuário sem permissões suficientes no MongoDB de destino"
        exit 1
    fi
    
    # Criar diretório temporário para o dump
    local temp_dir
    temp_dir=$(mktemp -d)
    trap 'rm -rf "$temp_dir"' EXIT
    
    log_info "Iniciando sincronização MongoDB via dump/restore..."
    log_info "Origem: $source_uri"
    log_info "Destino: $target_uri"
    
    # Fazer dump
    log_info "Realizando dump do banco de origem..."
    if ! do_dump "$source_uri" "$specific_db" "$temp_dir"; then
        log_error "Falha no dump do banco de origem"
        exit 1
    fi
    
    # Encontrar o diretório do dump
    local dump_dir
    if [ -n "$specific_db" ]; then
        dump_dir="$temp_dir/${specific_db}_${DATE_FORMAT}"
    else
        dump_dir="$temp_dir/all_${DATE_FORMAT}"
    fi
    
    # Fazer restore
    log_info "Realizando restore no banco de destino..."
    if ! do_restore "$source_uri" "$dump_dir" "$specific_db" "$dest_db" "$target_uri"; then
        log_error "Falha no restore do banco de destino"
        exit 1
    fi
    
    log_info "Sincronização via dump/restore concluída com sucesso!"
}

# Função para realizar sincronização via export/import de collections
do_sync_collections() {
    local specific_db="$1"
    local dest_db="$2"
    local source_uri="${MONGODB_SOURCE_URI:-}"
    local target_uri="${MONGODB_TARGET_URI:-}"
    
    # Validar URIs
    if ! validate_mongodb_uri "$source_uri" "origem"; then
        log_error "Use o parâmetro ou defina MONGODB_SOURCE_URI corretamente"
        show_help
        exit 1
    fi
    
    if ! validate_mongodb_uri "$target_uri" "destino"; then
        log_error "Use o parâmetro ou defina MONGODB_TARGET_URI corretamente"
        show_help
        exit 1
    fi
    
    # Verificar dependências antes de prosseguir
    check_mongodb_deps

    # Testar conexão com origem/destino
    log_info "Testando conexão com MongoDB de origem..."
    if ! test_mongodb_connection "$source_uri"; then
        log_error "Falha ao conectar com MongoDB de origem"
        exit 1
    elif ! (log_info "Testando conexão com MongoDB de destino..." && test_mongodb_connection "$target_uri"); then
        log_error "Falha ao conectar com MongoDB de destino"
        exit 1
    fi

    # O script continua aqui apenas se ambas as conexões foram bem-sucedidas.
    log_info "Ambas as conexões MongoDB foram bem-sucedidas."
    
    # Verificar permissões na origem
    log_info "Verificando permissões no MongoDB de origem..."
    if ! check_mongodb_permissions "$source_uri"; then
        log_error "Usuário sem permissões suficientes no MongoDB de origem"
        exit 1
    fi
    
    # Verificar permissões no destino
    log_info "Verificando permissões no MongoDB de destino..."
    if ! check_mongodb_permissions "$target_uri"; then
        log_error "Usuário sem permissões suficientes no MongoDB de destino"
        exit 1
    fi
    
    # Criar diretório temporário para o export
    local temp_dir
    temp_dir=$(mktemp -d)
    trap 'rm -rf "$temp_dir"' EXIT
    
    log_info "Iniciando sincronização MongoDB via export/import de collections..."
    log_info "Origem: $source_uri"
    log_info "Destino: $target_uri"
    
    # Fazer export
    log_info "Realizando export do banco de origem..."
    if ! do_export "$source_uri" "$specific_db" "$temp_dir"; then
        log_error "Falha no export do banco de origem"
        exit 1
    fi
    
    # Encontrar o diretório do export
    local export_dir
    if [ -n "$specific_db" ]; then
        export_dir="$temp_dir/${specific_db}_${DATE_FORMAT}"
    else
        export_dir="$temp_dir/all_export_${DATE_FORMAT}"
    fi
    
    # Fazer import
    log_info "Realizando import no banco de destino..."
    if ! do_import "$source_uri" "$export_dir" "$specific_db" "$dest_db" "$target_uri"; then
        log_error "Falha no import do banco de destino"
        exit 1
    fi
    
    log_info "Sincronização via export/import de collections concluída com sucesso!"
}

# Função de ajuda
show_help() {
    echo "Gerenciamento de Backup/Restore MongoDB"
    echo
    echo "Uso: opsmaster backup mongodb <ação> [argumentos]"
    echo
    echo "Ações:"
    echo "  dump [uri] [banco] [dir]      Realiza dump (de um banco específico ou todos)"
    echo "  restore [uri_origem] <path> [banco] [banco_destino] [uri_destino]   Restaura backup"
    echo "  export [uri] [banco] [dir]    Realiza export JSON (de um banco específico ou todos)"
    echo "  import [uri_origem] <path> [banco] [banco_destino] [uri_destino]   Importa arquivos JSON"
    echo "  sync-dump [banco] [banco_destino]   Sincroniza via dump/restore entre MongoDBs"
    echo "  sync-collections [banco] [banco_destino]   Sincroniza via export/import de collections entre MongoDBs"
    echo "  list                           Lista backups disponíveis com seus tamanhos"
    echo "  install-deps                   Instala dependências necessárias"
    echo
    echo "Argumentos:"
    echo "  uri                           URI de conexão do MongoDB (opcional se MONGODB_SOURCE_URI estiver definida)"
    echo "  uri_origem                    URI do MongoDB de origem (opcional se MONGODB_SOURCE_URI estiver definida)"
    echo "  uri_destino                   URI do MongoDB de destino (opcional se MONGODB_TARGET_URI estiver definida)"
    echo "  banco                         Nome do banco de dados (opcional)"
    echo "  banco_destino                 Nome do banco de destino (opcional, diferente do banco de origem)"
    echo "  dir                           Diretório de destino para o dump/export (opcional)"
    echo "  path                          Caminho do backup para restore/import"
    echo
    echo "Variáveis de ambiente:"
    echo "  BACKUP_DIR                    Diretório para armazenar backups (padrão: /tmp/mongodb_backups)"
    echo "  MONGODB_SOURCE_URI            URI do MongoDB de origem (opcional)"
    echo "  MONGODB_TARGET_URI            URI do MongoDB de destino (opcional)"
    echo
    echo "Exemplos:"
    echo "  1. Sincronizar via dump/restore:"
    echo "     export MONGODB_SOURCE_URI='mongodb://usuario:senha@servidor1:27017'"
    echo "     export MONGODB_TARGET_URI='mongodb://usuario:senha@servidor2:27017'"
    echo "     opsmaster backup mongodb sync-dump dummy dummy2"
    echo
    echo "  2. Sincronizar via export/import de collections:"
    echo "     export MONGODB_SOURCE_URI='mongodb://usuario:senha@servidor1:27017'"
    echo "     export MONGODB_TARGET_URI='mongodb://usuario:senha@servidor2:27017'"
    echo "     opsmaster backup mongodb sync-collections dummy dummy2"
    echo
    echo "  3. Dump de um banco específico:"
    echo "     opsmaster backup mongodb dump 'mongodb://usuario:senha@servidor:27017' meudb"
    echo
    echo "  4. Restore de um banco específico com nome diferente:"
    echo "     opsmaster backup mongodb restore 'mongodb://usuario:senha@servidor:27017' all_20240225_150226 catalog si-catalog-svc"
    echo
    echo "Notas importantes:"
    echo "  - Para MongoDB Atlas, use o formato mongodb+srv:// (ex: mongodb+srv://usuario:senha@seu-cluster.mongodb.net)"
    echo "  - Para MongoDB local/remoto, use o formato mongodb:// (ex: mongodb://usuario:senha@servidor:27017)"
    echo "  - Certifique-se de que as variáveis de ambiente estão definidas corretamente"
    echo "  - Use aspas ao definir as variáveis de ambiente: export MONGODB_SOURCE_URI='mongodb://...'"
    echo "  - Os backups são salvos em diretórios com o formato: <banco>_<data>_<hora>"
    echo "  - Para backups completos, usa-se o prefixo 'all' no nome do diretório"
    echo "  - Para MongoDB Atlas, é necessário ter o IP do servidor liberado no firewall"
    echo "  - Para MongoDB Atlas, o pacote dnspython é instalado automaticamente"
    echo
}

# Função principal
main() {
    local action="$1"
    local param1="$2"    # Banco de origem
    local param2="$3"    # Banco de destino
    local param3="$4"    # URI de origem (opcional)
    local param4="$5"    # URI de destino (opcional)
    
    case "$action" in
        dump)
            do_dump "$param1" "$param2" "$param3"  # param1: uri (opcional), param2: banco (opcional), param3: diretório (opcional)
            ;;
        restore)
            if [ -z "$param2" ]; then
                log_error "Caminho do backup não especificado"
                show_help
                exit 1
            fi
            do_restore "$param1" "$param2" "$param3" "$param4" "$param5"  # param1: uri_origem (opcional), param2: path, param3: banco (opcional), param4: banco_destino (opcional), param5: uri_destino (opcional)
            ;;
        export)
            do_export "$param1" "$param2" "$param3"  # param1: uri (opcional), param2: banco (opcional), param3: diretório (opcional)
            ;;
        import)
            if [ -z "$param2" ]; then
                log_error "Caminho do backup não especificado"
                show_help
                exit 1
            fi
            do_import "$param1" "$param2" "$param3" "$param4" "$param5"  # param1: uri_origem (opcional), param2: path, param3: banco (opcional), param4: banco_destino (opcional), param5: uri_destino (opcional)
            ;;
        sync-dump)
            do_sync_dump "$param1" "$param2"  # param1: banco (opcional), param2: banco_destino (opcional)
            ;;
        sync-collections)
            do_sync_collections "$param1" "$param2"  # param1: banco (opcional), param2: banco_destino (opcional)
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