# Comando `install`

Instala e configura ferramentas em múltiplas instâncias na nuvem em paralelo.

## Pré-requisitos

- AWS CLI configurado com credenciais válidas
- Instâncias EC2 com SSM Agent instalado e funcionando
- Arquivo CSV com lista de instâncias (formato: `instance_id,account,region`)

## Uso Básico

```bash
# Ver ajuda do comando install
opsmaster install -h

# Ver ajuda do subcomando puppet
opsmaster install puppet -h

# Instalação básica
opsmaster install puppet \
  --instances-file instances.csv \
  --puppet-server puppet.example.com

# Dry-run (validar sem executar)
opsmaster install puppet \
  --instances-file instances.csv \
  --puppet-server puppet.example.com \
  --dry-run
```

## Configuração de Retry

O opsmaster possui sistema de retry com backoff exponencial para lidar com falhas temporárias de rede e API. Você pode configurar o comportamento de retry com as seguintes flags:

### Flags de Retry Disponíveis

| Flag | Tipo | Padrão | Descrição |
|------|------|--------|-----------|
| `--max-retries` | int | 3 | Número máximo de tentativas para todas as operações |
| `--retry-delay` | duration | 2s | Delay base entre tentativas |
| `--retry-jitter` | bool | true | Adiciona variação aleatória aos delays (evita thundering herd) |
| `--ssm-retries` | int | 0 | Tentativas específicas para operações SSM (0 = usa `--max-retries`) |
| `--ec2-retries` | int | 0 | Tentativas específicas para operações EC2 (0 = usa `--max-retries`) |

### Exemplos de Configuração de Retry

```bash
# Retry conservador para produção
opsmaster install puppet \
  --instances-file instances.csv \
  --puppet-server puppet.example.com \
  --max-retries 2 \
  --retry-delay 5s

# Retry agressivo para ambientes instáveis
opsmaster install puppet \
  --instances-file instances.csv \
  --puppet-server puppet.example.com \
  --max-retries 10 \
  --retry-delay 1s

# Fine-tuning por tipo de operação
opsmaster install puppet \
  --instances-file instances.csv \
  --puppet-server puppet.example.com \
  --ssm-retries 5 \
  --ec2-retries 10 \
  --retry-delay 2s

# Sem jitter (timing previsível para debug)
opsmaster install puppet \
  --instances-file instances.csv \
  --puppet-server puppet.example.com \
  --retry-jitter=false

# Falhar rápido para CI/CD
opsmaster install puppet \
  --instances-file instances.csv \
  --puppet-server puppet.example.com \
  --max-retries 1 \
  --retry-delay 100ms
```

### Como o Retry Funciona

1. **Backoff Exponencial**: O delay entre tentativas aumenta exponencialmente (1s, 2s, 4s, 8s...)
2. **Jitter**: Adiciona variação aleatória (±25%) para evitar que múltiplas instâncias façam retry simultaneamente
3. **Classificação de Erros**: Só faz retry para erros temporários (timeout, throttling). Erros permanentes (permissão negada, instância não existe) falham imediatamente
4. **Políticas Específicas**:
   - **SSM**: Delays maiores (comandos Puppet podem demorar)
   - **EC2**: Delays menores (APIs EC2 são mais rápidas)

### Cenários de Uso

| Cenário | Configuração Recomendada | Exemplo |
|---------|------------------------|---------|
| **Produção** | Conservador, delays maiores | `--max-retries 2 --retry-delay 5s` |
| **Desenvolvimento** | Agressivo para debug | `--max-retries 5 --retry-delay 1s` |
| **CI/CD** | Falhar rápido | `--max-retries 1 --retry-delay 100ms` |
| **Rede instável** | Mais tentativas | `--max-retries 10 --retry-delay 2s` |
| **Debug timing** | Sem jitter | `--retry-jitter=false` |
