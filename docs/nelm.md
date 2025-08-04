# Comando Nelm 🚀

O comando `nelm` é uma funcionalidade "mágica" do OpsMaster que automatiza o fluxo completo de gerenciamento de releases Helm em ambientes Kubernetes. Ele combina validação, planejamento e execução em um único comando inteligente.

## 🎯 Conceito "Mágico"

Quando você executa `opsmaster nelm install`, o comando automaticamente:

1. **🔍 Valida** o chart com `nelm chart lint`
2. **📋 Planeja** a instalação com `nelm release plan install` (mostra o que será instalado/alterado)
3. **🚀 Executa** a instalação com `nelm release install` (após confirmação)

## 📋 Comandos Disponíveis

### `install` - Instalar Releases

Instala uma ou todas as releases detectadas no repositório.

```bash
# Instalar todas as releases detectadas
opsmaster nelm install --env stg --kube-context kubedev

# Instalar uma release específica
opsmaster nelm install -r sample-api --env stg --kube-context kubedev

# Com namespace customizado
opsmaster nelm install -r sample-api --env stg --kube-context kubedev -n default

# Com auto-approve (pula confirmação)
opsmaster nelm install -r sample-api --env stg --kube-context kubedev --auto-approve

# Com timeout e concorrência personalizados
opsmaster nelm install -r sample-api --env stg --kube-context kubedev --timeout 10m --max-concurrency 5
```

#### Flags Disponíveis

| Flag | Tipo | Obrigatório | Descrição |
|------|------|-------------|-----------|
| `--env` | string | ✅ | Ambiente de destino (ex: stg, prd) |
| `--kube-context` | string | ✅ | Contexto do kubeconfig a ser usado |
| `--release` | string | ❌ | Nome da release específica (se vazio, processa todas) |
| `--namespace` | string | ❌ | Namespace onde a release será instalada |
| `--auto-approve` | bool | ❌ | Pula a confirmação interativa |
| `--timeout` | string | ❌ | Timeout para a operação (ex: 10s, 1m, 1h) |
| `--max-concurrency` | int | ❌ | Máximo de releases executadas em paralelo |

### `status` - Verificar Status

Verifica o status de releases instaladas.

```bash
# Listar todas as releases
opsmaster nelm status --kube-context kubedev

# Verificar uma release específica
opsmaster nelm status --kube-context kubedev --release sample-api

# Com namespace específico
opsmaster nelm status --kube-context kubedev --namespace sample-api-stg
```

### `uninstall` - Desinstalar Releases

Remove releases do cluster.

```bash
# Desinstalar uma release
opsmaster nelm uninstall --release sample-api --kube-context kubedev

# Com auto-approve
opsmaster nelm uninstall --release sample-api --kube-context kubedev --auto-approve
```

### `rollback` - Fazer Rollback

Reverte uma release para uma versão anterior.

```bash
# Rollback para a revisão anterior
opsmaster nelm rollback --release sample-api --kube-context kubedev

# Rollback para uma revisão específica
opsmaster nelm rollback --release sample-api --kube-context kubedev --revision 2
```

## 🔧 Funcionalidades Inteligentes

### Detecção Automática de Releases

O comando automaticamente detecta releases no repositório:

```
📁 Seu Repositório/
├── 📁 sample-api/
│   ├── Chart.yaml          ← Release detectada
│   ├── values-stg.yaml     ← Valores específicos do ambiente
│   └── values-prd.yaml
├── 📁 another-service/
│   ├── Chart.yaml          ← Release detectada
│   └── values.yaml
└── 📁 docs/
    └── README.md           ← Ignorado (não tem Chart.yaml)
```

### Gerenciamento Inteligente de Valores

O comando detecta automaticamente arquivos de valores específicos por ambiente:

- Se existir `values-{env}.yaml` → usa esse arquivo
- Se não existir → usa `--set environment={env}`

### Execução Paralela

Quando instalando múltiplas releases, o comando executa em paralelo com controle de concorrência:

```bash
⚡ Executando releases em paralelo total=3 maxConcurrency=3
🚀 Iniciando processamento da release sample-api
🚀 Iniciando processamento da release another-service
🚀 Iniciando processamento da release third-service
```

### Validações Prévia

Antes de executar qualquer comando, o nelm valida:

- ✅ Existência do `Chart.yaml`
- ✅ Validade do contexto do Kubernetes
- ✅ Informações do chart (nome, versão, descrição)
- ✅ Dependências do chart

## 📊 Exemplo de Saída

```bash
$ opsmaster nelm install -r sample-api --env stg --kube-context kubedev

🔍 Iniciando validações prévias release=sample-api
📦 Chart: sample-api v1.0.0
📝 Descrição: Uma API web simples em Go
🔍 Verificando dependências do chart path=./sample-api/Chart.lock
✅ Dependências do chart verificadas
✅ Validações prévias concluídas release=sample-api

🔧 Executando 'nelm chart lint' release=sample-api
✅ Validação do chart concluída com sucesso! release=sample-api

📋 Executando 'nelm release plan install'... release=sample-api
📄 Usando arquivo de valores específico do ambiente file=values-stg.yaml release=sample-api

🚀 Resumo da instalação para 'sample-api':
   📁 Diretório: ./sample-api
   🌍 Ambiente: stg
   🎯 Namespace: sample-api
   🔧 Contexto K8s: kubedev
   📄 Valores: values-stg.yaml

Deseja aplicar estas alterações? (s/N): s

🚀 Executando 'nelm release install'... release=sample-api
✅ Instalação concluída com sucesso! release=sample-api
```

## 🛠️ Pré-requisitos

Para usar o comando `nelm`, você precisa ter:

1. **NelM** instalado e configurado
2. **kubectl** configurado com contextos válidos
3. **Helm charts** válidos no repositório
4. **Acesso** ao cluster Kubernetes

## 🧪 Testando o Comando

Use o script de teste incluído:

```bash
# Tornar executável (se necessário)
chmod +x scripts/test-nelm-commands.sh

# Executar teste completo
./scripts/test-nelm-commands.sh
```

## 🔍 Troubleshooting

### Erro: "contexto do Kubernetes inválido"
```bash
# Verificar contextos disponíveis
kubectl config get-contexts

# Usar um contexto válido
opsmaster nelm install --env stg --kube-context CONTEXTO_VALIDO
```

### Erro: "Chart.yaml não encontrado"
```bash
# Verificar se está no diretório correto
ls -la Chart.yaml

# Navegar para o diretório do chart
cd /caminho/para/seu/chart
opsmaster nelm install --env stg --kube-context kubedev
```

### Erro: "falha na validação do chart"
```bash
# Verificar sintaxe do Chart.yaml
nelm chart lint

# Corrigir problemas no chart antes de continuar
```

## 💡 Dicas de Uso

1. **Use `--auto-approve` em CI/CD** para automatizar deploys
2. **Configure timeouts adequados** para releases grandes
3. **Use `--max-concurrency`** para controlar carga no cluster
4. **Monitore logs** para identificar problemas rapidamente
5. **Teste em staging** antes de aplicar em produção

## 🔄 Fluxo de Trabalho Recomendado

1. **Desenvolvimento**: Use `--env dev` para testes locais
2. **Staging**: Use `--env stg` para validação
3. **Produção**: Use `--env prd` com `--auto-approve` em CI/CD

O comando `nelm` torna o gerenciamento de releases Helm muito mais simples e seguro, automatizando as tarefas repetitivas e fornecendo visibilidade completa do processo de deploy.
