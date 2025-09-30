# Order Service - Микросервис для обработки заказов | Ready To Check

[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

Высоконагруженный микросервис для обработки заказов с использованием Kafka, PostgreSQL и LRU кэшем. Разработан как решение тестового задания L0 для WB Техношколы.

## 🏗️ Архитектура

```
┌─────────────┐     ┌─────────────┐    ┌─────────────┐
│   Kafka     │     │ PostgreSQL  │    │   LRU       │
│  Producer   │───> │   Orders    │    │   Cache     │
└─────────────┘     └─────────────┘    └─────────────┘
       │                   ▲                   ▲
       │                   │                   │
       ▼                   │                   │
┌─────────────┐            │                   │
│   Kafka     │            │                   │
│  Consumer   │────────────┼───────────────────┘
└─────────────┘            │
       │                   │
       ▼                   │
┌─────────────┐            │
│   HTTP      │            │
│   API       │────────────┘
└─────────────┘
```

## ✨ Ключевые особенности

- **Микросервисная архитектура** - Четкое разделение на слои (транспорт, бизнес-логика, репозиторий)
- **Kafka Integration** - Получение заказов из топика Kafka
- **PostgreSQL** - Хранение данных о заказах в реляционной БД  
- **In-Memory Cache** - LRU-кэш с TTL для ускорения доступа к данным
- **Dead Letter Queue (DLQ)** - Обработка некорректных сообщений с возможностью повторной обработки
- **Graceful Shutdown** - Корректное завершение работы сервиса
- **Structured Logging** - Структурированное логирование с использованием `zap`
- **Metrics** - Экспорт метрик в формате Prometheus
- **Swagger Documentation** - Автоматически сгенерированная документация API
- **Testing** - Unit и интеграционные тесты

## 🚀 Технологии

- **Go 1.25**
- **Gin** (HTTP framework)
- **pgx/v5** (PostgreSQL driver)  
- **segmentio/kafka-go** (Kafka client)
- **zap** (Logger)
- **prometheus/client_golang** (Metrics)
- **testify** (Testing)
- **Docker & Docker Compose**
- **PostgreSQL 17**
- **Kafka** (Confluent Platform 7.9.2)

## 🚀 Быстрый старт

### Предварительные требования

- Docker & Docker Compose
- Go 1.25+ (для локальной разработки)
- Make (опционально)

### Запуск проекта

1. **Клонирование репозитория**
```bash
git clone <repository-url>
cd order-service
```

2. **Запуск всех сервисов**
```bash
make compose-up-all
# Или напрямую через docker-compose
# docker-compose up --build -d
```

3. **Проверка работоспособности**
```bash
# Health check
curl http://localhost:8080/health

# Получение заказа
curl http://localhost:8080/orders/{order_uid}
```

### Использование

1. Откройте веб-интерфейс: `http://localhost:8080`
2. Отправьте тестовое сообщение в Kafka:
```bash
# Локальный запуск (требует .env)
make run-producer

# Запуск через docker compose
docker run kafka-producer
```
3. Введите UID заказа в поле поиска веб-интерфейса
4. Проверьте API напрямую:
```bash
curl http://localhost:8080/orders/b563feb7b2b84b6test
```

## 📊 Мониторинг и документация

- **API**: http://localhost:8080
- **Swagger UI**: http://localhost:8080/swagger/index.html
- **Prometheus**: http://localhost:9090
- **Grafana**: http://localhost:3000 (admin/grafana)
- **Метрики приложения**: http://localhost:8081/metrics

## 🔧 Конфигурация

### Переменные окружения

Полный пример конфигурации см. в `.env.example`

## 🏗️ Структура проекта

```
├── cmd/                    # Точки входа
│   ├── order-service/      # Основной сервис
│   └── producer-service/   # Эмулятор Kafka producer
├── configs/               # Конфигурации
├── docs/                  # Swagger документация (автогенерируется)
├── internal/              # Внутренняя логика
│   ├── app/              # Инициализация приложения
│   ├── config/           # Конфигурация
│   ├── entity/           # Бизнес-сущности
│   ├── repository/       # Слой данных
│   ├── service/          # Бизнес-логика
│   └── transport/        # HTTP/Kafka транспорты
│       ├── http/         # HTTP handlers, middleware
│       └── kafka/        # Kafka consumer, DLQ
├── migrations/           # Миграции БД
├── pkg/                  # Переиспользуемые пакеты
│   ├── cache/           # LRU кэш
│   ├── kafka/           # Kafka utilities
│   ├── logger/          # Структурированное логирование
│   ├── metric/          # Prometheus метрики
│   └── storage/         # PostgreSQL клиент
│       └── transaction/ # Менеджер работы с транзакциями
├── tests/               # Тесты
│   ├── integration/     # Интеграционные тесты
├── web/                # Веб-интерфейс
└── volumes/            # Docker volumes
```

## 🧪 Разработка и тестирование

### Локальный запуск

```bash
# Запуск зависимостей
docker-compose --env-file .env up -d db kafka zookeeper

# Применение миграций
make migrate-up

# Запуск сервиса (требует .env)
make run
```

### Тестирование

```bash
# Unit тесты
make test

# Интеграционные тесты
make compose-up-integration-test

# Линтинг | Golangci linter
make linter-golangci

# Линтинг | Hadolint
make linter-hadolint

# Линтинг | DotEnv linter
make linter-dotenv

# Проверка зависимостей
make deps-audit
```

### Swagger документация

Генерация документации:
```bash
make swag-v1
```

## 📈 Производительность и метрики

### Оптимизации

1. **Кэширование**: LRU кэш с TTL для быстрого доступа к заказам
2. **Connection Pooling**: Оптимизированный пул соединений с БД
3. **Batch Processing**: Пакетная обработка сообщений Kafka
4. **Graceful Shutdown**: Корректное завершение с сохранением данных

### Доступные метрики

- HTTP запросы/ответы (количество, длительность, медленные запросы)
- Kafka сообщения (обработанные, ошибки, lag)
- Кэш (hit/miss, eviction, размер)  
- Транзакции БД (успехи, ошибки, retry)

## 📝 API Документация

### GET /health
Health check endpoint

**Response:**
```json
{
  "status": "ok"
}
```

### GET /orders/{order_uid}
Получение заказа по ID

**Response:**
```json
{
  "order_uid": "550e8400-e29b-41d4-a716-446655440000",
  "track_number": "WBILMTESTTRACK",
  "entry": "WBIL",
  "delivery": {
    "name": "Test Testov",
    "phone": "+9720000000",
    "zip": "2639809",
    "city": "Kiryat Mozkin",
    "address": "Ploshad Mira 15",
    "region": "Kraiot",
    "email": "test@gmail.com"
  },
  "payment": {
    "transaction": "b563feb7b2b84b6test",
    "currency": "USD",
    "provider": "wbpay",
    "amount": 1817,
    "payment_dt": 1637907727,
    "bank": "alpha",
    "delivery_cost": 1500,
    "goods_total": 317
  },
  "items": [
    {
      "chrt_id": 9934930,
      "track_number": "WBILMTESTTRACK",
      "price": 453,
      "name": "Mascaras",
      "sale": 30,
      "size": "0",
      "total_price": 317,
      "nm_id": 2389212,
      "brand": "Vivienne Sabo",
      "status": 202
    }
  ],
  "locale": "en",
  "customer_id": "test",
  "delivery_service": "meest",
  "date_created": "2021-11-26T06:22:19Z"
}
```

Полная документация API доступна в Swagger UI: http://localhost:8080/swagger/index.html

## 🚀 Развертывание

### Docker
```bash
docker build -t order-service .
docker run -p 8080:8080 order-service
```

## 🔒 Безопасность

- Валидация всех входных данных
- Структурированное логирование всех операций
- Graceful error handling
- Безопасная конфигурация подключений

## 📄 Лицензия

MIT 0 License - см. файл [LICENSE](LICENSE) для деталей.
