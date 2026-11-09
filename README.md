# go-sql

CLI-утилита для работы с SQLite (users, courses) на Go. Базовые CRUD-операции
генерируются sqlc, сложные сценарии (bulk upsert в транзакции, повторное
использование подготовленного выражения) остаются ручными.

## Зависимости

- Go 1.27.3
- `modernc.org/sqlite` — драйвер SQLite без CGO
- `sqlc v1.31.1` — генератор Go-кода из SQL.

sqlc объявлен директивой `tool` в go.mod, поэтому запускается через `go tool sqlc`
без глобальной установки. Версия зафиксирована в go.mod: директива
`tool github.com/sqlc-dev/sqlc/cmd/sqlc` + запись в `require`.

Установить/обновить версию:
    go get -tool github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1

## Генерация кода

    make sqlc
    # эквивалент: go tool sqlc generate

Схема читается из каталога migrations, запросы — из query, результат
пишется в internal/db. Сгенерированные файлы не редактируются руками:
после изменения миграций или запросов запусти генерацию заново и закоммить
результат.

## Структура

- migrations/ — DDL (схема таблиц), источник истины для sqlc и для initDB
- query/ — SQL-запросы с директивами -- name: ... :one / :many / :exec
- internal/db/ — сгенерированный sqlc пакет (models, Queries, Querier)
- internal/storage/repo.go — доменные модели, конвертация sql.Null*,
  ручные сценарии (bulk, prepared stmt)

## CLI

    go run ./cmd/app --action=create --email=a@example.com --name=Alice --age=30
    go run ./cmd/app --action=get --id=1
    go run ./cmd/app --action=list
    go run ./cmd/app --action=update --id=1 --age=31
    go run ./cmd/app --action=delete --id=1
    go run ./cmd/app --action=bulk
    go run ./cmd/app --action=import

## Сборка

    make build
