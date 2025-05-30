#!/usr/bin/env bats

# Carrega funções auxiliares
load 'test_helper'

# Configuração inicial
setup() {
    # Inicia os containers
    docker-compose up -d
    
    # Aguarda os containers estarem prontos
    wait_for_postgres postgres_source 5432
    wait_for_postgres postgres_target 5433
    
    # Cria dados de teste
    docker exec -i postgres_source psql -U postgres -d sourcedb < init-scripts/01-init-data.sql
}

# Limpeza após os testes
teardown() {
    # Para os containers
    docker-compose down -v
}

# Teste: Verifica se o script cria o arquivo de configuração
@test "init-config cria arquivo de configuração" {
    run opsmaster backup postgresql init-config
    
    [ "$status" -eq 0 ]
    [ -f "$HOME/.config/backups/postgresql.yaml" ]
}

# Teste: Verifica se o dump é criado corretamente
@test "dump cria arquivo de backup" {
    run opsmaster backup postgresql dump \
        --source-host localhost \
        --source-port 5432 \
        --source-db sourcedb \
        --source-user postgres \
        --source-pass sourcepass123
    
    [ "$status" -eq 0 ]
    [ -f "$HOME/backups/postgresql/sourcedb_"*".dump" ]
}

# Teste: Verifica se o restore funciona corretamente
@test "restore restaura dados corretamente" {
    # Primeiro faz o dump
    run opsmaster backup postgresql dump \
        --source-host localhost \
        --source-port 5432 \
        --source-db sourcedb \
        --source-user postgres \
        --source-pass sourcepass123
    
    [ "$status" -eq 0 ]
    local backup_file
    backup_file=$(ls -t "$HOME/backups/postgresql/sourcedb_"*".dump" | head -1)
    
    # Depois faz o restore
    run opsmaster backup postgresql restore \
        --target-host localhost \
        --target-port 5433 \
        --target-db targetdb \
        --target-user postgres \
        --target-pass targetpass123 \
        --backup-file "$backup_file"
    
    [ "$status" -eq 0 ]
    
    # Verifica se os dados foram restaurados
    run docker exec -i postgres_target psql -U postgres -d targetdb -c "SELECT COUNT(*) FROM users;"
    [ "$status" -eq 0 ]
    [ "$output" -gt 0 ]
}

# Teste: Verifica se o sync funciona corretamente
@test "sync realiza dump e restore em uma única operação" {
    run opsmaster backup postgresql sync \
        --source-host localhost \
        --source-port 5432 \
        --source-db sourcedb \
        --source-user postgres \
        --source-pass sourcepass123 \
        --target-host localhost \
        --target-port 5433 \
        --target-db targetdb \
        --target-user postgres \
        --target-pass targetpass123
    
    [ "$status" -eq 0 ]
    
    # Verifica se os dados foram sincronizados
    run docker exec -i postgres_target psql -U postgres -d targetdb -c "SELECT COUNT(*) FROM users;"
    [ "$status" -eq 0 ]
    [ "$output" -gt 0 ]
}

# Teste: Verifica se o list mostra os backups
@test "list mostra backups disponíveis" {
    # Cria alguns backups
    opsmaster backup postgresql dump \
        --source-host localhost \
        --source-port 5432 \
        --source-db sourcedb \
        --source-user postgres \
        --source-pass sourcepass123
    
    run opsmaster backup postgresql list
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "sourcedb_" ]]
}

# Teste: Verifica se o script valida parâmetros obrigatórios
@test "falha quando parâmetros obrigatórios não são fornecidos" {
    run opsmaster backup postgresql dump
    
    [ "$status" -ne 0 ]
    [[ "$output" =~ "Parâmetros obrigatórios não fornecidos" ]]
}

# Teste: Verifica se o script valida conexão com o banco
@test "falha quando não consegue conectar ao banco" {
    run opsmaster backup postgresql dump \
        --source-host localhost \
        --source-port 9999 \
        --source-db sourcedb \
        --source-user postgres \
        --source-pass sourcepass123
    
    [ "$status" -ne 0 ]
    [[ "$output" =~ "Falha ao realizar dump" ]]
} 