# OpsMaster 🚀

OpsMaster é uma ferramenta de linha de comando (CLI) moderna, construída em Go, projetada para simplificar e automatizar tarefas rotineiras de DevOps e SRE.

🤔 Por que o OpsMaster?

No dia a dia de um profissional de DevOps/SRE, executamos dezenas de comandos repetitivos: verificar o status de um serviço, checar uma porta, resolver um DNS, criar uma aplicação no Argo CD. O OpsMaster nasceu para ser um "canivete suíço" para operações, centralizando essas automações em uma única CLI rápida, consistente e fácil de usar.

Este projeto foi iniciado como uma forma prática de estudar a linguagem Go, aplicando seus conceitos na criação de uma ferramenta relevante para o dia a dia de um profissional de DevOps/SRE.

✨ Funcionalidades Principais

O OpsMaster é organizado em grupos de comandos lógicos segue abaixo alguns exemplos:

- `scan:` Realiza verificações ativas em alvos de rede (ports, monitor).

- `get:` Busca e exibe informações de rede e de sistema (ip, dns).

- `argocd:` Interage com a API do Argo CD para automatizar o ciclo de vida de aplicações (app, project, repo).

⚙️ Instalação

Para instalar o OpsMaster, você precisa ter o Go configurado na sua máquina.

```bash
go install github.com/estudosdevops/opsmaster@latest
```

📚 Documentação dos Comandos:

A documentação detalhada, com todas as flags e exemplos de uso para cada comando, [pode ser encontrada na pasta docs](./docs)

🚀 Exemplo Rápido

### Escaneia as portas 22, 80 e 443 em um host

opsmaster scan ports scanme.nmap.org --ports 22,80,443

### Busca os registros MX (servidores de e-mail) do google.com

opsmaster get dns google.com --type MX

Para exemplos de uso avançado, como a configuração e o deploy de aplicações com o ArgoCD, por favor, consulte a documentação dos comandos argocd.

🤝 Contribuição

Sinta-se à vontade para abrir issues ou pull requests. Toda contribuição é bem-vinda!

📄 Licença
Este projeto é distribuído sob a licença MIT.
