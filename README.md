[![Review Assignment Due Date](https://classroom.github.com/assets/deadline-readme-button-22041afd0340ce965d47ae6ef1cefeee28c7c493a6346c4f15d667ab976d596c.svg)](https://classroom.github.com/a/xR-tWBKa)

# Room Booking Service

Сервис бронирования переговорок.

## Запуск

```bash
docker compose up --build -d
```

Сервис доступен на `http://localhost:8080`. Все переменные окружения имеют дефолты, `.env` не требуется.

```bash
curl http://localhost:8080/_info
```

## Остановка

```bash
docker compose down -v
```

## Стек

- **Go** — chi (роутер), zerolog (логирование)
- **PostgreSQL 16** — хранение данных
- **JWT** (golang-jwt) — аутентификация
- **bcrypt** — хеширование паролей (для /register, /login)
- **goose** — миграции БД
- **Docker Compose** — оркестрация

## Архитектура

```
cmd/api/main.go          — точка входа, DI
internal/
  config/                — конфигурация из ENV
  db/                    — подключение к PostgreSQL, миграции (goose)
  domain/                — доменные модели и ошибки
  handler/               — HTTP-хендлеры, роутер, helpers
  jwt/                   — генерация и парсинг JWT
  middleware/             — auth middleware (Bearer token → context)
  repository/            — слой работы с БД (pgxpool)
  service/               — бизнес-логика
migrations/              — SQL-миграции (goose)
```

Слоистая архитектура: **handler → service → repository**. Хендлеры отвечают за HTTP (парсинг запроса, проверка роли, формирование ответа). Сервисы — за бизнес-логику (валидация, генерация слотов). Репозитории — за SQL-запросы.

## Ключевые решения

**Генерация слотов.** При создании расписания слоты генерируются сразу на 30 дней вперёд и сохраняются в БД. UUID слотов стабильные (генерируются PostgreSQL при INSERT). Это позволяет делать JOIN со слотами при проверке бронирований и не пересчитывать слоты на каждый запрос.

**Доступные слоты.** Запрос `GET /slots/list` использует LEFT JOIN с bookings и фильтрует по `b.id IS NULL` — возвращает только незанятые слоты. Индекс `(room_id, start_at)` оптимизирует запрос.

**Отмена бронирования.** Идемпотентная операция — повторный вызов на уже отменённой брони возвращает 200 с `status: cancelled`. Отмена меняет статус на `cancelled`, после чего слот снова появляется в списке доступных.

**Аутентификация.** JWT с claims `user_id` и `role`. `/dummyLogin` выдаёт токены с фиксированными UUID. Бонусные `/register` и `/login` используют bcrypt для хеширования паролей.

## Тестирование

### Unit-тесты

```bash
go test ./...
```

Покрытие >40%. Тестируются: валидация в хендлерах и сервисах, генерация слотов, парсинг JWT, auth middleware, маппинг ошибок.

### E2E-тесты

Требуют запущенного сервера:

```bash
docker compose up --build -d
go test -tags=e2e -v ./...
docker compose down -v
```

Два сценария:
1. Создание комнаты → расписание → получение слотов → бронирование
2. Создание брони → отмена → проверка идемпотентности → слот снова доступен