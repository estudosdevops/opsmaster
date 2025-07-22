#!/bin/bash

# --- Script para Executar os Testes do OpsMaster ---

echo -e "\n\e[1;34m>>> Executando testes unitários e de integração para os pacotes 'internal/'...\e[0m"

# go list ./... | grep -v '/cmd' | xargs gotestsum --format=short-verbose -- -cover
go test -race -cover ./internal/...

if [ $? -ne 0 ]; then
  echo -e "\n\e[1;31m❌ Testes falharam. O push foi abortado.\e[0m"
  exit 1
fi

echo -e "\n\e[1;32m✅ Todos os testes passaram com sucesso.\e[0m"
exit 0
