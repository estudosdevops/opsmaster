#!/usr/bin/env bash

# --- Configuração ---
# Define as variáveis em um único lugar para facilitar a alteração.

# Variáveis do OpsMaster e Argo CD
OPSMASTER_BIN="./opsmaster"
ARGO_CONTEXT="staging" # O nome do contexto no seu ~/.opsmaster.yaml

# Variáveis da Aplicação
APP_NAME="sample-api-stg"
PROJECT_NAME="sample-api"
DEST_NAMESPACE="sample-api-stg"
REPO_URL="https://github.com/estudosdevops/sample-api.git"
REPO_PATH="chart"
VALUES_FILE="values-stg.yaml"
IMAGE_REPO="fcruzcoelho/sample-api"
IMAGE_TAG="v0.1.0"
CHART_DEPENDENCY="generic-app"

# --- Funções de Ajuda ---

# Função para imprimir uma mensagem de etapa com cor
step() {
  echo -e "\n\e[1;34m>>> Etapa: $1\e[0m"
}

# --- Execução do Teste ---
step "Adicionando o repositório Git"
$OPSMASTER_BIN --context $ARGO_CONTEXT argocd repo add $REPO_URL
sleep 5s

step "Criando o projeto no Argo CD"
$OPSMASTER_BIN --context $ARGO_CONTEXT argocd project create $PROJECT_NAME \
    --description "Uma API web simples em Go" \
    --source-repo $REPO_URL
sleep 5s

step "Criando a aplicação (Deploy)"
$OPSMASTER_BIN --context $ARGO_CONTEXT argocd app create \
    --app-name "$APP_NAME" \
    --project "$PROJECT_NAME" \
    --dest-namespace "$DEST_NAMESPACE" \
    --repo-url "$REPO_URL" \
    --repo-path "$REPO_PATH" \
    --values "$VALUES_FILE" \
    --set-image-repo "$IMAGE_REPO" \
    --set-image-tag "$IMAGE_TAG" \
    --set-chart-dependency "$CHART_DEPENDENCY"
sleep 5s

step "Sincronizando a aplicação"
$OPSMASTER_BIN argocd app sync "$APP_NAME"

step "Aguardando a aplicação ficar saudável e sincronizada"
$OPSMASTER_BIN --context $ARGO_CONTEXT argocd app wait "$APP_NAME" --timeout 1m
sleep 5s

step "Buscando detalhes da aplicação"
$OPSMASTER_BIN --context $ARGO_CONTEXT argocd app get "$APP_NAME"

echo -e "\n\e[1;32m✅ Teste de deploy concluído com sucesso! A aplicação está no ar.\e[0m"
# shellcheck disable=SC2162
read -p "Pressione Enter para apagar os recursos e limpar o ambiente..."

# --- Limpeza ---

step "Apagando a aplicação"
$OPSMASTER_BIN --context $ARGO_CONTEXT argocd app delete "$APP_NAME"
sleep 5s

step "Apagando o projeto"
$OPSMASTER_BIN --context $ARGO_CONTEXT argocd project delete $PROJECT_NAME
sleep 5s

step "Apagando o repositório"
$OPSMASTER_BIN --context $ARGO_CONTEXT argocd repo delete $REPO_URL

echo -e "\n\e[1;32m🧹 Ambiente limpo!\e[0m"