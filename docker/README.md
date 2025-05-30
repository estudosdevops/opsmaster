# Ambientes de Teste Docker

Este diretório contém ambientes Docker Compose para testar diferentes funcionalidades do projeto OpsMaster.

## Estrutura

```
docker/
├── databases/           # Ambientes de teste para bancos de dados
│   ├── postgresql/     # Testes com PostgreSQL
│   └── mongodb/        # Testes com MongoDB
└── README.md           # Este arquivo
```

## Categorias de Teste

### Bancos de Dados
Ambientes para testar scripts de backup e restore de diferentes bancos de dados.
Veja [README específico](databases/README.md) para mais detalhes.

### Outros Ambientes
Outros ambientes de teste serão adicionados aqui conforme necessário.

## Uso

Cada subdiretório contém seu próprio ambiente Docker Compose e documentação específica.
Consulte o README do ambiente específico para instruções detalhadas de uso. 