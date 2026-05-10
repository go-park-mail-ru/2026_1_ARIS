#!/usr/bin/env sh
set -eu

: "${POSTGRES_DB:?POSTGRES_DB is required}"
: "${POSTGRES_USER:?POSTGRES_USER is required}"

: "${DB_MIGRATOR_USER:=aris_migrator}"
: "${DB_MIGRATOR_PASSWORD:=${DB_PASSWORD:-aris_migrator_password}}"
: "${DB_MONITOR_USER:=aris_monitor}"
: "${DB_MONITOR_PASSWORD:=${DB_PASSWORD:-aris_monitor_password}}"

: "${DB_AUTH_USER:=aris_auth}"
: "${DB_AUTH_PASSWORD:=${DB_PASSWORD:-aris_auth_password}}"
: "${DB_MEDIA_USER:=aris_media}"
: "${DB_MEDIA_PASSWORD:=${DB_PASSWORD:-aris_media_password}}"
: "${DB_USER_SERVICE_USER:=aris_user}"
: "${DB_USER_SERVICE_PASSWORD:=${DB_PASSWORD:-aris_user_password}}"
: "${DB_POST_USER:=aris_post}"
: "${DB_POST_PASSWORD:=${DB_PASSWORD:-aris_post_password}}"
: "${DB_CHAT_USER:=aris_chat}"
: "${DB_CHAT_PASSWORD:=${DB_PASSWORD:-aris_chat_password}}"
: "${DB_SUPPORT_USER:=aris_support}"
: "${DB_SUPPORT_PASSWORD:=${DB_PASSWORD:-aris_support_password}}"
: "${DB_COMMUNITY_USER:=aris_community}"
: "${DB_COMMUNITY_PASSWORD:=${DB_PASSWORD:-aris_community_password}}"
: "${DB_SEARCH_USER:=aris_search}"
: "${DB_SEARCH_PASSWORD:=${DB_PASSWORD:-aris_search_password}}"

psql_cmd() {
	psql --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" -v ON_ERROR_STOP=1 "$@"
}

create_or_update_login_role() {
	role_name="$1"
	role_password="$2"

	psql_cmd -v role_name="$role_name" -v role_password="$role_password" <<'SQL'
SELECT format('CREATE ROLE %I LOGIN PASSWORD %L', :'role_name', :'role_password')
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = :'role_name')
\gexec

SELECT format('ALTER ROLE %I WITH LOGIN PASSWORD %L', :'role_name', :'role_password')
WHERE EXISTS (SELECT 1 FROM pg_roles WHERE rolname = :'role_name')
\gexec
SQL
}

grant_app_base() {
	role_name="$1"

	psql_cmd -v role_name="$role_name" <<'SQL'
SELECT format('GRANT aris_app_base TO %I', :'role_name')
WHERE EXISTS (SELECT 1 FROM pg_roles WHERE rolname = :'role_name')
\gexec
SQL
}

psql_cmd -v database="$POSTGRES_DB" <<'SQL'
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;

REVOKE ALL ON DATABASE :"database" FROM PUBLIC;
REVOKE CREATE ON SCHEMA public FROM PUBLIC;

SELECT 'CREATE ROLE aris_app_base NOLOGIN'
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'aris_app_base')
\gexec
SQL

create_or_update_login_role "$DB_MIGRATOR_USER" "$DB_MIGRATOR_PASSWORD"
create_or_update_login_role "$DB_MONITOR_USER" "$DB_MONITOR_PASSWORD"
create_or_update_login_role "$DB_AUTH_USER" "$DB_AUTH_PASSWORD"
create_or_update_login_role "$DB_MEDIA_USER" "$DB_MEDIA_PASSWORD"
create_or_update_login_role "$DB_USER_SERVICE_USER" "$DB_USER_SERVICE_PASSWORD"
create_or_update_login_role "$DB_POST_USER" "$DB_POST_PASSWORD"
create_or_update_login_role "$DB_CHAT_USER" "$DB_CHAT_PASSWORD"
create_or_update_login_role "$DB_SUPPORT_USER" "$DB_SUPPORT_PASSWORD"
create_or_update_login_role "$DB_COMMUNITY_USER" "$DB_COMMUNITY_PASSWORD"
create_or_update_login_role "$DB_SEARCH_USER" "$DB_SEARCH_PASSWORD"

grant_app_base "$DB_AUTH_USER"
grant_app_base "$DB_MEDIA_USER"
grant_app_base "$DB_USER_SERVICE_USER"
grant_app_base "$DB_POST_USER"
grant_app_base "$DB_CHAT_USER"
grant_app_base "$DB_SUPPORT_USER"
grant_app_base "$DB_COMMUNITY_USER"
grant_app_base "$DB_SEARCH_USER"

psql_cmd \
	-v database="$POSTGRES_DB" \
	-v migrator_user="$DB_MIGRATOR_USER" \
	-v monitor_user="$DB_MONITOR_USER" <<'SQL'
GRANT CONNECT ON DATABASE :"database" TO aris_app_base;
GRANT CONNECT ON DATABASE :"database" TO :"migrator_user";
GRANT CONNECT ON DATABASE :"database" TO :"monitor_user";

ALTER SCHEMA public OWNER TO :"migrator_user";
GRANT USAGE, CREATE ON SCHEMA public TO :"migrator_user";
GRANT USAGE ON SCHEMA public TO aris_app_base;

GRANT pg_monitor TO :"monitor_user";

SELECT format('ALTER TABLE %I.%I OWNER TO %I', schemaname, tablename, :'migrator_user')
FROM pg_tables
WHERE schemaname = 'public'
	AND tablename NOT LIKE 'pg_stat_statements%'
\gexec

SELECT format('ALTER SEQUENCE %I.%I OWNER TO %I', schemaname, sequencename, :'migrator_user')
FROM pg_sequences
WHERE schemaname = 'public'
\gexec

SELECT format('ALTER VIEW %I.%I OWNER TO %I', schemaname, viewname, :'migrator_user')
FROM pg_views
WHERE schemaname = 'public'
	AND viewname NOT LIKE 'pg_stat_statements%'
\gexec

SELECT format('ALTER FUNCTION %s OWNER TO %I', proc.oid::regprocedure, :'migrator_user')
FROM pg_proc proc
JOIN pg_namespace namespace ON namespace.oid = proc.pronamespace
WHERE namespace.nspname = 'public'
	AND proc.proname NOT LIKE 'pg_stat_statements%'
\gexec

GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO :"migrator_user";
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO :"migrator_user";
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO :"migrator_user";

ALTER DEFAULT PRIVILEGES FOR ROLE :"migrator_user" IN SCHEMA public
	GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO aris_app_base;
ALTER DEFAULT PRIVILEGES FOR ROLE :"migrator_user" IN SCHEMA public
	GRANT USAGE, SELECT, UPDATE ON SEQUENCES TO aris_app_base;
ALTER DEFAULT PRIVILEGES FOR ROLE :"migrator_user" IN SCHEMA public
	GRANT EXECUTE ON FUNCTIONS TO aris_app_base;

GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO aris_app_base;
GRANT USAGE, SELECT, UPDATE ON ALL SEQUENCES IN SCHEMA public TO aris_app_base;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO aris_app_base;

DO $$
BEGIN
	IF to_regclass('public.pg_stat_statements') IS NOT NULL THEN
		REVOKE ALL ON TABLE public.pg_stat_statements FROM aris_app_base;
	END IF;

	IF to_regclass('public.pg_stat_statements_info') IS NOT NULL THEN
		REVOKE ALL ON TABLE public.pg_stat_statements_info FROM aris_app_base;
	END IF;

	IF to_regclass('public.schema_migrations') IS NOT NULL THEN
		REVOKE ALL ON TABLE public.schema_migrations FROM aris_app_base;
	END IF;
END
$$;

SELECT pg_stat_statements_reset();
SQL
