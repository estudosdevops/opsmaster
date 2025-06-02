#!/usr/bin/env bats

load "common/setup"
load "common/utils"

@test "dump postgresql deve criar arquivo de backup" {
    # Setup
    local backup_file="$TEST_BACKUP_DIR/test_backup.dump"
    
    # Execução
    run opsmaster backup postgresql dump --backup-file "$backup_file"
    
    # Verificação
    [ "$status" -eq 0 ]
    [ -f "$backup_file" ]
    backup_is_valid "$backup_file" "postgresql"
}

@test "dump postgresql deve usar diretório padrão se não especificado" {
    # Execução
    run opsmaster backup postgresql dump
    
    # Verificação
    [ "$status" -eq 0 ]
    [ -d "$TEST_BACKUP_DIR" ]
    [ "$(ls -1 "$TEST_BACKUP_DIR" | wc -l)" -eq 1 ]
}

@test "dump postgresql deve falhar se banco não existir" {
    # Setup
    local backup_file="$TEST_BACKUP_DIR/test_backup.dump"
    
    # Execução
    run opsmaster backup postgresql dump --source-db nonexistent --backup-file "$backup_file"
    
    # Verificação
    [ "$status" -ne 0 ]
    [ ! -f "$backup_file" ]
}

@test "dump postgresql deve falhar se usuário não tiver permissão" {
    # Setup
    local backup_file="$TEST_BACKUP_DIR/test_backup.dump"
    
    # Execução
    run opsmaster backup postgresql dump --source-user nonexistent --backup-file "$backup_file"
    
    # Verificação
    [ "$status" -ne 0 ]
    [ ! -f "$backup_file" ]
}

@test "dump postgresql deve falhar se senha estiver incorreta" {
    # Setup
    local backup_file="$TEST_BACKUP_DIR/test_backup.dump"
    
    # Execução
    run opsmaster backup postgresql dump --source-pass wrong --backup-file "$backup_file"
    
    # Verificação
    [ "$status" -ne 0 ]
    [ ! -f "$backup_file" ]
}

@test "dump postgresql deve falhar se host não estiver acessível" {
    # Setup
    local backup_file="$TEST_BACKUP_DIR/test_backup.dump"
    
    # Execução
    run opsmaster backup postgresql dump --source-host nonexistent --backup-file "$backup_file"
    
    # Verificação
    [ "$status" -ne 0 ]
    [ ! -f "$backup_file" ]
}

@test "dump postgresql deve falhar se porta não estiver acessível" {
    # Setup
    local backup_file="$TEST_BACKUP_DIR/test_backup.dump"
    
    # Execução
    run opsmaster backup postgresql dump --source-port 9999 --backup-file "$backup_file"
    
    # Verificação
    [ "$status" -ne 0 ]
    [ ! -f "$backup_file" ]
}

@test "dump postgresql deve criar diretório de backup se não existir" {
    # Setup
    local backup_dir="$TEST_BACKUP_DIR/new_dir"
    local backup_file="$backup_dir/test_backup.dump"
    
    # Execução
    run opsmaster backup postgresql dump --backup-file "$backup_file"
    
    # Verificação
    [ "$status" -eq 0 ]
    [ -d "$backup_dir" ]
    [ -f "$backup_file" ]
}

@test "dump postgresql deve manter permissões corretas do arquivo" {
    # Setup
    local backup_file="$TEST_BACKUP_DIR/test_backup.dump"
    
    # Execução
    run opsmaster backup postgresql dump --backup-file "$backup_file"
    
    # Verificação
    [ "$status" -eq 0 ]
    [ -f "$backup_file" ]
    file_has_permissions "$backup_file" "600"
}

@test "dump postgresql deve criar arquivo com tamanho maior que zero" {
    # Setup
    local backup_file="$TEST_BACKUP_DIR/test_backup.dump"
    
    # Execução
    run opsmaster backup postgresql dump --backup-file "$backup_file"
    
    # Verificação
    [ "$status" -eq 0 ]
    [ -f "$backup_file" ]
    [ "$(stat -c "%s" "$backup_file")" -gt 0 ]
}

@test "dump postgresql deve criar arquivo recentemente" {
    # Setup
    local backup_file="$TEST_BACKUP_DIR/test_backup.dump"
    
    # Execução
    run opsmaster backup postgresql dump --backup-file "$backup_file"
    
    # Verificação
    [ "$status" -eq 0 ]
    [ -f "$backup_file" ]
    file_was_modified_recently "$backup_file" 5
} 