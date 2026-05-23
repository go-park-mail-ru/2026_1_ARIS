local fiber = require('fiber')

local function env(name, fallback)
    local value = os.getenv(name)
    if value == nil or value == '' then
        return fallback
    end
    return value
end

box.cfg({
    listen = env('TARANTOOL_LISTEN', '0.0.0.0:3301'),
    work_dir = env('TARANTOOL_WORK_DIR', '/var/lib/tarantool'),
    memtx_memory = tonumber(env('TARANTOOL_MEMTX_MEMORY', '134217728')),
    wal_mode = env('TARANTOOL_WAL_MODE', 'write'),
})

local cache_user = env('TARANTOOL_USER', 'cache')
local cache_password = env('TARANTOOL_PASSWORD', 'local-tarantool-password')

if not box.schema.user.exists(cache_user) then
    box.schema.user.create(cache_user, {password = cache_password})
else
    box.schema.user.passwd(cache_user, cache_password)
end
box.schema.user.grant(cache_user, 'read,write,execute', 'universe', nil, {if_not_exists = true})

local function ensure_space(name, format, primary_parts)
    local space = box.space[name]
    if space == nil then
        space = box.schema.space.create(name, {if_not_exists = true})
    end
    space:format(format)
    if space.index.primary == nil then
        space:create_index('primary', {
            parts = primary_parts,
            if_not_exists = true,
        })
    end
    return space
end

ensure_space('profile_details', {
    {name = 'profile_id', type = 'unsigned'},
    {name = 'payload', type = 'string'},
    {name = 'updated_at', type = 'number'},
}, {{field = 'profile_id', type = 'unsigned'}})

ensure_space('profile_summaries', {
    {name = 'profile_id', type = 'unsigned'},
    {name = 'payload', type = 'string'},
    {name = 'updated_at', type = 'number'},
}, {{field = 'profile_id', type = 'unsigned'}})

ensure_space('auth_users', {
    {name = 'user_account_id', type = 'unsigned'},
    {name = 'payload', type = 'string'},
    {name = 'updated_at', type = 'number'},
}, {{field = 'user_account_id', type = 'unsigned'}})

ensure_space('profile_id_by_account', {
    {name = 'user_account_id', type = 'unsigned'},
    {name = 'profile_id', type = 'unsigned'},
    {name = 'updated_at', type = 'number'},
}, {{field = 'user_account_id', type = 'unsigned'}})

ensure_space('post_like_counts', {
    {name = 'post_id', type = 'unsigned'},
    {name = 'count', type = 'unsigned'},
    {name = 'updated_at', type = 'number'},
}, {{field = 'post_id', type = 'unsigned'}})

ensure_space('presence', {
    {name = 'user_account_id', type = 'unsigned'},
    {name = 'is_online', type = 'boolean'},
    {name = 'last_seen_at', type = 'number'},
    {name = 'updated_at', type = 'number'},
    {name = 'connections', type = 'unsigned'},
}, {{field = 'user_account_id', type = 'unsigned'}})

local function now()
    return fiber.time()
end

function presence_online(user_account_id)
    user_account_id = tonumber(user_account_id)
    local row = box.space.presence:get{user_account_id}
    local connections = 1
    if row ~= nil then
        connections = (tonumber(row.connections) or tonumber(row[5]) or 0) + 1
    end
    local ts = now()
    return box.space.presence:replace{user_account_id, true, ts, ts, connections}
end

function presence_offline(user_account_id)
    user_account_id = tonumber(user_account_id)
    local row = box.space.presence:get{user_account_id}
    local connections = 0
    if row ~= nil then
        connections = math.max((tonumber(row.connections) or tonumber(row[5]) or 1) - 1, 0)
    end
    local ts = now()
    return box.space.presence:replace{user_account_id, connections > 0, ts, ts, connections}
end

function presence_heartbeat(user_account_id)
    user_account_id = tonumber(user_account_id)
    local row = box.space.presence:get{user_account_id}
    if row == nil then
        return nil
    end
    local connections = tonumber(row.connections) or tonumber(row[5]) or 0
    if connections <= 0 then
        return row
    end
    local ts = now()
    return box.space.presence:replace{user_account_id, true, ts, ts, connections}
end
