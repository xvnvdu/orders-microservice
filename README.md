# Orders Microservice
**Сервис для обработки и отображения данных заказа с использованием Kafka, PostgreSQL и Redis.**

## Скриншоты
![image_1](images/homepage.png)
![image_2](images/order-not-found.png)
![image_3](images/404.png)
![image_4](images/400.png)
![image_5](images/docs.png)

## Функционал
- Генерация и отправка заказов в топик
- Получение данных о заказе из Kafka
- Сохранение полученных заказов в PostgreSQL
- Кэширование новых заказов для быстрого доступа
- Кэширование последних заказов на старте сервиса
- HTTP API для получения данных о заказе по ID
- Минималистичный интерфейс для просмотра заказов

## Технологии
- **Язык:** Golang 1.23.5
- **База данных:** PostgreSQL 17.6
- **Кэш:** Redis – Sorted Set (LRU)
- **Брокер сообщений:** Bitnami/Kafka (Legacy)
- **Контейнеризация:** Docker, Docker Compose
- **Веб-сервер:** net/http
- **SQL-Go код:** sqlc
- **Документация:** Swagger
- **Моки:** mock/gomock (uber)

## Установка и запуск
### Требования и зависимости
1) Установите Docker и Docker Compose на вашу систему соответствующим образом

### Подготовка к запуску
1) Перейдите в директорию, куда хотите сохранить сервис или создайте новую:
```
mkdir <YOUR-DIRECTORY-NAME> && cd <YOUR-DIRECTORY-NAME>
```
2) Клонируйте репозиторий проекта внутри директории:
```
git clone https://github.com/xvnvdu/orders-microservice.git
```
3) Запустите сборку и ожидайте ее окончания:
```
docker compose up -d
```

Эта команда запустит контейнеры в фоновом режиме:
- Сам сервис на порту **8080**
- PostgreSQL на порту **5432**
- Redis на порту **6379**
- Kafka на порту **9092**

4) Проверьте успешность запуска по логам:
```
docker logs orders-microservice-backend-1
```
В случае успешного запуска интерфейс будет доступен в вашем любимом браузере на ```localhost:8080```

### Основные эндпоинты
- ```/orders``` – список всех сохраненных заказов в формате JSON
- ```/orders/{order_uid}``` – информация о заказе в формате JSON, где ```{order_uid}``` – ID заказа
- ```/random/{amount}``` – генерация заказов, где ```{amount}``` – число генерируемых заказов 
- ```/docs``` – мини-документация Swagger 

### Полезное
1) Вы можете посмотреть список всех контейнеров (в том числе неактивные) и их статусы:
```
docker ps -a
```
2) Для взаимодействия с бд напрямую через контейнер используйте:
```
docker exec -it orders-microservice-db-1 psql -U orders_user -d orders_db 
```
Для выхода используйте ```exit```

3) Прекратить работу контейнеров:
```
docker compose down
```
Для очистки томов добавьте флаг ```-v```

## Тестирование
### База данных (65.2%)
1) Перед тем как запустить тесты для бд, пожалуйста, создайте тестовую базу данных, используя команду ниже:
```
docker exec -it orders-microservice-db-1 psql -U orders_user -d orders_db -c "CREATE DATABASE orders_test_db;"
```
2) Также необходимо инициализировать таблицы в бд, для этого используйте следующую команду:
```
docker exec -it orders-microservice-db-1 psql -U orders_user -d orders_test_db -f /docker-entrypoint-initdb.d/init.sql
```
3) После чего вы можете запустить тесты следующим образом:
```
go test -coverprofile=coverage.out ./internal/repository/ -v
```
4) А также проверить покрытие в сгенерированном html файле:
```
go tool cover -html=coverage.out
```

#### Результаты тестирования базы данных:
```
go test -coverprofile=coverage.out ./internal/repository/ -v
=== RUN   TestSaveAndGet
2025/11/24 01:35:17 Database connection opened on db:5432
    repository_test.go:61: Test order saved to orders_test_db, order uid: 0683dddb-4677-4559-8700-0105211420fc
    repository_test.go:66: Preparing for retrieving order from DB...
    repository_test.go:71: Order retrieved successfully
    repository_test.go:74: Expected OrderUID: 0683dddb-4677-4559-8700-0105211420fc, got: 0683dddb-4677-4559-8700-0105211420fc
    repository_test.go:77: Expected TrackNumber: MLDNTHLK2U7F12, got: MLDNTHLK2U7F12
    repository_test.go:80: Expected CustomerID: df4fec17-81ca-4621-b05d-b145cbf794e6, got: df4fec17-81ca-4621-b05d-b145cbf794e6
--- PASS: TestSaveAndGet (0.11s)
=== RUN   TestGetAllOrders
2025/11/24 01:35:17 Database connection opened on db:5432
    repository_test.go:89: Preparing to generate 19 random orders total...
    repository_test.go:97: Successfully saved 19 orders to DB
    repository_test.go:119: Successfully retrieved orders list from DB. Expected: 19, got: 19
--- PASS: TestGetAllOrders (12.26s)
=== RUN   TestGetLatestOrders
2025/11/24 01:35:29 Database connection opened on db:5432
    repository_test.go:89: Preparing to generate 83 random orders total...
    repository_test.go:97: Successfully saved 83 orders to DB
    repository_test.go:139: Preparing to retrieve 50 latest orders...
    repository_test.go:144: Successfully retrieved latest orders. Expected: 50/83, got: 50
--- PASS: TestGetLatestOrders (1.11s)
PASS
coverage: 65.2% of statements
ok  	orders/internal/repository	(cached)	coverage: 65.2% of statements
```

### Кэш (64%)
1) Запустите тесты следующей командой:
```
go test -coverprofile=coverage.out ./internal/cache/ -v
```
2) После чего вы также можете проверить покрытие:
```
go tool cover -html=coverage.out
```

#### Результаты тестирования кэша:
```
go test -coverprofile=coverage.out ./internal/cache/ -v
=== RUN   TestCachePutAndGet
    cache_test.go:30: Cache updated successfully
    cache_test.go:35: Order retrieved successfully
    cache_test.go:39: Retrieved order is not nil
    cache_test.go:43: Expected: 1d8b238a-3b90-4440-a61a-cfcb4d552f13 | Got: 1d8b238a-3b90-4440-a61a-cfcb4d552f13
    cache_test.go:46: Expected: eba31b0a-7e97-4b24-9829-4f37bf7d9b6a | Got: eba31b0a-7e97-4b24-9829-4f37bf7d9b6a
    cache_test.go:49: Expected: EA2FDGDO2G3MC8 | Got: EA2FDGDO2G3MC8
--- PASS: TestCachePutAndGet (0.00s)
=== RUN   TestCacheLRU
    cache_test.go:69: Cache filled with first order: aac95b64-2f8d-4921-b48f-2d67353eb779
    cache_test.go:71: Filling cache with 200 more orders...
    cache_test.go:80: Cache filled with 200/200 orders
    cache_test.go:85: Current cache capacity: 200
    cache_test.go:86: Total orders count: 201
2025/11/24 02:24:37 Can't find cached data for aac95b64-2f8d-4921-b48f-2d67353eb779
    cache_test.go:92: First order is no longer cached: was displaced by newcomers
--- PASS: TestCacheLRU (0.04s)
=== RUN   TestLoadInitialOrders
    cache_test.go:116: Cache capacity before loading initial orders: 0
    cache_test.go:118: Filling cache with orders...
2025/11/24 02:24:37 Cache filled with 67/200 orders, running on redis:6379
    cache_test.go:124: Expected 67/200 orders, got 67
--- PASS: TestLoadInitialOrders (0.02s)
PASS
coverage: 64.0% of statements
ok  	orders/internal/cache	0.062s	coverage: 64.0% of statements
```

### Брокер сообщений (31.2%)
1) Запуск тестов:
```
go test -coverprofile=coverage.out ./internal/kafka/ -v
```
2) Проверка покрытия:
```
go tool cover -html=coverage.out
```

#### Результаты тестирования брокера сообщений:
```
go test -coverprofile=coverage.out ./internal/kafka/ -v
=== RUN   TestValidateOrders
=== RUN   TestValidateOrders/All_orders_are_valid
    consumer_test.go:17: Preparing to generate 10 valid orders...
    consumer_test.go:23: All 10 orders were validated. Returned 10/10 as valid
=== RUN   TestValidateOrders/Having_invalid_orders
    consumer_test.go:28: Generating order with invalid OrderUID...
    consumer_test.go:33: Generating order with invalid TrackNumber...
    consumer_test.go:38: Generating order with invalid CustomerID...
    consumer_test.go:43: Generating order with invalid Phone number...
    consumer_test.go:48: Generating valid order...
    consumer_test.go:61: Total orders in one message: 5
    consumer_test.go:64: Expected valid orders in one message: 1
2025/11/24 02:28:47 Invalid order data found: missing OrderUID. Ignoring this order
2025/11/24 02:28:47 Invalid order data found: missing TrackNumber. Ignoring this order
2025/11/24 02:28:47 Invalid order data found: missing CustomerID. Ignoring this order
2025/11/24 02:28:47 Invalid order phone data found: starts with 0. Ignoring this order
    consumer_test.go:69: All 5 orders were validated. Returned 1/1 as valid
--- PASS: TestValidateOrders (0.00s)
    --- PASS: TestValidateOrders/All_orders_are_valid (0.00s)
    --- PASS: TestValidateOrders/Having_invalid_orders (0.00s)
=== RUN   TestWriteMessage
=== RUN   TestWriteMessage/Successful_write
=== RUN   TestWriteMessage/Failed_write
2025/11/24 02:28:47 Failed to write message: Simulated Kafka error
--- PASS: TestWriteMessage (0.00s)
    --- PASS: TestWriteMessage/Successful_write (0.00s)
    --- PASS: TestWriteMessage/Failed_write (0.00s)
PASS
coverage: 31.2% of statements
ok  	orders/internal/kafka	0.006s	coverage: 31.2% of statements
```

## Архитектура
1) **```cmd/server/main.go```**
- Основной исполняемый файл. 
- Инициализирует переменные окружения, зависимости и само приложение
- Запускает HTTP-сервер

2) **```internal/app/app.go```**
- Ядро приложения
- Управляет всеми процессами сервиса: запуск/остановка
- Хранит в себе все внешние зависимости сервиса
- Выполняет обработку хэндлеров

3) **```internal/dependencies/dependencies.go```**
- Модуль инициализации внешних зависимостей сервиса
- Запускает новый кэш, бд, продюсера, консьюмера
- Также создает топик

4) **```internal/cache/cache.go```**
- Redis кэш на основе LRU
- Основная логика кэширования данных:
    - Инициализация кэша
    - Заполнение кэша на старте сервиса
    - Обновление кэша при взаимодействии с заказами из бд

5) **```internal/database/```**
- Сгенерированная с помощью sqlc директория для взаимодействия с бд
- Описаны основные модели, структура заказов
- Хранит SQL-Go функции для модифицирования базы данных напрямую

6) **```internal/generator/```**
- Директория генерации случайных заказов:
    - Содержит основной скрипт для генерации ```generator.go```
    - Реализованы вспомогательные модели для структуры заказов

7) **```internal/kafka/```**
- Ключевая логика брокера сообщений Kafka:
    - Консьюмер создает новый топик на старте сервиса и слушает сообщения фоном
    - Продюсер сообщений записывает сгенерированные заказы в топик
    - Консьюмер пытается сохранить полученное сообщение с заказами в бд
    - При неудаче сохранения в бд сообщение НЕ коммитится и повторно обрабатывается в будущем

8) **```internal/repository/repository.go```**
- Модуль для взаимодействия с сохраненными данными
- Инициализация и проверка успешного подключения к бд
- Хранит в себе объекты самой базы данных и кэша
- Сохраняет заказы в бд, извлекает их из кэша и бд

9) **```internal/mocks```**
- Содержит сгенерированные моки для внешних зависимостей:
    - ```cache_mock.go```
    - ```kafka_mock.go```
    - ```repository_mock.go```

10) **```sql/```**
- Стартовый скрипт инициализации таблиц для базы данных
- Основные sql-запросы для взаимодействия с бд

11) **```web/```**
- Содержит статику и шаблоны для web-страниц

12) **```docs/```**
- Документация Swagger, написанная в ```.yaml``` формате
- Описывает API-эндпоинты сервиса
- Рендерится на запуске программы в ```cmd/server/main.go```
- Доступ к документации через ```/docs/index.html``` (редирект через ```/docs```)

13) **```.env```**
- Переменные окружения, используемые приложением:
    - Строка подключения к PostgreSQL
    - Данные пользователя, название самой бд
    - Строка подключения к Redis

14) **```Dockerfile```** и **```docker-compose.yaml```**
- Файлы конфигурации Docker-окружения
- Создание образа приложения через ```Dockerfile```
- Инструкции для запуска основной инфраструктуры и самого сервиса в отдельных контейнерах 

15) **```sqlc.yaml```**
- Инструкция для генерации SQL-Go команд через sqlc

## Структура базы данных
![image_6](images/orders-database.png)

## Демо сервиса
#### https://disk.yandex.ru/i/eF3stexR_e1CRw
