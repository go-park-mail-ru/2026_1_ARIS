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

Локальное окружение с Go-микросервисами, Postgres, Redis, MinIO и nginx запускается из `arisback`.

1. Скопируйте `.env.compose.example` в `.env.compose`
2. Запустите:

```bash
make local-up
```

Что поднимется автоматически:

- Postgres
- Redis
- MinIO
- миграции и seed-данные
- Go-микросервисы `auth`, `media`, `user`, `post`, `chat`, `support`, `community`, `search`
- nginx-gateway на `http://localhost:8080`
- Prometheus на `http://localhost:9090`
- Grafana на `http://localhost:3000` с дашбордом `ARIS Service and Machine Metrics`
- node-exporter для метрик CPU, памяти и диска машины

Полный сброс локальной базы:

```bash
make reset-db
```

Логи сервисов:

```bash
make logs
```

### Запуск микросервисов на сервере

На сервере frontend запускается через systemd на хосте. Go-микросервисы, Postgres,
Redis, MinIO и nginx поднимаются из `arisback` через Docker Compose.

1. Скопируйте `.env.server.example` в `.env.server` и замените пароли.
2. Проверьте, что `APP_ENDPOINT=https://arisnet.ru`.
3. Поднимите инфраструктуру, Go-микросервисы и контейнерный nginx:

```bash
make server-up
```

Команда стартует `auth`, `media`, `user`, `post`, `chat`, `support`; их зависимости
`db`, `redis`, `minio`, `migrate`, `seed` Docker Compose поднимет автоматически. Серверный
nginx использует `config/nginx.container.server.conf`, слушает `80/443`, проксирует
frontend на `host.docker.internal:3001` и читает сертификаты из `/etc/letsencrypt`.

Проверка и reload контейнерного nginx:

```bash
make server-nginx-test
make server-nginx-reload
```

Если нужен nginx на хосте вместо контейнера, используйте `config/nginx.server.conf` и цели
`server-nginx-install`, `server-host-nginx-test`, `server-host-nginx-reload`.
