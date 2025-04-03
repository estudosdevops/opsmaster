#!/usr/bin/env bash
# Description: Gerencia aplicações e projetos no ArgoCD

source "/usr/local/lib/opsmaster/common.sh"

# Verificar dependências necessárias
check_dependency "argocd" "kubectl" "python3"

# Adicionar verificação de dependência do python3 e jinja2
check_dependency "python3"
pip3 show jinja2 >/dev/null 2>&1 || {
    log_error "Jinja2 não encontrado. Instalando..."
    pip3 install jinja2
}

# Função para criar projeto ArgoCD
create_project() {
    local project_name="$1"
    local description="${2:-}"
    
    log_info "Criando projeto ArgoCD: $project_name"
    
    if [ -z "$description" ]; then
        description="Projeto $project_name gerenciado via opsmaster"
    fi
    
    if argocd proj create "$project_name" \
        --description "$description" >/dev/null 2>&1; then
        log_info "✅ Projeto '$project_name' criado com sucesso"
    else
        log_error "Falha ao criar projeto '$project_name'"
        exit 1
    fi
}

# Função para criar aplicação ArgoCD
create_app() {
    local app_name="$1"
    local repo_url="$2"
    local path="$3"
    local dest_namespace="$4"
    local project="${5:-default}"
    local dest_server="${6:-https://kubernetes.default.svc}"
    
    log_info "Criando aplicação ArgoCD: $app_name"
    
    if argocd app create "$app_name" \
        --repo "$repo_url" \
        --path "$path" \
        --dest-server "$dest_server" \
        --dest-namespace "$dest_namespace" \
        --project "$project" \
        --sync-policy automated \
        --auto-prune \
        --self-heal >/dev/null 2>&1; then
        log_info "✅ Aplicação '$app_name' criada com sucesso"
    else
        log_error "Falha ao criar aplicação '$app_name'"
        exit 1
    fi
}

# Função para sincronizar aplicação
sync_app() {
    local app_name="$1"
    local prune="${2:-false}"
    
    log_info "Sincronizando aplicação: $app_name"
    
    local cmd="argocd app sync $app_name"
    [ "$prune" = "true" ] && cmd="$cmd --prune"
    
    if eval "$cmd" >/dev/null 2>&1; then
        log_info "✅ Aplicação '$app_name' sincronizada com sucesso"
    else
        log_error "Falha ao sincronizar aplicação '$app_name'"
        exit 1
    fi
}

# Função para observar status da aplicação
watch_app() {
    local app_name="$1"
    local timeout="${2:-300}"  # timeout padrão de 5 minutos
    
    log_info "Observando status da aplicação: $app_name"
    log_info "Timeout definido: ${timeout}s"
    
    if argocd app wait "$app_name" --timeout "$timeout"; then
        log_info "✅ Aplicação '$app_name' está saudável"
    else
        log_error "Timeout ou erro ao aguardar aplicação '$app_name'"
        exit 1
    fi
}

# Função para listar aplicações
list_apps() {
    local project="${1:-}"
    
    log_info "Listando aplicações ArgoCD"
    echo "----------------------------------------"
    echo "Nome                  Status    Projeto"
    echo "----------------------------------------"
    
    if [ -n "$project" ]; then
        argocd app list -p "$project" -o wide
    else
        argocd app list -o wide
    fi
}

# Função para adicionar repositório
add_repository() {
    local repo_url="$1"
    local repo_name="$2"
    local repo_type="${3:-git}"  # git por padrão
    local username="$4"
    local password="$5"
    
    log_info "Adicionando repositório: $repo_name ($repo_url)"
    
    local cmd="argocd repo add $repo_url --name $repo_name --type $repo_type"
    
    # Adicionar credenciais se fornecidas
    if [ -n "$username" ] && [ -n "$password" ]; then
        cmd="$cmd --username $username --password $password"
    fi
    
    if eval "$cmd" >/dev/null 2>&1; then
        log_info "✅ Repositório '$repo_name' adicionado com sucesso"
    else
        log_error "Falha ao adicionar repositório '$repo_name'"
        exit 1
    fi
}

# Função para processar template usando Jinja2
process_template() {
    local template="$1"
    local output="$2"
    shift 2
    local vars=("$@")
    
    if [ ! -f "$template" ]; then
        log_error "Template não encontrado: $template"
        exit 1
    }
    
    # Criar arquivo temporário para variáveis em formato JSON
    local temp_vars
    temp_vars=$(mktemp)
    trap 'rm -f "$temp_vars"' EXIT
    
    # Converter variáveis para JSON
    echo "{" > "$temp_vars"
    local first=true
    for var in "${vars[@]}"; do
        local key="${var%%=*}"
        local value="${var#*=}"
        
        # Detectar se o valor é um array ou objeto JSON
        if [[ "$value" =~ ^\[ || "$value" =~ ^\{ ]]; then
            # Valor já está em formato JSON
            if [ "$first" = true ]; then
                echo "  \"$key\": $value" >> "$temp_vars"
            else
                echo "  ,\"$key\": $value" >> "$temp_vars"
            fi
        else
            # Valor é uma string
            if [ "$first" = true ]; then
                echo "  \"$key\": \"$value\"" >> "$temp_vars"
            else
                echo "  ,\"$key\": \"$value\"" >> "$temp_vars"
            fi
        fi
        first=false
    done
    echo "}" >> "$temp_vars"
    
    # Processar template usando Python e Jinja2
    python3 -c "
import sys, json
from jinja2 import Template, Environment, StrictUndefined

# Configurar ambiente Jinja2
env = Environment(undefined=StrictUndefined)

# Ler template e variáveis
with open('$template', 'r') as f:
    template = env.from_string(f.read())
with open('$temp_vars', 'r') as f:
    vars = json.load(f)

# Renderizar template
try:
    result = template.render(**vars)
    with open('$output', 'w') as f:
        f.write(result)
except Exception as e:
    print(f'Erro ao processar template: {str(e)}', file=sys.stderr)
    sys.exit(1)
"
    
    if [ $? -eq 0 ]; then
        log_info "Template processado com sucesso: $output"
    else
        log_error "Falha ao processar template"
        exit 1
    fi
}

# Função para criar diretório temporário
create_temp_dir() {
    local temp_dir
    temp_dir=$(mktemp -d)
    echo "$temp_dir"
}

# Função para processar e aplicar manifesto
apply_manifest() {
    local name="$1"
    local action="$2"
    local template="$3"
    shift 3
    local vars=("$@")
    
    log_info "Criando $action a partir do template: $name"
    
    # Criar diretório temporário para o manifesto processado
    local temp_dir
    temp_dir=$(create_temp_dir)
    trap 'rm -rf "$temp_dir"' EXIT
    
    local manifest="$temp_dir/$name.yaml"
    
    # Processar template
    process_template "$template" "$manifest" "${vars[@]}"
    
    # Aplicar manifesto usando o comando apropriado
    if argocd "$action" create -f "$manifest" >/dev/null 2>&1; then
        log_info "$action '$name' criado com sucesso"
    else
        log_error "Falha ao criar $action '$name'"
        exit 1
    fi
}

# Função para criar aplicação a partir de template
create_app_from_template() {
    local app_name="$1"
    local template="$2"
    shift 2
    apply_manifest "$app_name" "app" "$template" "$@"
}

# Função para criar projeto a partir de template
create_project_from_template() {
    local project_name="$1"
    local template="$2"
    shift 2
    apply_manifest "$project_name" "proj" "$template" "$@"
}

# Função de ajuda
show_help() {
    echo "Gerenciamento de Aplicações ArgoCD"
    echo
    echo "Uso: opsmaster infra argocd <ação> [argumentos]"
    echo
    echo "Ações:"
    echo "  login <server> [token] [insecure]  Configura autenticação global ArgoCD"
    echo "  create-project <nome> <template> [VAR=valor]         Cria projeto a partir de template"
    echo "  create-app <nome> <repo> <path> <namespace> [proj]   Cria uma nova aplicação"
    echo "  sync <app> [prune]                                   Sincroniza uma aplicação"
    echo "  watch <app> [timeout]                                Observa status da aplicação"
    echo "  list [projeto]                                       Lista aplicações"
    echo "  add-repo <url> <nome> [tipo] [usuario] [senha]      Adiciona repositório"
    echo "  create-from-template <app> <template> [VAR=valor]   Cria app a partir de template"
    echo
    echo "Argumentos:"
    echo "  nome        Nome do projeto ou aplicação"
    echo "  descrição   Descrição do projeto (opcional)"
    echo "  repo        URL do repositório Git"
    echo "  path        Caminho para os manifestos no repositório"
    echo "  namespace   Namespace de destino no cluster"
    echo "  proj        Nome do projeto ArgoCD (opcional, default: default)"
    echo "  prune       Flag para remover recursos (true/false, default: false)"
    echo "  timeout     Tempo máximo de espera em segundos (default: 300)"
    echo
    echo "Argumentos para templates:"
    echo "  app         Nome da aplicação"
    echo "  template    Caminho para o arquivo de template YAML"
    echo "  VAR=valor   Variáveis para substituir no template (múltiplas)"
    echo
    echo "Argumentos para login:"
    echo "  server    URL do servidor ArgoCD (ex: argocd.exemplo.com)"
    echo "  token     Token de autenticação (opcional)"
    echo "  insecure  Ignorar validação SSL (true/false, default: false)"
    echo
    echo "Notas sobre autenticação:"
    echo "  - O token é armazenado em ~/.config/opsmaster/argocd-auth"
    echo "  - Se não fornecer token, será iniciado login interativo/SSO"
    echo "  - O token permanece válido até ser revogado no servidor"
    echo
    echo "Exemplos:"
    echo "  1. Login interativo:"
    echo "     opsmaster infra argocd login argocd.exemplo.com"
    echo
    echo "  2. Login com token:"
    echo "     opsmaster infra argocd login argocd.exemplo.com token123"
    echo
    echo "  3. Login inseguro (ignorar SSL):"
    echo "     opsmaster infra argocd login argocd.exemplo.com token123 true"
    echo
    echo "  4. Criar projeto:"
    echo "     opsmaster infra argocd create-project meu-projeto templates/project.yaml \\"
    echo "       NAME=meu-projeto DESCRIPTION='Meu projeto' \\"
    echo "       SOURCE_REPOS='git@github.com:org/*' \\"
    echo "       DESTINATION_NAMESPACES='prod,staging'"
    echo
    echo "  5. Criar aplicação:"
    echo "     opsmaster infra argocd create-app minha-app git@github.com:org/repo.git apps/manifests prod meu-projeto"
    echo
    echo "  6. Sincronizar aplicação:"
    echo "     opsmaster infra argocd sync minha-app true"
    echo
    echo "  7. Observar aplicação:"
    echo "     opsmaster infra argocd watch minha-app 600"
    echo
    echo "  8. Listar aplicações:"
    echo "     opsmaster infra argocd list"
    echo "     opsmaster infra argocd list meu-projeto"
    echo
    echo "  9. Adicionar repositório:"
    echo "     opsmaster infra argocd add-repo git@github.com:org/repo.git meu-repo"
    echo
    echo "  10. Criar app a partir de template:"
    echo "      opsmaster infra argocd create-from-template minha-app templates/app.yaml \\"
    echo "        NAME=minha-app NAMESPACE=prod REPO=git@github.com:org/repo.git PATH=apps/manifests"
    echo
}

# Nova função para gerenciar token do ArgoCD
setup_argocd_auth() {
    local config_dir="$HOME/.config/opsmaster"
    local auth_file="$config_dir/argocd-auth"
    
    # Criar diretório de configuração se não existir
    mkdir -p "$config_dir"
    
    # Verificar se já existe token válido
    if [ -f "$auth_file" ] && [ -n "$(cat "$auth_file")" ]; then
        export ARGOCD_AUTH_TOKEN="$(cat "$auth_file")"
        # Testar se o token ainda é válido
        if argocd account get-user-info >/dev/null 2>&1; then
            return 0
        fi
    fi
    
    log_error "Token ArgoCD não encontrado ou inválido"
    return 1
}

# Nova função para login com token
do_login() {
    local server="$1"
    local token="$2"
    local insecure="${3:-false}"
    local config_dir="$HOME/.config/opsmaster"
    local auth_file="$config_dir/argocd-auth"
    
    log_info "Configurando autenticação ArgoCD para: $server"
    
    # Criar diretório de configuração
    mkdir -p "$config_dir"
    
    # Se não foi fornecido token, tentar obter via SSO ou prompt
    if [ -z "$token" ]; then
        log_info "Token não fornecido, iniciando processo de login interativo..."
        local cmd="argocd login $server --sso"
        [ "$insecure" = "true" ] && cmd="$cmd --insecure"
        
        if ! eval "$cmd"; then
            log_error "Falha no processo de login"
            exit 1
        fi
        
        # Extrair token após login bem sucedido
        token=$(argocd account generate-token)
    fi
    
    # Salvar token
    echo "$token" > "$auth_file"
    chmod 600 "$auth_file"
    
    # Exportar token para uso imediato
    export ARGOCD_AUTH_TOKEN="$token"
    
    log_info "Autenticação configurada com sucesso"
    argocd account get-user-info
}

# Função principal
main() {
    local action="$1"
    shift
    
    case "$action" in
        login)
            if [ -z "$1" ]; then
                log_error "Servidor ArgoCD não especificado"
                show_help
                exit 1
            fi
            do_login "$@"
            ;;
        create-project|create-app|sync|watch|list|add-repo|create-from-template)
            # Tentar configurar autenticação
            setup_argocd_auth || {
                log_error "Configure a autenticação primeiro:"
                log_error "opsmaster infra argocd login <server> [token] [insecure]"
                exit 1
            }
            
            case "$action" in
                create-project)
                    if [ -z "$2" ]; then
                        log_error "Nome do projeto e template são obrigatórios"
                        show_help
                        exit 1
                    fi
                    local project_name="$1"
                    local template="$2"
                    shift 2
                    create_project_from_template "$project_name" "$template" "$@"
                    ;;
                create-app)
                    if [ -z "$4" ]; then
                        log_error "Argumentos insuficientes para criar aplicação"
                        show_help
                        exit 1
                    fi
                    create_app "$1" "$2" "$3" "$4" "$5"
                    ;;
                sync)
                    if [ -z "$1" ]; then
                        log_error "Nome da aplicação não especificado"
                        show_help
                        exit 1
                    fi
                    sync_app "$1" "$2"
                    ;;
                watch)
                    if [ -z "$1" ]; then
                        log_error "Nome da aplicação não especificado"
                        show_help
                        exit 1
                    fi
                    watch_app "$1" "$2"
                    ;;
                list)
                    list_apps "$1"
                    ;;
                add-repo)
                    if [ -z "$2" ]; then
                        log_error "URL e nome do repositório são obrigatórios"
                        show_help
                        exit 1
                    fi
                    add_repository "$1" "$2" "$3" "$4" "$5"
                    ;;
                create-from-template)
                    if [ -z "$2" ]; then
                        log_error "Nome da aplicação e template são obrigatórios"
                        show_help
                        exit 1
                    fi
                    local app_name="$1"
                    local template="$2"
                    shift 2
                    create_app_from_template "$app_name" "$template" "$@"
                    ;;
            esac
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