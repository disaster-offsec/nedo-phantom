```
nedo-phantom/
├── cmd/
│   ├── agent/          # точка входа для агента (Linux)
│   │   └── main.go
│   └── c2/             # точка входа для C2-сервера
│       └── main.go
├── internal/
│   ├── agent/
│   │   ├── core/       # ядро: подключение, цикл задач
│   │   ├── modules/    # exec, file, persistence, info, pivot
│   │   └── crypto/     # шифрование
│   ├── c2/
│   │   ├── api/        # обработчики HTTP
│   │   ├── db/         # работа с очередью задач
│   │   └── web/        # статика для UI (опционально)
│   └── common/         # общие структуры и утилиты
├── configs/            # примеры конфигов (agent.yaml, c2.yaml)
├── build/              # скрипты сборки (Makefile)
└── go.mod
```
