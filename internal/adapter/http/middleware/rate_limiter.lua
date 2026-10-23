local capacity = tonumber(ARGV[1])
local refill_rate = tonumber(ARGV[2])
local now = tonumber(ARGV[3])

local data = redis.call("HMGET", KEYS[1], "tokens", "last_refill")
local tokens = tonumber(data[1])
local last_refill = tonumber(data[2])

if tokens == nil then
    tokens = capacity
    last_refill = now
end

local delta = now - last_refill
if delta > 0 then
    tokens = math.min(capacity, tokens + (delta / 1000) * refill_rate)
    last_refill = now
end

local allowed = 0
if tokens >= 1 then
    tokens = tokens - 1
    allowed = 1
end

redis.call("HMSET", KEYS[1], "tokens", tokens, "last_refill", last_refill)
redis.call("PEXPIRE", KEYS[1], math.ceil(capacity / refill_rate * 1000))
return allowed
