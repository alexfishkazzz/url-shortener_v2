# URL Shortener Service

Небольшой HTTP-сервис для сокращения ссылок на Go.

## Возможности

- Создание короткой ссылки из длинного URL
- Перенаправление с короткой ссылки на оригинальный URL
- Аутентификация пользователя через подписанную cookie (HMAC-SHA256)
- Получение списка всех сокращённых URL текущего пользователя

## Требования

- Go 1.22 или выше

## Запуск

```bash
go run ./cmd/shortener
```

Сервис запустится на `http://localhost:8080`.

## Тесты

```bash
go test ./...
```

## API

### Создать короткую ссылку

```http
POST /api/shorten
Content-Type: application/json

{
  "url": "https://example.com/some/long/path"
}
```

Ответ:

```json
{
  "result": "http://localhost:8080/abc123def456"
}
```

### Перейти по короткой ссылке

```http
GET /{id}
```

Перенаправляет на оригинальный URL (статус 307 Temporary Redirect).

### Получить все URL пользователя

```http
GET /api/user/urls
```

Ответ (если есть ссылки):

```json
[
  {
    "short_url": "http://localhost:8080/abc123def456",
    "original_url": "https://example.com/some/long/path"
  }
]
```

Если ссылок нет — `204 No Content`.

## Аутентификация

Сервис использует cookie `user_id` в формате `userID:signature`, где подпись — HMAC-SHA256.

Если cookie отсутствует или подпись неверна, сервис выдаёт новую cookie.
