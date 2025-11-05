# DelayedNotifier

## Описание

Сервис отложенных уведомлений: планирование и отправка уведомлений (Email, Telegram) через RabbitMQ с кэшированием в Redis и хранением в PostgreSQL. Включает REST API и простую веб-страницу для просмотра.

## Состав репозитория

- **cmd/main.go** — точка входа через FX DI.
- **internal/**
  - **app/** — модели данных Notification, NotificationRequest.
  - **broker/** — продюссер RabbitMQ.
  - **config/** — загрузка конфигурации из YAML.
  - **consumer/** — консьюмер RabbitMQ.
  - **db/** — работа с PostgreSQL (CRUD, кэш-загрузка).
  - **redis/** — реализация кэша через Redis.
  - **sender/** — реализация отправки уведомлений (Telegram, Email).
  - **web/** — HTTP-обработчики и роутер.
- **config/local.yaml** — пример конфигурации.
- **migrations/** — SQL-миграции для PostgreSQL.
- **docs/** — Swagger-документация.
- **web/index.html** — простая страница для отправки, получения, удаления уведомлений.
- **docker-compose.yml** — запуск PostgreSQL, Redis, RAbbitMQ через Docker.
- **.env.example** - пример env файла для кредов.


---

## Быстрый старт

### 1. Запуск инфраструктуры

```sh
docker-compose up -d
```
(Запустит контейнеры: postgres → порт 5433, RabbitMQ → 5672/15672, Redis → 6379.)

### 2. Настроить переменные окружения и конфигурацию
(пример в .env.example + config/local.yaml)

### 3. Применить миграции (migrate):

```sh
migrate -path migrations -database "postgres://user:password@localhost:5433/dbname?sslmode=disable" up
```

### 4. Запуск сервиса

```sh
go run ./cmd/main.go
```

Сервис стартует на порту 8080.



## API

- **POST /notify** — создать уведомление (JSON: channel, recipient, message, send_at);
- **GET /notify/{id}** — получение статуса уведомления;
- **DELETE /notify/{id}** —  отмена запланированного уведомления;
- **Swagger**: [http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html)

---

## Веб-интерфейс
Откройте index.html в браузере — простая страница для просмотра уведомлений/отправки тестов через API.


## Тесты
Юнит-тесты: `go test ./internal/...`

## Миграции

- `migrations/000001_create_tables.up.sql` — создание таблиц.
- `migrations/000001_create_tables.down.sql` — удаление таблиц.

---

## Логирование и метрики
Логирование реализовано через wbf/zlog (используется в internal/*).

## Зависимости

- Go 1.25+
- PostgreSQL 16+
- RabbitMQ 3.13+
- Redis 7+
- Docker (для локального запуска инфраструктуры)

---

## Swagger

- Swagger: [docs/swagger.yaml](docs/swagger.yaml)
- Документация генерируется автоматически и доступна по `/swagger/*`.