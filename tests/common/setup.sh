#!/usr/bin/env bash

# Configuração inicial para os testes
setup() {
    # Criar diretórios temporários
    export TEST_TMP_DIR=$(mktemp -d)
    export TEST_BACKUP_DIR="$TEST_TMP_DIR/backups"
    export TEST_CONFIG_DIR="$TEST_TMP_DIR/config"
    
    mkdir -p "$TEST_BACKUP_DIR"
    mkdir -p "$TEST_CONFIG_DIR"
    
    # Configurar variáveis de ambiente para testes
    export BACKUP_DIR="$TEST_BACKUP_DIR"
    export CONFIG_FILE="$TEST_CONFIG_DIR/test_config.yaml"
    
    # Criar arquivo de configuração de teste
    cat > "$CONFIG_FILE" << EOF
source:
  host: localhost
  port: 5432
  database: test_source
  username: test_user
  password: test_pass

target:
  host: localhost
  port: 5432
  database: test_target
  username: test_user
  password: test_pass

backup_dir: $TEST_BACKUP_DIR
EOF
}

# Limpeza após os testes
teardown() {
    # Remover diretórios temporários
    rm -rf "$TEST_TMP_DIR"
    
    # Limpar variáveis de ambiente
    unset TEST_TMP_DIR
    unset TEST_BACKUP_DIR
    unset TEST_CONFIG_DIR
    unset BACKUP_DIR
    unset CONFIG_FILE
} 