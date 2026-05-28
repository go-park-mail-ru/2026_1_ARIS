# PostgreSQL access and diagnostics

## Configuration layout

PostgreSQL no longer receives operational settings as a long list of `postgres -c ...` options in Compose.

- `config/postgres/postgresql.conf` is the entry config and includes `config/postgres/conf.d`.
- `config/postgres/conf.d/aris.conf` contains runtime settings: `listen_addresses`, `max_connections`, `pg_stat_statements`, `auto_explain`, logging and lock diagnostics.
- `config/postgres/pg_hba.conf` restricts host access to loopback and Docker private networks with `scram-sha-256`.
- `config/postgres/pg_ident.conf` is mounted and reserved for peer-auth mappings if local socket auth is tightened.

Compose still passes `config_file=/etc/postgresql/postgresql.conf` to tell the official Postgres image to use the mounted config file. The actual PostgreSQL parameters live in versioned config files.

## Roles

`db/init/01_roles_and_observability.sh` creates:

- `aris_migrator` for migrations and ownership of application objects.
- `aris_monitor` for metrics collection with `pg_monitor`.
- Service roles: `aris_media`, `aris_user`, `aris_post`, `aris_chat`, `aris_support`, `aris_community`, `aris_search`, `aris_game`.
- `aris_app_base` as a shared NOLOGIN role for application DML privileges.

Runtime services connect through PgBouncer with their service roles. Migrations and seed connect directly to Postgres as `aris_migrator`.

## PgBouncer

PgBouncer is configured in `config/pgbouncer/pgbouncer.ini` and listens on `pgbouncer:6432`.

The current pool mode is `session`, which is conservative for Go/pgx services because it avoids prepared-statement issues that can appear with transaction pooling. After explicitly configuring pgx statement cache behavior, this can be revisited.

## Diagnostics

Postgres writes logs inside the `aris_postgres_data` volume under `log/postgresql.log` (see `log_directory` in `conf.d/aris.conf`). The config enables:

- slow query logging from 500 ms;
- lock-wait logging;
- temp-file logging;
- `auto_explain` plans for slow statements;
- `pg_stat_statements` for query statistics.

Generate a pgBadger report:

```bash
make db-pgbadger
```

The HTML report is written to `.tmp/db-observability/pgbadger.html`.
