# Comando get

Este documento detalha o uso dos subcomandos do grupo opsmaster get.

## opsmaster get ip

Busca e exibe informações detalhadas sobre seus endereços de IP, tanto da sua rede local quanto o seu IP público na internet.

## Uso

```bash
opsmaster get ip [flags]
```

Flags

- `--local, -l:` Exibe apenas as informações da rede local.

- `--public, -p:` Exibe apenas as informações do seu IP público.

Se nenhuma flag for fornecida, o comando exibirá ambas as informações.

Detalhes da Saída

- Informações Locais: Exibe uma tabela com os detalhes da sua principal interface de rede, incluindo:

  - `INTERFACE:` O nome da sua placa de rede (ex: eth0, en0).

  - `IP:` O seu endereço de IP privado na rede.

  - `MÁSCARA:` A máscara de sub-rede.

  - `GATEWAY:` O gateway padrão da sua rede.

  - `MAC ADDRESS:` O endereço físico da sua placa de rede.

- Informações Públicas: Utiliza o serviço ipinfo.io para buscar detalhes sobre o seu IP na internet, incluindo:

  - `IP:` O seu endereço de IP público.

  - `CIDADE, REGIÃO, PAÍS:` A localização geográfica estimada do seu IP.

  - `ORGANIZAÇÃO:` O provedor de internet (ISP) associado ao seu IP.

Exemplos

```bash
# Exibe todas as informações de IP (público e local)
opsmaster get ip

# Exibe apenas os detalhes da rede local
opsmaster get ip --local

# Exibe apenas os detalhes do IP público
opsmaster get ip -p
```

## opsmaster get dns <domínio>

Realiza uma consulta DNS para um domínio, similar a ferramentas como dig ou nslookup.

## Uso

```bash
opsmaster get dns [domínio] [flags]
```

Flags

- `--type, -t string:` O tipo de registro DNS a ser consultado (ex: A, AAAA, MX, TXT, CNAME, NS). (padrão: "A")

Exemplos

```bash
# Busca os registros A (endereços IPv4) do google.com
opsmaster get dns google.com

# Busca os registros MX (servidores de e-mail) do gmail.com
opsmaster get dns gmail.com -t MX

# Busca os registros TXT (registros de texto) do github.com
opsmaster get dns github.com --type TXT
```
