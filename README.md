# Order Service - Микросервис для обработки заказов

[![Go Report Card](https://goreportcard.com/badge/github.com/your-username/order-service)](https://goreportcard.com/report/github.com/your-username/order-service) 
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

Высоконагруженный микросервис для обработки заказов с использованием Kafka, PostgreSQL и кэширования. Разработан как решение тестового задания для Wildberries.

## 🏗️ Архитектура

```
┌─────────────┐     ┌─────────────┐    ┌─────────────┐
│   Kafka     │     │ PostgreSQL  │    │   LRU       │
│  Producer   │───▶ │   Orders    │    │   Cache     │
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
- **Comprehensive Testing** - Unit, интеграционные и E2E тесты

## 🚀 Технологии

- **Go 1.25**
- **Gin** (HTTP framework)
- **pgx/v5** (PostgreSQL driver)  
- **segmentio/kafka-go** (Kafka client)
- **zap** (Logger)
- **prometheus/client_golang** (Metrics)
- **testify** (Testing)
- **Docker & Docker Compose**
- **PostgreSQL 16**
- **Kafka** (Confluent Platform 7.6.1)

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
curl http://localhost:8080/api/v1/orders/{order_uid}
```

### Использование

1. Откройте веб-интерфейс: `http://localhost:8080`
2. Отправьте тестовое сообщение в Kafka:
```bash
make run-producer
```
3. Введите UID заказа в поле поиска веб-интерфейса
4. Проверьте API напрямую:
```bash
curl http://localhost:8080/api/v1/orders/b563feb7b2b84b6test
```

## 📊 Мониторинг и документация

- **API**: http://localhost:8080
- **Swagger UI**: http://localhost:8080/swagger/index.html
- **Prometheus**: http://localhost:9090
- **Grafana**: http://localhost:3000 (admin/grafana)
- **Метрики приложения**: http://localhost:8081/metrics

## 🔧 Конфигурация

### Переменные окружения

Основные настройки в `.env`:

```env
# Приложение
APP_NAME=OrderService
APP_VERSION=1.0.0
APP_PORT=:8080

# База данных
DB_HOST=localhost
DB_PORT=5432
DB_NAME=orders
DB_USER=postgres
DB_PASSWORD=postgres123

# Кэш
CACHE_CAPACITY=1000
CACHE_TTL=1h

# Kafka
KAFKA_GROUP_ID=order-service-group
KAFKA_BROKERS=kafka1:39092,kafka2:39093,kafka3:39094
KAFKA_TOPIC=orders
```

Полный пример конфигурации см. в `.env.example`

## 🏗️ Структура проекта

```
├── cmd/                    # Точки входа
│   ├── order-service/      # Основной сервис
│   └── producer-service/   # Эмулятор Kafka producer
├── configs/               # Конфигурации
├── docs/                  # Swagger документация (генерируется)
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
├── tests/               # Тесты
│   ├── integration/     # Интеграционные тесты
│   └── e2e/            # End-to-end тесты
├── web/                # Веб-интерфейс
└── volumes/            # Docker volumes
```

## 🧪 Разработка и тестирование

### Локальный запуск

```bash
# Запуск зависимостей
docker-compose up -d db kafka zookeeper

# Применение миграций
make migrate-up

# Запуск сервиса
make run
```

### Тестирование

```bash
# Unit тесты
make test

# Интеграционные тесты
make compose-up-integration-test

# Линтинг
make linter-golangci

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

### GET /api/v1/orders/{order_uid}
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

### Kubernetes
```bash
kubectl apply -f k8s/
```

## 🔒 Безопасность

- Валидация всех входных данных
- Структурированное логирование всех операций
- Graceful error handling
- Безопасная конфигурация подключений

## 🤝 Вклад в проект

1. Fork репозитория
2. Создайте feature branch (`git checkout -b feature/amazing-feature`)
3. Commit изменения (`git commit -m 'Add amazing feature'`)
4. Push в branch (`git push origin feature/amazing-feature`)
5. Откройте Pull Request

## 📄 Лицензия

MIT License - см. файл [LICENSE](LICENSE) для деталей.

---

**Контакты**
- Email: your-email@example.com
- GitHub: [@your-username](https://github.com/your-username)

