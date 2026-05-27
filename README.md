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

- `make lint` - проверяет `gofmt` и `go vet`.
- `make test` - запускает `go test -v ./...`.
- `make ci` - запускает lint и тесты; deploy jobs стартуют только после успешного `make ci`.

GitHub Actions использует следующие secrets:

- Staging: `STAGING_DEPLOY_HOST`, `STAGING_DEPLOY_USER`, `STAGING_DEPLOY_SSH_KEY`, `STAGING_DEPLOY_PATH`.
- Production: `PROD_DEPLOY_HOST`, `PROD_DEPLOY_USER`, `PROD_DEPLOY_SSH_KEY`, `PROD_DEPLOY_PATH`.

На staging-сервере в `.env.server` должны быть заданы `APP_ENDPOINT=https://arisnet.online` и `APP_DOMAIN=arisnet.online`. На production-сервере - `APP_ENDPOINT=https://arisnet.ru` и `APP_DOMAIN=arisnet.ru`.
