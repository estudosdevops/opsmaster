# Ambientes de Teste para Bancos de Dados

Este diretório contém ambientes Docker Compose para testar os scripts de backup e restore de diferentes bancos de dados.

## Estrutura

```
databases/
├── postgresql/
│   ├── docker-compose.yml    # Ambiente PostgreSQL
│   └── init-scripts/         # Scripts de inicialização
│       └── 01-init-data.sql  # Dados de teste
└── mongodb/
    └── docker-compose.yml    # Ambiente MongoDB
```

## PostgreSQL

### Configuração
- **Origem**: localhost:5432
  - Banco: sourcedb
  - Usuário: postgres
  - Senha: sourcepass123
  - Superusuário: postgres
  - Senha do superusuário: sourceadmin123
- **Destino**: localhost:5433
  - Banco: targetdb
  - Usuário: postgres
  - Senha: targetpass123
  - Superusuário: postgres
  - Senha do superusuário: targetadmin123

### Estrutura de Dados
O banco de origem (`sourcedb`) é inicializado com as seguintes tabelas:
- `users`: Usuários do sistema
- `products`: Catálogo de produtos
- `orders`: Pedidos dos usuários
- `order_items`: Itens de cada pedido

### Uso
```bash
# Iniciar ambiente
cd databases/postgresql
docker-compose up -d

# Criar configuração do script
opsmaster backup postgresql init-config

# Editar configuração (~/.config/backups/postgresql.yaml)
source:
  host: localhost
  port: 5432
  database: sourcedb
  username: postgres
  password: sourcepass123

target:
  host: localhost
  port: 5433
  database: targetdb
  username: postgres
  password: targetpass123

# Verificar dados no banco de origem (usando superusuário)
docker exec -it postgres_source psql -U postgres -d sourcedb -c "SELECT COUNT(*) FROM users;"
docker exec -it postgres_source psql -U postgres -d sourcedb -c "SELECT COUNT(*) FROM products;"
docker exec -it postgres_source psql -U postgres -d sourcedb -c "SELECT COUNT(*) FROM orders;"

# Testar script
opsmaster backup postgresql sync

# Verificar dados no banco de destino (usando superusuário)
docker exec -it postgres_target psql -U postgres -d targetdb -c "SELECT COUNT(*) FROM users;"
docker exec -it postgres_target psql -U postgres -d targetdb -c "SELECT COUNT(*) FROM products;"
docker exec -it postgres_target psql -U postgres -d targetdb -c "SELECT COUNT(*) FROM orders;"
```

## MongoDB

### Configuração
- **Origem**: localhost:27017
- **Destino**: localhost:27018
- **Banco**: mydb
- **Usuário**: admin
- **Senha**: admin123

### Uso
```bash
# Iniciar ambiente
cd databases/mongodb
docker-compose up -d

# Criar configuração do script
opsmaster backup mongodb init-config

# Editar configuração (~/.config/backups/mongodb.yaml)
source:
  host: localhost
  port: 27017
  database: mydb
  username: admin
  password: admin123

target:
  host: localhost
  port: 27018
  database: mydb
  username: admin
  password: admin123

# Criar dados de teste
docker exec -it mongodb_source mongosh -u admin -p admin123 --eval 'db.test.insertMany([{name: "test1"}, {name: "test2"}])'

# Testar script
opsmaster backup mongodb sync
```

## Limpeza

Para parar e remover os containers:
```bash
# PostgreSQL
cd databases/postgresql
docker-compose down

# MongoDB
cd databases/mongodb
docker-compose down
```

Para remover também os volumes (apaga todos os dados):
```bash
# PostgreSQL
cd databases/postgresql
docker-compose down -v

# MongoDB
cd databases/mongodb
docker-compose down -v
``` 