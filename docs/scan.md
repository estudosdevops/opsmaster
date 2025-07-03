# Comando scan

Este documento detalha o uso dos subcomandos do grupo opsmaster scan.

## opsmaster scan ports <host>

Verifica o status de portas TCP (abertas, fechadas ou filtradas) em um determinado host.

## Uso

```bash
opsmaster scan ports [host] [flags]
```

Flags:

- `--ports, -p string:` Portas a escanear. Pode ser uma lista separada por vírgulas ou um intervalo. (padrão: "1-1024")

- `--timeout, -t duration:` Timeout para cada tentativa de conexão de porta. (padrão: 2s)

Exemplos

```bash
# Escaneia as portas 22, 80 e 443 em um host
opsmaster scan ports scanme.nmap.org -p 22,80,443

# Escaneia as primeiras 100 portas de um endereço IP
opsmaster scan ports 1.1.1.1 --ports 1-100

# Aumenta o timeout para 3 segundos para redes mais lentas
opsmaster scan ports scanme.nmap.org -p 80 -t 3s
```

## opsmaster scan monitor <url>

Monitora uma URL em intervalos regulares para verificar sua disponibilidade.

## Uso

```bash
opsmaster scan monitor [url] [flags]
```

Flags

- `--interval, -i duration:` Intervalo de tempo entre as verificações. (padrão: 10s)

- `--count, -c int:` Número de verificações a serem feitas. Use 0 para monitorar indefinidamente (até ser interrompido com CTRL+C). (padrão: 0)

Exemplos

```bash
# Monitora o google.com a cada 30 segundos
opsmaster scan monitor https://google.com -i 30s

# Verifica a URL 5 vezes, com um intervalo de 5 segundos entre cada verificação
opsmaster scan monitor https://meu-servico.com -c 5 -i 5s
```
