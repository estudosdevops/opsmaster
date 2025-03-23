#!/usr/bin/env bash

source "$(dirname "$(dirname "$(dirname "$0")")")/lib/common.sh"

# Verificar dependências necessárias
check_dependency "psql"

# Função para criar banco e usuário
create_database() {
    local db_name="$1"
    local db_user="$2"
    local db_password="$3"
    
    log_info "Criando banco de dados '$db_name' e usuário '$db_user'..."
    
    # Criar o banco de dados
    if ! sudo -u postgres psql -c "CREATE DATABASE $db_name;" >/dev/null 2>&1; then
        log_error "Falha ao criar o banco de dados '$db_name'"
        exit 1
    fi
    
    # Criar o usuário com senha
    if ! sudo -u postgres psql -c "CREATE USER $db_user WITH ENCRYPTED PASSWORD '$db_password';" >/dev/null 2>&1; then
        log_error "Falha ao criar o usuário '$db_user'"
        # Limpar banco criado em caso de erro
        sudo -u postgres psql -c "DROP DATABASE $db_name;" >/dev/null 2>&1
        exit 1
    fi
    
    # Garantir que o usuário só tenha acesso ao seu banco específico
    sudo -u postgres psql -c "REVOKE ALL ON ALL TABLES IN SCHEMA public FROM $db_user;" >/dev/null 2>&1
    sudo -u postgres psql -c "REVOKE ALL ON ALL SEQUENCES IN SCHEMA public FROM $db_user;" >/dev/null 2>&1
    sudo -u postgres psql -c "REVOKE ALL ON ALL FUNCTIONS IN SCHEMA public FROM $db_user;" >/dev/null 2>&1
    
    # Conceder privilégios específicos apenas no banco do usuário
    sudo -u postgres psql -c "GRANT ALL PRIVILEGES ON DATABASE $db_name TO $db_user;" >/dev/null 2>&1
    sudo -u postgres psql -d "$db_name" -c "GRANT ALL ON SCHEMA public TO $db_user;" >/dev/null 2>&1
    sudo -u postgres psql -d "$db_name" -c "ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO $db_user;" >/dev/null 2>&1
    sudo -u postgres psql -d "$db_name" -c "ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO $db_user;" >/dev/null 2>&1
    
    log_info "✅ Banco '$db_name' e usuário '$db_user' criados com sucesso!"
    log_info "Detalhes da conexão:"
    echo "----------------------------------------"
    echo "Host:     localhost"
    echo "Porta:    5432"
    echo "Banco:    $db_name"
    echo "Usuário:  $db_user"
    echo "Senha:    $db_password"
    echo "----------------------------------------"
    echo "String de conexão: postgresql://$db_user:$db_password@localhost:5432/$db_name"
}

# Função para listar bancos
list_databases() {
    log_info "Bancos de dados disponíveis:"
    echo "----------------------------------------"
    echo "Nome                            Tamanho"
    echo "----------------------------------------"
    sudo -u postgres psql -t -A -F";" -c "
        SELECT d.datname, 
               pg_size_pretty(pg_database_size(d.datname)) as size
        FROM pg_database d
        WHERE d.datname NOT IN ('template0', 'template1', 'postgres')
        ORDER BY d.datname;" | \
    while IFS=';' read -r db size; do
        printf "%-30s %10s\n" "$db" "$size"
    done
    echo "----------------------------------------"
}

# Função para remover banco e usuário
drop_database() {
    local db_name="$1"
    local db_user="$2"
    
    log_info "Removendo banco '$db_name' e usuário '$db_user'..."
    
    # Remover banco
    if ! sudo -u postgres psql -c "DROP DATABASE IF EXISTS $db_name;" >/dev/null 2>&1; then
        log_error "Falha ao remover o banco '$db_name'"
        exit 1
    fi
    
    # Remover usuário
    if ! sudo -u postgres psql -c "DROP USER IF EXISTS $db_user;" >/dev/null 2>&1; then
        log_error "Falha ao remover o usuário '$db_user'"
        exit 1
    fi
    
    log_info "✅ Banco '$db_name' e usuário '$db_user' removidos com sucesso!"
}

# Função de ajuda
show_help() {
    echo "Gerenciamento de Bancos PostgreSQL"
    echo
    echo "Uso: opsmaster infra postgres-db <ação> [argumentos]"
    echo
    echo "Ações:"
    echo "  create <nome> <senha>    Cria banco e usuário com o mesmo nome"
    echo "  drop <nome>             Remove banco e usuário"
    echo "  list                    Lista bancos existentes"
    echo
    echo "Exemplos:"
    echo "  opsmaster infra postgres-db create billing minhasenha"
    echo "  opsmaster infra postgres-db drop billing"
    echo "  opsmaster infra postgres-db list"
    echo
    echo "Notas:"
    echo "  - Requer sudo ou ser executado como usuário postgres"
    echo "  - O usuário criado terá acesso apenas ao seu banco específico"
    echo "  - A senha deve seguir a política de segurança do PostgreSQL"
    echo
}

# Função principal
main() {
    local action="$1"
    local db_name="$2"
    local db_password="$3"
    
    case "$action" in
        create)
            if [ -z "$db_name" ] || [ -z "$db_password" ]; then
                log_error "Nome do banco e senha são obrigatórios"
                show_help
                exit 1
            fi
            create_database "$db_name" "$db_name" "$db_password"
            ;;
        drop)
            if [ -z "$db_name" ]; then
                log_error "Nome do banco é obrigatório"
                show_help
                exit 1
            fi
            drop_database "$db_name" "$db_name"
            ;;
        list)
            list_databases
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