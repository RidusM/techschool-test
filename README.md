# Order Service - Микросервис для обработки заказов

Высоконагруженный микросервис для обработки заказов с использованием Kafka, PostgreSQL и кэширования.

## 🏗️ Архитектура

```
┌─────────────┐     ┌─────────────┐    ┌─────────────┐
│   Kafka     │     │ PostgreSQL  │    │   LRU       │
│  Producer   │───▶│   Orders    │    │   Cache     │
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

## 🚀 Быстрый старт

### Предварительные требования

- Docker & Docker Compose
- Go 1.25+
- Make

### Запуск проекта

1. **Клонирование репозитория**
```bash
git clone <repository-url>
cd l0
```

2. **Запуск всех сервисов**
```bash
make compose-up-all
```

3. **Проверка работоспособности**
```bash
# Health check
curl http://localhost:8080/health

# Получение заказа
curl http://localhost:8080/api/v1/orders/{order_uid}
```

## 📊 Мониторинг

- **Prometheus**: http://localhost:9090
- **Grafana**: http://localhost:3000
- **Метрики приложения**: http://localhost:8080/metrics

## 🔧 Конфигурация

### Переменные окружения

Основные настройки в `configs/dev.env`:

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

## 🏗️ Структура проекта

```
├── cmd/                    # Точки входа
│   └── main.go
├── configs/               # Конфигурации
│   ├── dev.env
│   ├── prod.env
│   └── test.env
├── internal/              # Внутренняя логика
│   ├── app/              # Инициализация приложения
│   ├── config/           # Конфигурация
│   ├── entity/           # Бизнес-сущности
│   ├── repository/       # Слой данных
│   ├── service/          # Бизнес-логика
│   └── transport/        # HTTP/Kafka транспорты
├── migrations/           # Миграции БД
├── pkg/                  # Переиспользуемые пакеты
│   ├── cache/           # Кэширование
│   ├── kafka/           # Kafka клиенты
│   ├── logger/          # Логирование
│   ├── metric/          # Метрики
│   └── storage/         # Хранилище данных
└── volumes/             # Docker volumes
```

## 🧪 Тестирование

### Unit тесты
```bash
make test
```

### Интеграционные тесты
```bash
make integration-test
```

### Линтинг
```bash
make linter-golangci
```

## 📈 Производительность

### Оптимизации

1. **Кэширование**: LRU кэш с TTL для быстрого доступа к заказам
2. **Connection Pooling**: Оптимизированный пул соединений с БД
3. **Batch Processing**: Пакетная обработка сообщений Kafka
4. **Graceful Shutdown**: Корректное завершение с сохранением данных

### Метрики

- HTTP запросы/ответы
- Время отклика БД
- Hit/miss ratio кэша
- Lag Kafka consumer
- Количество ошибок

## 🔒 Безопасность

- Валидация всех входных данных
- Безопасные пароли (мин. 8 символов)
- Логирование всех операций
- Graceful error handling

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

## 📝 API Документация

### GET /health
Health check endpoint

**Response:**
```json
{
  "status": "ok",
  "timestamp": "2024-01-01T00:00:00Z",
  "service": "order-service"
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
    "request_id": "",
    "currency": "USD",
    "provider": "wbpay",
    "amount": 1817,
    "payment_dt": 1637907727,
    "bank": "alpha",
    "delivery_cost": 1500,
    "goods_total": 317,
    "custom_fee": 0
  },
  "items": [
    {
      "chrt_id": 9934930,
      "track_number": "WBILMTESTTRACK",
      "price": 453,
      "rid": "ab4219087a764ae0btest",
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
  "internal_signature": "",
  "customer_id": "test",
  "delivery_service": "meest",
  "shardkey": "9",
  "sm_id": 99,
  "date_created": "2021-11-26T06:22:19Z",
  "oof_shard": "1"
}
```

## 🤝 Contributing

1. Fork репозитория
2. Создайте feature branch (`git checkout -b feature/amazing-feature`)
3. Commit изменения (`git commit -m 'Add amazing feature'`)
4. Push в branch (`git push origin feature/amazing-feature`)
5. Откройте Pull Request

## 📄 Лицензия

MIT License - см. файл [LICENSE](LICENSE) для деталей.











# Order Service (Тестовое задание WB)

[![Go Report Card](https://goreportcard.com/badge/github.com/your-username/order-service)](https://goreportcard.com/report/github.com/your-username/order-service) 
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

## Описание

Демонстрационный микросервис на Go, реализующий систему управления заказами. Сервис получает данные заказов из Kafka, сохраняет их в PostgreSQL, кэширует в памяти и предоставляет HTTP API для их получения. Также реализован простой веб-интерфейс для поиска заказов.

Этот проект был разработан как решение тестового задания для Wildberries.

## Ключевые особенности

*   **Микросервисная архитектура:** Четкое разделение на слои (транспорт, бизнес-логика, репозиторий).
*   **Kafka Integration:** Получение заказов из топика Kafka.
*   **PostgreSQL:** Хранение данных о заказах в реляционной БД.
*   **In-Memory Cache:** LRU-кэш с TTL для ускорения доступа к часто запрашиваемым данным.
*   **Dead Letter Queue (DLQ):** Обработка некорректных сообщений Kafka с возможностью повторной обработки.
*   **Graceful Shutdown:** Корректное завершение работы сервиса.
*   **Structured Logging:** Структурированное логирование с использованием `zap`.
*   **Metrics:** Экспорт метрик в формате Prometheus.
*   **Configuration:** Гибкая настройка через `.env` файлы.
*   **Database Migrations:** Автоматическое применение миграций при запуске.
*   **Validation:** Валидация конфигурации и входящих данных.
*   **Docker & Docker Compose:** Легкое развертывание всех компонентов.
*   **Testing:** Unit-тесты, интеграционные тесты и E2E-тесты.
*   **Swagger Documentation:** Автоматически сгенерированная документация API.
*   **Retry Logic:** Повторные попытки подключения к БД и обработки транзакций.

## Архитектура

Сервис состоит из следующих основных компонентов:

1.  **Transport Layer (`internal/transport`):**
    *   `http`: HTTP-сервер, обработчики API (Gin), middleware.
    *   `kafka`: Kafka consumer для получения заказов, DLQ processor.
2.  **Business Logic Layer (`internal/service`):** Основная логика работы с заказами (получение, создание, кэширование).
3.  **Data Access Layer (`internal/repository`):** Репозитории для взаимодействия с PostgreSQL.
4.  **Entities (`internal/entity`):** Модели данных (Order, Delivery, Payment, Item).
5.  **Configuration (`internal/config`):** Загрузка и валидация конфигурации из `.env`.
6.  **Packages (`pkg`):**
    *   `cache`: Реализация LRU-кэша.
    *   `kafka`: Вспомогательные функции для работы с Kafka.
    *   `kafka/dlq`: Логика DLQ.
    *   `logger`: Адаптер логгера (Zap).
    *   `metric`: Сбор и экспорт метрик (Prometheus).
    *   `storage/postgres`: Подключение к PostgreSQL, менеджер транзакций.
7.  **Infrastructure:**
    *   `migrations`: SQL-скрипты для создания таблиц.
    *   `Dockerfile`, `Dockerfile.producer`: Docker-образы для сервиса и эмулятора.
    *   `docker-compose.yml`: Оркестрация всех сервисов (PostgreSQL, Kafka, Zookeeper, Prometheus, Grafana, Order Service).
    *   `Makefile`: Автоматизация сборки, запуска, тестирования.
    *   `web/index.html`: Простой веб-интерфейс для поиска заказов.

## Технологии

*   **Go 1.25**
*   **Gin** (HTTP framework)
*   **pgx/v5** (PostgreSQL driver)
*   **segmentio/kafka-go** (Kafka client)
*   **zap** (Logger)
*   **prometheus/client_golang** (Metrics)
*   **testify** (Testing)
*   **gomock** (Mocking)
*   **Docker & Docker Compose**
*   **PostgreSQL 16**
*   **Kafka (Confluent Platform 7.6.1)**
*   **Prometheus**
*   **Grafana**

## Быстрый старт

### Требования

*   Docker & Docker Compose
*   Go 1.25+ (для локальной разработки)
*   Make (опционально, для использования Makefile)

### Запуск с помощью Docker Compose

1.  Клонируйте репозиторий:
    ```bash
    git clone <your-repo-url>
    cd order-service
    ```
2.  (Опционально) Отредактируйте `.env` файл, если необходимо изменить параметры по умолчанию.
3.  Запустите все сервисы:
    ```bash
    make compose-up-all
    # Или напрямую через docker-compose
    # docker-compose up --build -d
    ```
4.  Дождитесь запуска всех сервисов. Проверить статус можно через:
    ```bash
    docker-compose ps
    ```
5.  Сервис будет доступен:
    *   **API:** `http://localhost:8080`
    *   **Web UI:** `http://localhost:8080` (откроется `web/index.html`)
    *   **Metrics:** `http://localhost:8081/metrics`
    *   **Prometheus:** `http://localhost:9090`
    *   **Grafana:** `http://localhost:3000` (логин/пароль: `admin`/`grafana`)

### Использование

1.  Откройте веб-интерфейс: `http://localhost:8080`
2.  Отправьте тестовое сообщение в Kafka топик `orders`. Для этого можно использовать эмулятор:
    ```bash
    # В отдельном терминале
    make run-producer # или docker-compose -f docker-compose.yml run --rm kafka-producer
    ```
3.  Введите UID заказа (например, `b563feb7b2b84b6test` из примера) в поле ввода веб-интерфейса и нажмите "Search".
4.  Проверьте API напрямую:
    ```bash
    curl http://localhost:8080/api/v1/order/b563feb7b2b84b6test
    ```

### Swagger документация

Swagger документация доступна по адресу: `http://localhost:8080/swagger/index.html` (после запуска сервиса).

Для генерации документации используется `swag`:
```bash
make swag-v1

Разработка 
Локальный запуск 

    Убедитесь, что PostgreSQL и Kafka запущены (например, через docker-compose up -d db kafka zookeeper).
    Создайте БД и примените миграции:
    bash
     

 
1
make migrate-up
 
 
Запустите сервис:
bash
 

     
    1
    2
    3
    make run
    # Или напрямую
    # go run -tags migrate ./cmd/order-service
     
     
     

Тестирование 

    Unit-тесты:
    bash
     

 
1
2
3
make test
# Или напрямую
# go test -v -race -covermode atomic -coverprofile=coverage.txt ./internal/...
 
 
Интеграционные тесты:
bash
 
 
1
2
3
make compose-up-integration-test
# Или напрямую
# docker-compose -f docker-compose.yml -f tests/integration/docker-compose-integration-test.yml up --build --abort-on-container-exit --exit-code-from integration-test
 
 
E2E тесты:
bash
 
 
1
2
# Требует запущенного основного приложения (make compose-up-all)
# docker-compose -f docker-compose.yml -f tests/e2e/docker-compose-e2e.yaml up --build --abort-on-container-exit --exit-code-from e2e-test
 
 
Проверка зависимостей:
bash
 
 
1
make deps-audit
 
 
Линтинг:
bash
 

     
    1
    2
    3
    make linter-golangci
    make linter-hadolint
    make linter-dotenv
     
     
     

Команды Makefile 

Для просмотра всех доступных команд make просто выполните: 
bash
 
 
1
2
3
make
# Или
make help
 
 
Конфигурация 

Конфигурация осуществляется через переменные окружения, определенные в .env файле. Пример конфигурации находится в .env.example. 

Основные группы конфигурации: 

    APP_*: Параметры приложения.
    DB_*: Подключение к PostgreSQL.
    CACHE_*: Настройки кэша.
    HTTP_*: Настройки HTTP-сервера.
    KAFKA_*: Подключение к Kafka.
    DLQ_*: Настройки Dead Letter Queue.
    METRICS_*: Настройки сервера метрик.
    LOGGER_*: Настройки логирования.
     

Метрики 

Сервис экспортирует метрики в формате Prometheus по адресу /metrics на порту METRICS_PORT (по умолчанию 8081). 

Доступны метрики по: 

    HTTP запросам (количество, длительность, медленные запросы)
    Kafka сообщениям (обработанные, ошибки, lag)
    Кэшу (hit/miss, eviction, размер)
    Транзакциям БД (успехи, ошибки, retry)
     

Структура проекта 
 
 
1
2
3
4
5
6
7
8
9
10
11
12
13
14
15
16
17
18
19
20
21
22
23
24
25
26
27
28
29
30
31
32
33
34
.
├── cmd/                    # Точка входа в приложение
│   ├── order-service/      # Основной сервис
│   └── producer-service/   # Эмулятор Kafka producer
├── configs/                # Конфигурационные файлы (.env)
├── docs/                   # Swagger документация (генерируется)
├── internal/               # Основной код приложения
│   ├── app/                # Инициализация приложения
│   ├── config/             # Загрузка конфигурации
│   ├── entity/             # Модели данных
│   ├── repository/         # Репозитории для работы с БД
│   ├── service/            # Бизнес-логика
│   └── transport/          # Транспортные слои (HTTP, Kafka)
├── migrations/             # SQL миграции БД
├── pkg/                    # Переиспользуемые пакеты
│   ├── cache/              # Реализация кэша
│   ├── kafka/              # Вспомогательные функции Kafka
│   ├── logger/             # Логгер
│   ├── metric/             # Метрики
│   └── storage/postgres/   # Работа с PostgreSQL
├── tests/                  # Тесты
│   ├── integration/        # Интеграционные тесты
│   └── e2e/                # End-to-end тесты
├── volumes/                # Данные для Prometheus и Grafana
├── web/                    # Статические файлы (index.html)
├── .env                    # Локальный .env файл
├── .env.example            # Пример .env файла
├── docker-compose.yml      # Основной docker-compose файл
├── Dockerfile              # Dockerfile для основного сервиса
├── Dockerfile.producer     # Dockerfile для эмулятора
├── go.mod                  # Go модули
├── go.sum                  # Go суммы
├── Makefile                # Makefile для автоматизации
└── README.md               # Этот файл
 
 
Вклад 

Пулл-реквесты приветствуются. Для значительных изменений, пожалуйста, сначала откройте issue для обсуждения. 
Лицензия 

MIT  
Контакты 

Ваше имя - your-email@example.com  
 
 
1
2
3
4
5
6
7
8
9
10
11

**Инструкции по адаптации:**

1.  Замените `your-username` и `<your-repo-url>` на актуальные данные вашего репозитория.
2.  Убедитесь, что путь к эмулятору Kafka (`kafka-producer`) в `cmd/producer-service/` и соответствующий `Dockerfile.producer` настроены корректно, если вы хотите использовать `make run-producer`.
3.  Проверьте, соответствует ли структура каталогов в `README.md` фактической структуре вашего проекта (например, `tests/intergration` vs `tests/integration` в выводе).
4.  Обновите раздел "Контакты" своими данными.
5.  Добавьте бейджи (badges) в начале файла, если планируете использовать GitHub Actions для CI/CD и отчетов о качестве кода.
6.  Проверьте команды `make`, чтобы убедиться, что они соответствуют вашему `Makefile`.

Этот `README.md` дает полное представление о вашем проекте, его возможностях и том, как с ним работать.