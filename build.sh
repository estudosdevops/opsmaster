#!/bin/bash

# --- Script de Build para o OpsMaster ---
# Este script compila a aplicação Go, gerando um binário otimizado.

# Define o nome do binário final que será gerado.
BINARY_NAME="opsmaster"

echo ">>> Sincronizando dependências..."
# Executa 'go mod tidy' para garantir que os arquivos go.mod e go.sum
# estão atualizados e que todas as dependências foram baixadas.
go mod tidy

echo ">>> Iniciando o build do $BINARY_NAME..."

# Executa o comando 'go build'.
# -o $BINARY_NAME: Define o nome do arquivo de saída como "opsmaster".
# -ldflags="-s -w": São flags que instruem o compilador a remover
#                   informações de depuração e a tabela de símbolos,
#                   o que reduz drasticamente o tamanho do binário final.
go build -ldflags="-s -w" -o $BINARY_NAME .

# Verifica se o comando de build foi bem-sucedido.
# shellcheck disable=SC2181
if [ $? -eq 0 ]; then
    echo ""
    echo "✅ Build concluído com sucesso!"
    echo "   Binário gerado: ./$BINARY_NAME"
    echo "   Para executar: ./$BINARY_NAME --help"
else
    echo ""
    echo "❌ Falha no build."
    exit 1
fi