# 🛠️ OPSMaster CLI

## 📖 Sobre o Projeto

OPSMaster é uma CLI (Command Line Interface) desenvolvida em Shell Script que simplifica operações comuns de DevOps e administração de sistemas.

## 🎯 Por que OPSMaster?

- 🔄 Automatiza tarefas operacionais repetitivas
- 🎨 Interface consistente para diferentes operações
- 📦 Fácil de estender com novos comandos
- 🚀 Agiliza o trabalho de DevOps e SREs


## 🚀 Instalação

### Requisitos do Sistema
- Sistema Operacional Linux (Ubuntu, Debian, CentOS, etc.)
- Bash 4.0 ou superior
- Git instalado
- Permissões sudo

### Passo a Passo

1. Clone o repositório:

```
git clone https://github.com/estudosdevops/opsmaster.git
```

2. Acesse o diretório do projeto:

```
cd opsmaster
```

3. Dê permissão de execução ao script de instalação:

```
chmod +x install.sh
```

4. Execute o script de instalação:

```
sudo ./install.sh
```

O script de instalação irá:
- Copiar o executável para `/usr/local/bin/`
- Configurar as permissões necessárias
- Instalar dependências básicas (se necessário)
- Criar diretórios de configuração

5. Verifique se a instalação foi bem-sucedida:

```
opsmaster --version
```

### Desinstalação

Para remover o OPSMaster, execute:

```
sudo ./uninstall.sh
```

## 💻 Comandos Disponíveis
opsmaster [comando] [subcomando] [flags]

Exemplos de comandos:
- `opsmaster aws`: Gerencia recursos AWS
- `opsmaster k8s`: Operações relacionadas ao Kubernetes
- `opsmaster infra`: Gerenciamento de infraestrutura
- `opsmaster monitor`: Monitoramento de recursos
- `opsmaster backup`: Realiza operações de backup e restauração

Para ver todos os comandos disponíveis, execute:
opsmaster --help

## 🔍 Funções Comuns Disponíveis

A biblioteca `common.sh` fornece as seguintes funções:

- `log_info "mensagem"`: ✅ Log com destaque verde
- `log_warn "mensagem"`: ⚠️ Log com destaque amarelo
- `log_error "mensagem"`: ❌ Log com destaque vermelho
- `check_dependency "comando"`: Verifica se uma dependência está instalada
- `check_env_var "VARIAVEL"`: Verifica se uma variável de ambiente está definida

## 🤝 Contribuindo

1. Faça um fork do projeto
2. Crie sua branch de feature (`git checkout -b feature/AmazingFeature`)
3. Commit suas mudanças (`git commit -m 'Add some AmazingFeature'`)
4. Push para a branch (`git push origin feature/AmazingFeature`)
5. Abra um Pull Request

## 📝 Licença

Este projeto está sob a licença MIT. Veja o arquivo [LICENSE](LICENSE) para mais detalhes.

## 🐛 Reportando Problemas

Encontrou um bug? Por favor, abra uma issue descrevendo:
- O que aconteceu
- O que deveria acontecer
- Passos para reproduzir
- Ambiente (SO, versões das dependências)
