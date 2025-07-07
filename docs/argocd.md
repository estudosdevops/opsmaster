# Comandos argocd

Este documento detalha o uso dos subcomandos do grupo opsmaster argocd.

🔧 Configuração Inicial

Para usar os comandos do Argo CD, o OpsMaster precisa saber como se conectar ao seu servidor. A forma recomendada é criar um arquivo de configuração em `~/.opsmaster.yaml.`

Exemplo:

```yaml
# Define qual contexto usar por padrão
current-context: staging

# Lista de todos os contextos disponíveis
contexts:
  staging:
    argocd:
      server: "argocd.seu-dominio.com"
      token: "SEU_TOKEN_DE_STAGING_AQUI"
      insecure: true # Use 'true' apenas para ambientes de homelab/teste
  
  producao:
    argocd:
      server: "argo.empresa.com"
      token: "SEU_TOKEN_DE_PRODUCAO_AQUI"
      insecure: false
 ```

Você também pode passar as flags `--server,` `--token` e `--insecure` diretamente na linha de comando para sobrescrever o arquivo de configuração.

🚀 Exemplos de Uso
A seguir, exemplos de como usar os comandos mais comuns.

## Explorando Comandos

Para mais informações, exemplos e todas as flags disponíveis para um comando específico, use a flag `--help`

```bash
# Exemplo: Ver todas as opções para o comando 'app list'
opsmaster argocd app list --help

# Exemplo: Ver todas as opções para o comando 'project create'
opsmaster argocd project create --help
```

## Listando, Criando e Removendo Recursos

```bash
# Adiciona um novo repositório Git
opsmaster argocd repo add https://github.com/sua-empresa/meu-servico.git

# Cria um novo projeto
opsmaster argocd project create meu-projeto-staging --description "Projeto para Staging"

# Exibe detalhes de uma aplicação para depuração
opsmaster argocd app get meu-servico-stg

# Força a sincronização de uma aplicação
opsmaster argocd app sync meu-servico-stg

# Apaga uma aplicação
opsmaster argocd app delete meu-servico-stg

# Apaga um projeto
opsmaster argocd project delete meu-projeto-staging

# Apaga um repositório
opsmaster argocd repo delete https://github.com/sua-empresa/meu-servico.git
```

Para o fluxo completo de deploy de uma nova aplicação com o comando app create, consulte a documentação de referência abaixo.

📖 Referência de Comandos

## Comandos `repo`

- opsmaster argocd repo add <url-do-repositorio>

  Registra um novo repositório Git no Argo CD. Use as flags --username e --password para repositórios privados.

- opsmaster argocd repo list

  Exibe uma tabela com todos os repositórios Git registrados no Argo CD.

- opsmaster argocd repo delete <url-do-repositorio>

  Remove o registro de um repositório do Argo CD.

## Comandos `project`

- opsmaster argocd project create <nome-do-projeto>

  Cria um novo AppProject no Argo CD. Use --description para adicionar uma descrição e --source-repo para permitir repositórios de origem.

- opsmaster argocd project list [nome-do-projeto]

  Exibe uma tabela com todos os projetos ou os detalhes de um projeto específico.

- opsmaster argocd project delete <nome-do-projeto>

  Apaga um projeto do Argo CD. Apenas funciona se não houver aplicações associadas a ele.

## Comandos `app`

- opsmaster argocd app create

  Cria ou atualiza uma aplicação. Este comando possui várias flags para especificar os detalhes do deploy. Use ... app create --help para ver todas as opções.

- opsmaster argocd app get <nome-da-aplicacao>

  Exibe informações detalhadas de uma aplicação, incluindo uma tabela com o status de todos os seus recursos sincronizados. Ideal para depuração

- opsmaster argocd app list [nome-da-aplicacao]

  Exibe uma tabela com todas as aplicações ou os detalhes de uma aplicação específica.

- opsmaster argocd app sync <nome-da-aplicacao>
  
  Inicia uma sincronização imediata para uma aplicação. Use a flag --force para substituir recursos e apagar os que não existem mais no Git

- opsmaster argocd app wait <nome-da-aplicacao>

  Pausa a execução e aguarda até que uma aplicação atinja o estado Healthy e Synced. Muito útil para pipelines.

- opsmaster argocd app delete <nome-da-aplicacao>

  Apaga uma aplicação do Argo CD.