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
