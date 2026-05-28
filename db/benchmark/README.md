# ARIS: нагрузочный тест PostgreSQL

Процесс нагрузочного тестирования для сценария хронологической ленты постов. 
Проверяемый SQL соответствует запросу из `postStorage.GetFeedPage`.

## Файлы

- `seed_feed_data.sql`: наполняет тестовую БД 5000 профилями и 500000 постами.
- `feed_public.sql`: транзакция `pgbench` для чтения публичной ленты.
- `explain_feed.sql`: `EXPLAIN (ANALYZE, BUFFERS)` для публичной ленты и ленты по авторам.
- `metrics.sql`: статистика `pg_stat_statements`, использование индексов и счетчики таблиц.
- `seed_chat_friendship_data.sql`: наполняет БД данными для проверки истории чата и списка друзей.
- `chat_messages.sql`: транзакция `pgbench` для чтения последних сообщений чата.
- `friendship_get_friends.sql`: транзакция `pgbench` для чтения списка друзей.
- `explain_chat_friendship.sql`: `EXPLAIN (ANALYZE, BUFFERS)` для истории чата и списка друзей.
- `metrics_chat_friendship.sql`: метрики для сценариев чата и друзей.

## Изолированная тестовая БД

Используется отдельная БД.

Загрузить переменные из `.env`.

```bash
set -a
. ./.env
set +a

export BENCH_DB_HOST="${DB_HOST:-127.0.0.1}"
export BENCH_DB_PORT="${DB_PORT:-5432}"
export BENCH_DB_USER="${DB_MIGRATOR_USER:-${DB_USER}}"
export BENCH_DB_PASSWORD="${DB_MIGRATOR_PASSWORD:-${DB_PASSWORD}}"
export BENCH_DB_NAME="${DB_NAME}"
export BENCH_SSL_MODE="${SSL_MODE:-disable}"
export BENCH_DATABASE_URL="postgres://${BENCH_DB_USER}:${BENCH_DB_PASSWORD}@${BENCH_DB_HOST}:${BENCH_DB_PORT}/${BENCH_DB_NAME}?sslmode=${BENCH_SSL_MODE}"
```

```bash
docker compose -p aris_perf_202 --env-file .env -f docker-compose.yml up -d db

for i in $(seq 1 60); do
  status=$(docker inspect -f '{{.State.Health.Status}}' aris_perf_202-db-1 2>/dev/null || echo missing)
  [ "$status" = healthy ] && break
  sleep 1
done
```

Для baseline нужно применить миграции только до `000030`. 
Миграция `000031` содержит оптимизацию индексов и применяется только после первого замера.

```bash
migrate -source file://./db/migrations \
  -database "$BENCH_DATABASE_URL" \
  up 30
```

Заполнение тестовыми данными:

```bash
PGPASSWORD="$BENCH_DB_PASSWORD" psql -h "$BENCH_DB_HOST" -p "$BENCH_DB_PORT" \
  -U "$BENCH_DB_USER" -d "$BENCH_DB_NAME" \
  -v ON_ERROR_STOP=1 \
  -f db/benchmark/seed_feed_data.sql
```

## Замер до оптимизации

Сбросить статистику запросов и снять планы выполнения:

```bash
PGPASSWORD="$BENCH_DB_PASSWORD" psql -h "$BENCH_DB_HOST" -p "$BENCH_DB_PORT" \
  -U "$BENCH_DB_USER" -d "$BENCH_DB_NAME" \
  -c 'SELECT pg_stat_statements_reset();'

PGPASSWORD="$BENCH_DB_PASSWORD" psql -h "$BENCH_DB_HOST" -p "$BENCH_DB_PORT" \
  -U "$BENCH_DB_USER" -d "$BENCH_DB_NAME" \
  -v ON_ERROR_STOP=1 \
  -f db/benchmark/explain_feed.sql
```

Прогреть БД коротким прогоном и выполнить основной нагрузочный тест:

```bash
PGPASSWORD="$BENCH_DB_PASSWORD" pgbench -h "$BENCH_DB_HOST" -p "$BENCH_DB_PORT" \
  -U "$BENCH_DB_USER" -d "$BENCH_DB_NAME" \
  -M prepared -n -c 8 -j 4 -T 10 \
  -f db/benchmark/feed_public.sql

PGPASSWORD="$BENCH_DB_PASSWORD" psql -h "$BENCH_DB_HOST" -p "$BENCH_DB_PORT" \
  -U "$BENCH_DB_USER" -d "$BENCH_DB_NAME" \
  -c 'SELECT pg_stat_statements_reset();'

PGPASSWORD="$BENCH_DB_PASSWORD" pgbench -h "$BENCH_DB_HOST" -p "$BENCH_DB_PORT" \
  -U "$BENCH_DB_USER" -d "$BENCH_DB_NAME" \
  -M prepared -n -c 8 -j 4 -T 30 \
  -f db/benchmark/feed_public.sql
```

Собрать метрики и логи:

```bash
PGPASSWORD="$BENCH_DB_PASSWORD" psql -h "$BENCH_DB_HOST" -p "$BENCH_DB_PORT" \
  -U "$BENCH_DB_USER" -d "$BENCH_DB_NAME" \
  -v ON_ERROR_STOP=1 \
  -f db/benchmark/metrics.sql

docker stats --no-stream --format 'table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.BlockIO}}' \
  aris_perf_202-db-1

docker exec aris_perf_202-db-1 sh -c 'tail -n 120 /var/lib/postgresql/data/log/postgresql.log'
```

## Применение оптимизации

Применение миграции `000031`:

```bash
migrate -source file://./db/migrations \
  -database "$BENCH_DATABASE_URL" \
  up 1

PGPASSWORD="$BENCH_DB_PASSWORD" psql -h "$BENCH_DB_HOST" -p "$BENCH_DB_PORT" \
  -U "$BENCH_DB_USER" -d "$BENCH_DB_NAME" \
  -c 'ANALYZE post;'
```

Добавленные индексы:

```sql
CREATE INDEX IF NOT EXISTS post_active_public_feed_idx
    ON post (is_public_demo, created_at DESC, id DESC)
    WHERE is_active = TRUE;

CREATE INDEX IF NOT EXISTS post_active_author_feed_idx
    ON post (author_id, is_public_demo, created_at DESC, id DESC)
    WHERE is_active = TRUE;
```

Обоснование:

- публичная лента фильтрует `is_active` и `is_public_demo`, затем сортирует по
  `(created_at DESC, id DESC)`;
- приватная лента дополнительно фильтрует `author_id = ANY(...)`;
- индексы позволяют избежать полного сканирования и сортировки таблицы `post`.

## Замер После оптимизации

После применения миграции нужно повторить шаги из блока "Замер До Оптимизации"

## Полученные результаты

Тестовая выборка:

- `profile`: 5000 строк.
- `post`: 500000 строк.
- строк, подходящих под публичную ленту: около 250000.

До оптимизации:

- время выполнения по `EXPLAIN`: 49.464 ms
- транзакций `pgbench`: 1437
- ошибок `pgbench`: 0
- средняя задержка `pgbench`: 167.351 ms
- pgbench TPS: 47.803604
- `pg_stat_statements.mean_exec_time`: 166.398 ms
- `pg_stat_statements.max_exec_time`: 334.386 ms
- попаданий в shared buffers: 10198815

После оптимизации:

- время выполнения по `EXPLAIN`: 0.089 ms
- транзакций `pgbench`: 586262
- ошибок `pgbench`: 0
- средняя задержка `pgbench`: 0.409 ms
- pgbench TPS: 19563.018660
- `pg_stat_statements.mean_exec_time`: 0.159 ms
- `pg_stat_statements.max_exec_time`: 1.971 ms
- попаданий в shared buffers: 2931310

Проверка ленты по автору:

- без `post_active_author_feed_idx`: 28.592 ms
- с `post_active_author_feed_idx`: 0.966 ms

Улучшение:

- `EXPLAIN`: 49.464 ms -> 0.089 ms, примерно в 556 раз быстрее.
- pgbench TPS: 47.803604 -> 19563.018660, примерно в 409 раз выше.
- средняя задержка `pgbench`: 167.351 ms -> 0.409 ms, примерно в 409 раз ниже.

## Дополнительные сценарии

После проверки ленты были выбраны еще два тяжелых запроса:

- история сообщений чата из `messageStorage.GetByChatID`;
- список друзей из `friendshipStorage.GetFriends`.

Заполнение данных:

```bash
PGPASSWORD="$BENCH_DB_PASSWORD" psql -h "$BENCH_DB_HOST" -p "$BENCH_DB_PORT" \
  -U "$BENCH_DB_USER" -d "$BENCH_DB_NAME" \
  -v ON_ERROR_STOP=1 \
  -f db/benchmark/seed_chat_friendship_data.sql
```

Тестовая выборка:

- `message`: 500000 строк в одном benchmark-чате.
- `friendship`: 1000000 строк.

Снять baseline:

```bash
PGPASSWORD="$BENCH_DB_PASSWORD" psql -h "$BENCH_DB_HOST" -p "$BENCH_DB_PORT" \
  -U "$BENCH_DB_USER" -d "$BENCH_DB_NAME" \
  -c 'SELECT pg_stat_statements_reset();'

PGPASSWORD="$BENCH_DB_PASSWORD" psql -h "$BENCH_DB_HOST" -p "$BENCH_DB_PORT" \
  -U "$BENCH_DB_USER" -d "$BENCH_DB_NAME" \
  -v ON_ERROR_STOP=1 \
  -f db/benchmark/explain_chat_friendship.sql

PGPASSWORD="$BENCH_DB_PASSWORD" pgbench -h "$BENCH_DB_HOST" -p "$BENCH_DB_PORT" \
  -U "$BENCH_DB_USER" -d "$BENCH_DB_NAME" \
  -M prepared -n -c 8 -j 4 -T 10 \
  -f db/benchmark/chat_messages.sql

PGPASSWORD="$BENCH_DB_PASSWORD" psql -h "$BENCH_DB_HOST" -p "$BENCH_DB_PORT" \
  -U "$BENCH_DB_USER" -d "$BENCH_DB_NAME" \
  -c 'SELECT pg_stat_statements_reset();'

PGPASSWORD="$BENCH_DB_PASSWORD" pgbench -h "$BENCH_DB_HOST" -p "$BENCH_DB_PORT" \
  -U "$BENCH_DB_USER" -d "$BENCH_DB_NAME" \
  -M prepared -n -c 8 -j 4 -T 30 \
  -f db/benchmark/chat_messages.sql

PGPASSWORD="$BENCH_DB_PASSWORD" pgbench -h "$BENCH_DB_HOST" -p "$BENCH_DB_PORT" \
  -U "$BENCH_DB_USER" -d "$BENCH_DB_NAME" \
  -M prepared -n -c 8 -j 4 -T 10 \
  -f db/benchmark/friendship_get_friends.sql

PGPASSWORD="$BENCH_DB_PASSWORD" psql -h "$BENCH_DB_HOST" -p "$BENCH_DB_PORT" \
  -U "$BENCH_DB_USER" -d "$BENCH_DB_NAME" \
  -c 'SELECT pg_stat_statements_reset();'

PGPASSWORD="$BENCH_DB_PASSWORD" pgbench -h "$BENCH_DB_HOST" -p "$BENCH_DB_PORT" \
  -U "$BENCH_DB_USER" -d "$BENCH_DB_NAME" \
  -M prepared -n -c 8 -j 4 -T 30 \
  -f db/benchmark/friendship_get_friends.sql
```

Применить миграцию `000032`:

```bash
migrate -source file://./db/migrations \
  -database "$BENCH_DATABASE_URL" \
  up 1

PGPASSWORD="$BENCH_DB_PASSWORD" psql -h "$BENCH_DB_HOST" -p "$BENCH_DB_PORT" \
  -U "$BENCH_DB_USER" -d "$BENCH_DB_NAME" \
  -c 'ANALYZE message; ANALYZE friendship;'
```

Добавленные индексы:

```sql
CREATE INDEX IF NOT EXISTS message_active_chat_created_idx
    ON message (chat_id, created_at DESC, id DESC)
    WHERE is_active = TRUE;

CREATE INDEX IF NOT EXISTS message_active_chat_id_idx
    ON message (chat_id, id ASC)
    WHERE is_active = TRUE;

CREATE INDEX IF NOT EXISTS friendship_requester_status_idx
    ON friendship (requester_id, status, addressee_id);

CREATE INDEX IF NOT EXISTS friendship_addressee_status_idx
    ON friendship (addressee_id, status, requester_id);
```

Повторить `EXPLAIN`, `pgbench` и сбор метрик:

```bash
PGPASSWORD="$BENCH_DB_PASSWORD" psql -h "$BENCH_DB_HOST" -p "$BENCH_DB_PORT" \
  -U "$BENCH_DB_USER" -d "$BENCH_DB_NAME" \
  -v ON_ERROR_STOP=1 \
  -f db/benchmark/explain_chat_friendship.sql

PGPASSWORD="$BENCH_DB_PASSWORD" psql -h "$BENCH_DB_HOST" -p "$BENCH_DB_PORT" \
  -U "$BENCH_DB_USER" -d "$BENCH_DB_NAME" \
  -v ON_ERROR_STOP=1 \
  -f db/benchmark/metrics_chat_friendship.sql
```

Результаты для истории сообщений:

- `EXPLAIN`: 72.483 ms -> 0.370 ms, примерно в 196 раз быстрее.
- pgbench TPS: 28.804493 -> 16294.175955, примерно в 566 раз выше.
- средняя задержка `pgbench`: 277.734 ms -> 0.491 ms, примерно в 566 раз ниже.
- `pg_stat_statements.mean_exec_time`: 276.473 ms -> 0.208 ms.

Результаты для списка друзей:

- `EXPLAIN`: 49.265 ms -> 2.263 ms, примерно в 22 раза быстрее.
- pgbench TPS: 62.538754 -> 3228.733028, примерно в 52 раза выше.
- средняя задержка `pgbench`: 127.921 ms -> 2.478 ms, примерно в 52 раза ниже.
- `pg_stat_statements.mean_exec_time`: 126.712 ms -> 1.925 ms.