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

🚀 Fluxo de Deploy Completo (Exemplo)

Este é o fluxo completo para implantar um novo serviço usando o OpsMaster.

1. Adicionar o Repositório do Serviço
Primeiro, se o repositório do seu serviço for privado, você precisa registrá-lo no Argo CD.

```bash
opsmaster argocd repo add https://github.com/sua-empresa/meu-servico.git \
    --username seu-user \
    --password $GIT_TOKEN
```

2. Criar o Projeto
Em seguida, crie um projeto no Argo CD para agrupar suas aplicações. Este projeto deve ter permissão para usar o repositório que você adicionou.

```bash
opsmaster argocd project create meu-projeto-staging \
    --description "Projeto para o ambiente de Staging" \
    --source-repo "https://github.com/sua-empresa/meu-servico.git"
```

3. Criar a Aplicação (O Deploy)
Com o repositório e o projeto prontos, você pode criar a Application. Este comando aponta para o repositório do seu serviço e usa um values.yaml específico para o ambiente.

```bash
opsmaster argocd app create \
    --app-name "meu-servico-stg" \
    --project "meu-projeto-staging" \
    --dest-namespace "staging" \
    --repo-url "https://github.com/sua-empresa/meu-servico.git" \
    --repo-path "chart" \
    --values "values-stg.yaml" \
    --set-image-repo "meu-registro/meu-servico" \
    --set-image-tag "v1.2.3" \
    --set-chart-dependency "generic-app"
```

## Nota sobre Ambientes

A flag `--target-revision` (que por padrão é HEAD) é usada para apontar para diferentes versões do seu código. É uma prática comum usar uma branch para ambientes de desenvolvimento/staging (ex: `--target-revision "develop"`) e uma tag Git para ambientes de produção (ex: `--target-revision "v1.2.3"`).

4. Aguardar o Deploy Ficar Pronto
Use o comando wait para pausar a sua pipeline até que a aplicação esteja saudável e sincronizada.

```bash
opsmaster argocd app wait meu-servico-stg
```

5. Listar e Confirmar o Status
Finalmente, use o comando list para obter um relatório final do status da sua aplicação.

```bash
opsmaster argocd app list meu-servico-stg
```

6. Limpeza Completa
Após os testes, você pode usar os comandos `delete` para limpar completamente o ambiente.

```bash
# Apaga a aplicação
opsmaster argocd app delete meu-servico-stg

# Apaga o projeto (após a aplicação ser removida)
opsmaster argocd project delete meu-projeto-staging

# Apaga o registro do repositório
opsmaster argocd repo delete https://github.com/sua-empresa/meu-servico.git
```