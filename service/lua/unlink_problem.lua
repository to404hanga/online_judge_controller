local p = ARGV[1]
local keys = redis.call('KEYS', p)
local deleted = 0
local batch = 500
for i = 1, #keys, batch do
    local slice = {}
    local j = i
    while j <= #keys and j < i + batch do
        table.insert(slice, keys[j])
        j = j + 1
    end
    if #slice > 0 then
        deleted = deleted + redis.call('UNLINK', unpack(slice))
    end
end
return deleted
