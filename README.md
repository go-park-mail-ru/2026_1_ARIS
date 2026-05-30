# 2026_1_ARIS

Проект "ВК" команды АРИС

### Команда

- [Сергей Шульгиненко](https://github.com/londonwaterloo)
- [Иван Хвостов](https://github.com/KokInside)

### Менторы

- [Софья Ситниченко](https://github.com/sonichka-s) — Frontend
- [Константин Галанин](https://github.com/KonstantinGalanin) — Backend
- [Владислав Алехин](https://github.com/3kybika) — Database
- Даниил Хасьянов - UX

### Ссылки

- [Деплой](https://arisnet.ru)
- [Frontend repository](https://github.com/frontend-park-mail-ru/2026_1_ARIS)
- [Figma](https://www.figma.com/design/fhzdyBQ8qjNFRCRVriSrK9/VK.com?node-id=8-16&p=f&t=u2EXBO6Pxh6QqWVC-0)

### CI/CD

Backend проверяется и деплоится через GitHub Actions. Пуш в `ARIS-*`, включая `ARIS-197`, запускает только CI-проверки без деплоя. Пуш в `dev` после успешных проверок собирает Docker-образы и деплоит staging на `https://arisnet.online`. Пуш в `main` или `deploy` после успешных проверок собирает Docker-образы и деплоит production на `https://arisnet.ru`.

Обязательный gate для CI/CD:

- `make lint` - проверяет `gofmt`, `go vet` и `staticcheck`.
- `make test` - запускает `go test -v ./...`.
- `make ci` - запускает lint и тесты; deploy jobs стартуют только после успешного `make ci`.

GitHub Actions использует следующие secrets:

- Staging: `STAGING_DEPLOY_HOST`, `STAGING_DEPLOY_USER`, `STAGING_DEPLOY_SSH_KEY`, `STAGING_DEPLOY_PATH`.
- Production: `PROD_DEPLOY_HOST`, `PROD_DEPLOY_USER`, `PROD_DEPLOY_SSH_KEY`, `PROD_DEPLOY_PATH`.

На staging-сервере в `.env.server` должны быть заданы `APP_ENDPOINT=https://arisnet.online` и `APP_DOMAIN=arisnet.online`. На production-сервере - `APP_ENDPOINT=https://arisnet.ru` и `APP_DOMAIN=arisnet.ru`.

### Информация о проекте

- Команда: АРИС
- Состав команды:
  - [Сергей Шульгиненко](https://github.com/londonwaterloo) - Frontend
  - [Иван Хвостов](https://github.com/KokInside) - Backend
- Production: https://arisnet.ru
- Swagger: https://arisnet.ru/swagger/index.html
- Figma: https://www.figma.com/design/fhzdyBQ8qjNFRCRVriSrK9/VK.com?node-id=8-16&p=f&t=u2EXBO6Pxh6QqWVC-0
- Frontend repository: https://github.com/frontend-park-mail-ru/2026_1_ARIS

### How-to-run

Локальный backend запускается через Docker Compose.

1. Скопируйте локальное окружение:

```bash
cp .env.example .env
```

2. Запустите backend-инфраструктуру и микросервисы:

```bash
make local-up
```

3. После изменений в backend пересоберите контейнеры:

```bash
make local-rebuild
```

4. Для остановки или полного сброса окружения используйте:

```bash
make local-down
make local-reset
```

Backend API будет доступен на `http://localhost:8080`, healthcheck - на `http://localhost:8080/health`.

Frontend запускается из соседнего репозитория:

```bash
cd ../arisfront
npm install
BACKEND_URL=http://localhost:8080 npm run dev
```

Frontend будет доступен на `http://localhost:3001`.
