# Testes do OpsMaster

Este diretório contém os testes automatizados para os scripts do OpsMaster.

## Estrutura de Diretórios

```
tests/
├── README.md                 # Este arquivo
├── common/                   # Testes comuns e utilitários
│   ├── setup.sh             # Script de setup para os testes
│   └── utils.sh             # Funções utilitárias para testes
├── backup/                   # Testes dos scripts de backup
│   ├── postgresql/          # Testes do script postgresql.sh
│   │   ├── test_dump.sh     # Testes da função dump
│   │   ├── test_restore.sh  # Testes da função restore
│   │   └── test_list.sh     # Testes da função list
│   └── mongodb/             # Testes do script mongodb.sh
│       ├── test_dump.sh
│       ├── test_restore.sh
│       └── test_list.sh
└── docker/                   # Testes dos scripts Docker
    ├── test_build.sh
    └── test_run.sh
```

## Como Executar os Testes

Para executar todos os testes:
```bash
./run_tests.sh
```

Para executar testes específicos:
```bash
# Testes do PostgreSQL
./run_tests.sh backup/postgresql

# Testes do MongoDB
./run_tests.sh backup/mongodb

# Testes do Docker
./run_tests.sh docker
```

## Convenções

1. Todos os arquivos de teste devem:
   - Começar com `test_`
   - Terminar com `.sh`
   - Ser executáveis (`chmod +x`)

2. Cada arquivo de teste deve:
   - Importar `common/utils.sh`
   - Usar as funções de teste do BATS
   - Ter uma descrição clara do que está testando
   - Limpar seu próprio ambiente após os testes

3. Nomenclatura dos testes:
   - `test_<script>_<function>.sh`
   - Exemplo: `test_postgresql_dump.sh`

## Exemplo de Teste

```bash
#!/usr/bin/env bats

load "common/utils"

@test "dump postgresql deve criar arquivo de backup" {
    # Setup
    local backup_file="/tmp/test_backup.dump"
    
    # Execução
    run opsmaster backup postgresql dump --backup-file "$backup_file"
    
    # Verificação
    [ "$status" -eq 0 ]
    [ -f "$backup_file" ]
    
    # Limpeza
    rm -f "$backup_file"
} 