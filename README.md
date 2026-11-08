hexlet-go-sql
CLI‑приложение на Go для управления пользователями (CRUD) с хранением в SQLite.
Все результаты выводятся в stdout в формате JSON, ошибки — в stderr, код выхода: 0 при успехе, 1 при ошибке.

Структура проекта
text
.
├── cmd
│   └── app
│       └── main.go          # Точка входа CLI
├── data.db                  # Файл базы данных (создаётся автоматически)
├── go.mod
├── go.sum
└── internal
    └── storage
        ├── models.go        # Структуры сущностей и DTO
        └── repo.go          # Репозиторий: CRUD-методы
Требования
Go 1.27+
ОС: Linux / macOS / Windows
Как собрать и запустить
Сборка бинарного файла (в корне проекта):

bash
go build -o app ./cmd/app
На Windows получится app.exe.

Запуск CLI:

bash
./app --action=create --email=alice@example.com --name=Alice --age=30
Для Windows: .\app.exe ...

Доступные действия (--action)
create — создать пользователя
Флаги:

--email (обязательно)
--name (опционально)
--age (опционально)
Пример:

bash
./app --action=create --email=alice@example.com --name=Alice --age=30
Успешный вывод (stdout, JSON):

json
{
  "id": 1,
  "email": "alice@example.com",
  "name": "Alice",
  "age": 30,
  "created_at": "2024-12-10T12:34:56Z"
}
Если email уже занят (нарушение UNIQUE), в stderr будет сообщение об ошибке, код выхода — 1.

get — получить пользователя по ID
Флаги:

--id (обязательно)
Пример:

bash
./app --action=get --id=1
Вывод:

json
{
  "id": 1,
  "email": "alice@example.com",
  "name": "Alice",
  "age": 30,
  "created_at": "2024-12-10T12:34:56Z"
}
При отсутствии пользователя — ошибка в stderr, код 1.

list — список всех пользователей
Флаги: нет

Пример:

bash
./app --action=list
Вывод (массив JSON):

json
[
  {
    "id": 1,
    "email": "alice@example.com",
    "name": "Alice",
    "age": 30,
    "created_at": "2024-12-10T12:34:56Z"
  },
  {
    "id": 2,
    "email": "bob@example.com",
    "name": null,
    "age": null,
    "created_at": "2024-12-10T13:00:00Z"
  }
]
Поля name и age выводятся как null, если они не заданы.

update — частичное обновление пользователя
Обновляются только переданные поля.

Флаги:

--id (обязательно)
--name (опционально)
--age (опционально)
Примеры:

bash
# Обновить только возраст
./app --action=update --id=1 --age=31

# Обновить имя и возраст
./app --action=update --id=1 --name="Alice Updated" --age=32
Вывод: актуальное состояние пользователя после обновления (в том же JSON‑формате).

delete — удалить пользователя
Флаги:

--id (обязательно)
Пример:

bash
./app --action=delete --id=1
Вывод: JSON‑объект удалённого пользователя (как будто «получили перед удалением»).
Если ID не найден — ошибка в stderr, код 1.

Формат JSON
Структура сущности User:

json
{
  "id": 1,                  // int64
  "email": "string",         // string, NOT NULL, UNIQUE
  "name": "string | null",  // string или null
  "age": "integer | null",  // integer или null
  "created_at": "string"    // RFC3339 timestamp
}
Данные выводятся в stdout.
Ошибки — в stderr.
created_at формируется автоматически в SQLite и возвращается в формате RFC3339.
Автосоздание таблиц
При первом запуске любой команды CLI автоматически создаются таблицы (если их нет):

users: id, email, name, age, created_at.
courses: id, slug, title, price.
Это происходит через CREATE TABLE IF NOT EXISTS внутри initDB.

Чтобы начать с чистой базы, удалите файл:

bash
rm data.db
После следующего запуска таблицы будут созданы заново.

Примеры использования с обработкой JSON
Получить email и имя через jq:

bash
./app --action=get --id=1 | jq '.email, .name'
Создать и сразу вывести список:

bash
./app --action=create --email=test@example.com --name=Test && ./app --action=list
Посмотреть справку по флагам:

bash
./app -h

Транзакции
Все операции, требующие атомарности (массовые загрузки, переводы, согласованные изменения в нескольких таблицах), выполняются внутри транзакций через обёртку storage.withTx.

Как это работает
BeginTx начинает транзакцию.
defer tx.Rollback() гарантирует откат при любой ошибке, даже если она возникнет в середине цикла.
Если все операции успешны, вызывается tx.Commit(). После коммита повторный Rollback безопасно игнорируется.
Все запросы (ExecContext, QueryContext) внутри транзакции выполняются через объект *sql.Tx, а не через *sql.DB.
Что происходит при ошибке
Транзакция немедленно откатывается: изменения не сохраняются в БД.
CLI выводит JSON с "status": "failed" и понятное сообщение об ошибке.
В БД не остаётся частично применённых изменений (например, при импорте не будет «половины» пользователей).