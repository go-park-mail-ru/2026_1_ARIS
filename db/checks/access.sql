\pset pager off

SELECT
	rolname,
	rolcanlogin,
	rolsuper,
	rolcreatedb,
	rolcreaterole,
	rolinherit
FROM pg_roles
WHERE rolname LIKE 'aris_%'
ORDER BY rolname;

SELECT
	member_role.rolname AS member,
	parent_role.rolname AS granted_role
FROM pg_auth_members auth_members
JOIN pg_roles member_role ON member_role.oid = auth_members.member
JOIN pg_roles parent_role ON parent_role.oid = auth_members.roleid
WHERE member_role.rolname LIKE 'aris_%'
	OR parent_role.rolname LIKE 'aris_%'
ORDER BY member_role.rolname, parent_role.rolname;

SELECT
	rolname,
	has_database_privilege(rolname, current_database(), 'CONNECT') AS can_connect,
	has_database_privilege(rolname, current_database(), 'CREATE') AS can_create_database_objects,
	has_schema_privilege(rolname, 'public', 'USAGE') AS can_use_public_schema,
	has_schema_privilege(rolname, 'public', 'CREATE') AS can_create_in_public_schema
FROM pg_roles
WHERE rolname LIKE 'aris_%'
ORDER BY rolname;

SELECT
	grantee,
	table_schema,
	table_name,
	string_agg(privilege_type, ', ' ORDER BY privilege_type) AS privileges
FROM information_schema.role_table_grants
WHERE table_schema = 'public'
	AND grantee LIKE 'aris_%'
GROUP BY grantee, table_schema, table_name
ORDER BY grantee, table_name
LIMIT 80;
