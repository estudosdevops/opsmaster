#!/usr/bin/env bash

# Função para aguardar o PostgreSQL estar pronto
wait_for_postgres() {
    local container="$1"
    local port="$2"
    local max_attempts=30
    local attempt=1

    echo "Aguardando PostgreSQL em $container:$port..."
    while [ $attempt -le $max_attempts ]; do
        if docker exec "$container" pg_isready -h localhost -p "$port" >/dev/null 2>&1; then
            echo "PostgreSQL está pronto!"
            return 0
        fi
        echo "Tentativa $attempt de $max_attempts..."
        sleep 1
        attempt=$((attempt + 1))
    done

    echo "Timeout aguardando PostgreSQL"
    return 1
}

# Função para verificar se um comando existe
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Função para verificar se o BATS está instalado
check_bats() {
    if ! command_exists bats; then
        echo "BATS não está instalado. Instalando..."
        if command_exists apt-get; then
            sudo apt-get update
            sudo apt-get install -y bats
        elif command_exists brew; then
            brew install bats-core
        else
            echo "Não foi possível instalar o BATS automaticamente"
            echo "Por favor, instale manualmente: https://github.com/bats-core/bats-core"
            exit 1
        fi
    fi
}

# Função para executar os testes
run_tests() {
    check_bats
    bats "$@"
} 