# Orders

Учебный проект — система приёма и обработки заказов.

## Описание

Репозиторий содержит два независимых Go-проекта (каждый со своим `go.mod`):

| Сервис | Директория | Модуль | Назначение |
|--------|-----------|--------|-----------|
| **order-service** | `order-service/` | `github.com/dimaglobin/order-service` | HTTP API для приёма заказов: регистрирует заказ, сохраняет в PostgreSQL и передаёт событие в Kafka через паттерн Transactional Outbox |
| **notifier** | `notifier/` | `github.com/dimaglobin/notifier` | Kafka-консьюмер: читает события о заказах и отправляет уведомления пользователям |

## Архитектура

Каждый сервис построен по принципам чистой архитектуры:

- **model** — доменные структуры, без зависимостей
- **interfaces** — интерфейсы определены там, где используются (Go best practice)
- **service** — бизнес-логика, зависит только от интерфейсов
- **handler/consumer** — транспортный слой (HTTP для order-service, Kafka для notifier)

Сервисы связаны через событие `OrderCreated`, которое определено в каждом проекте независимо (одинаковые поля, разные пакеты).

## Стек технологий

- **Go 1.26** — язык разработки
- **PostgreSQL** — основная реляционная БД (order-service)
- **Kafka** — брокер сообщений между сервисами
- **Transactional Outbox** — гарантированная доставка событий без двойной записи
- **log/slog** — структурированное логирование (stdlib)
- **cleanenv** — загрузка конфигурации из YAML + переменных окружения

## Запуск

```bash
# Orders API — с конфигурацией по умолчанию
cd order-service && go run ./cmd/

# Orders API — с yaml-файлом
cd order-service && go run ./cmd/ -config config/config.yml

# Notifier — с конфигурацией по умолчанию
cd notifier && go run ./cmd/

# Notifier — с yaml-файлом
cd notifier && go run ./cmd/ -config config/config.yml
```

Для остановки нажмите `Ctrl+C` (SIGINT) или пошлите SIGTERM процессу.

## Конфигурация

Значения задаются в YAML-файле (флаг `-config`) и/или через переменные окружения. Переменные окружения имеют приоритет.

### Orders API

| Переменная | Описание | По умолчанию |
|------------|----------|--------------|
| `HTTP_HOST` | Адрес HTTP-сервера | `localhost` |
| `HTTP_PORT` | Порт HTTP-сервера (1-65535) | `8080` |
| `DB_HOST` | Хост PostgreSQL | `localhost` |
| `DB_PORT` | Порт PostgreSQL (1-65535) | `5432` |
| `DB_USER` | Пользователь БД | `postgres` |
| `DB_PASSWORD` | Пароль БД | `postgres` |
| `DB_NAME` | Имя базы данных | `orders` |
| `DB_SSLMODE` | Режим SSL | `disable` |
| `KAFKA_BROKERS` | Адреса брокеров Kafka (через запятую) | `localhost:9092` |
| `KAFKA_TOPIC` | Топик Kafka | `orders` |
| `LOG_LEVEL` | Уровень логирования: debug/info/warn/error | `info` |
| `LOG_FORMAT` | Формат логов: json/text | `json` |

### Notifier

| Переменная | Описание | По умолчанию |
|------------|----------|--------------|
| `KAFKA_BROKERS` | Адреса брокеров Kafka (через запятую) | `localhost:9092` |
| `KAFKA_TOPIC` | Топик Kafka | `orders` |
| `KAFKA_GROUP_ID` | Consumer group ID | `notifier` |
| `LOG_LEVEL` | Уровень логирования: debug/info/warn/error | `info` |
| `LOG_FORMAT` | Формат логов: json/text | `json` |

## Структура проекта

```
.
├── order-service/               # Сервис приёма заказов (HTTP API)
│   ├── cmd/main.go           # Точка входа
│   ├── config/config.yml     # Пример конфигурации
│   └── internal/
│       ├── apperrors/        # Типы ошибок приложения
│       ├── config/           # Загрузка и валидация конфигурации
│       ├── orders/           # Доменная логика заказов
│       └── outbox/           # Transactional Outbox
├── notifier/                 # Сервис уведомлений (Kafka-консьюмер)
│   ├── cmd/main.go           # Точка входа
│   ├── config/config.yml     # Пример конфигурации
│   └── internal/
│       ├── apperrors/        # Типы ошибок приложения
│       ├── config/           # Загрузка и валидация конфигурации
│       └── notifier/         # Доменная логика уведомлений
└── README.md
```

## Тесты

```bash
# Orders API
cd order-service && go test ./...

# Notifier
cd notifier && go test ./...
```
