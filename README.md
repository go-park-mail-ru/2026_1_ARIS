# 2026_1_ARIS

Проект "ВК" команды АРИС

### Команда

- [Сергей Шульгиненко](https://github.com/londonwaterloo)
- [Иван Хвостов](https://github.com/KokInside)
- [Ринат Байков](https://github.com/ronprog)

### Менторы

- [Софья Ситниченко](https://github.com/sonichka-s) — Frontend
- [Константин Галанин](https://github.com/KonstantinGalanin) — Backend
- [Владислав Алехин](https://github.com/3kybika) — Database
- Даниил Хасьянов - UX

### Деплой

[ARISNET](https://arisnet.ru/)

### Ссылки
- [Frontend repository](https://github.com/frontend-park-mail-ru/2026_1_ARIS)
- [Figma](https://www.figma.com/design/fhzdyBQ8qjNFRCRVriSrK9/VK.com?node-id=8-16&p=f&t=u2EXBO6Pxh6QqWVC-0)

### Локальный запуск

Единое локальное окружение для backend, frontend, Postgres и MinIO запускается из `arisback`.

1. Скопируйте `.env.compose.example` в `.env.compose`
2. Убедитесь, что рядом с `arisback` лежит папка `arisfront`
3. Запустите:

```bash
make dev
```

После готовности появится сообщение со ссылками на frontend, backend, MinIO и Postgres.

Что поднимется автоматически:

- Postgres
- MinIO
- миграции backend
- backend на `http://localhost:8080`
- frontend на `http://localhost:3001`

Полный сброс локальной базы:

```bash
make reset-db
```

Логи сервисов:

```bash
make logs
```

### Запуск на сервере

На сервере nginx и Node запускаются без контейнеров. Go-микросервисы, Postgres,
Redis и MinIO поднимаются из `arisback` через Docker Compose.

1. Скопируйте `.env.compose.example` в `.env.compose` и замените пароли.
2. Проверьте, что `APP_ENDPOINT=https://arisnet.ru`.
3. Поднимите инфраструктуру и Go-микросервисы:

```bash
make microservices-up
```

Команда стартует `auth`, `media`, `user`, `post`, `chat`; их зависимости
`db`, `redis`, `minio`, `migrate`, `seed` Docker Compose поднимет автоматически.
Порты инфраструктуры и микросервисов проброшены только на `127.0.0.1`.

Host nginx ставится отдельно:

```bash
make server-nginx-install
make server-nginx-reload
```

`config/nginx.server.conf` ожидает frontend Node на `127.0.0.1:3001`,
микросервисы на `127.0.0.1:8081-8085` и MinIO на `127.0.0.1:9000`.
