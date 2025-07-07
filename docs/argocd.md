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
