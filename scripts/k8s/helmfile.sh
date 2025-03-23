#!/usr/bin/env bash

source "$(dirname "$(dirname "$(dirname "$0")")")/lib/common.sh"

# Verificar dependências necessárias
check_dependency "helmfile" "kubectx" "kubectl"

# Função para listar todas as releases disponíveis
list_releases() {
    log_info "Releases disponíveis:"
    ls -1 releases/ 2>/dev/null || log_error "Diretório 'releases' não encontrado"
}

# Função para executar comandos helmfile
execute_helmfile() {
    local release="$1"
    local action="$2"
    local environment="$3"

    if [ ! -d "releases/$release" ]; then
        log_error "Release '$release' não encontrada no diretório releases/"
        exit 1
    }

    log_info "Executando helmfile para $release (ambiente: $environment)"
    
    cd "releases/$release" || exit 1
    
    # Configurar o contexto Kubernetes
    log_info "Configurando contexto Kubernetes para $environment"
    kubectx "kube$environment" || {
        log_error "Falha ao mudar contexto para kube$environment"
        exit 1
    }

    # Executar comando helmfile
    log_info "Executando: helmfile -e $environment $action"
    helmfile -e "$environment" "$action"
    
    if [ $? -eq 0 ]; then
        log_info "Comando executado com sucesso! ✨"
    else
        log_error "Falha na execução do comando"
    fi
}

# Função de ajuda
show_help() {
    echo "Gerenciamento de releases Helmfile"
    echo
    echo "Uso: opsmaster infra helmfile <ação> <release> [ambiente]"
    echo
    echo "Ações:"
    echo "  list                    Lista todas as releases disponíveis"
    echo "  diff <release> [env]    Mostra as diferenças para uma release (default: dev)"
    echo "  apply <release> [env]   Aplica uma release (default: dev)"
    echo "  sync <release> [env]    Sincroniza uma release (default: dev)"
    echo "  destroy <release> [env] Remove uma release (default: dev)"
    echo
    echo "Exemplos:"
    echo "  opsmaster infra helmfile list"
    echo "  opsmaster infra helmfile diff my-app"
    echo "  opsmaster infra helmfile apply my-app prod"
    echo
}

# Função principal
main() {
    local action="$1"
    local release="$2"
    local environment="${3:-dev}"  # default para dev se não especificado

    case "$action" in
        list)
            list_releases
            ;;
        diff|apply|sync|destroy)
            if [ -z "$release" ]; then
                log_error "Release não especificada"
                show_help
                exit 1
            fi
            execute_helmfile "$release" "$action" "$environment"
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