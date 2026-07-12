-- wrk stress testing script for Article CRUD endpoints
-- Run with: wrk -t2 -c10 -d10s -s stress.lua http://localhost:8123

math.randomseed(os.time())

local active_ids = {}

local base_headers = {
    ["Cookie"] = "environment=%7B%7D"
}

local json_headers = {
    ["Content-Type"] = "application/json",
    ["Cookie"] = "environment=%7B%7D"
}

local function random_string(length)
    local chars = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789'
    local res = {}
    for i = 1, length do
        local idx = math.random(1, #chars)
        res[i] = string.sub(chars, idx, idx)
    end
    return table.concat(res)
end

function request()
    -- Switch between 3 workflows randomly
    local workflow = math.random(1, 3)

    if workflow == 1 then
        -- Workflow 1: Create, Update random field, or Delete
        local action = math.random(1, 3)
        if action == 1 or #active_ids == 0 then
            -- Create
            local body = string.format('{"title":"W1_%s","content":"Content_%s"}', random_string(5), random_string(10))
            return wrk.format("POST", "/api/articles/", json_headers, body)
        elseif action == 2 then
            -- Update random (pick from the newest elements in the pool)
            local min_idx = math.max(1, #active_ids - 50)
            local idx = math.random(min_idx, #active_ids)
            local id = active_ids[idx]
            local body = string.format('{"title":"W1_UPD_%s","content":"Content_UPD_%s"}', random_string(5), random_string(10))
            return wrk.format("PUT", "/api/articles/" .. id .. "/", json_headers, body)
        else
            -- Delete (only delete the oldest elements if we have a safe buffer)
            if #active_ids > 100 then
                local id = table.remove(active_ids, 1)
                return wrk.format("DELETE", "/api/articles/" .. id .. "/", base_headers, nil)
            end
            -- Fallback to Create if buffer is not populated yet
            local body = string.format('{"title":"W1_%s","content":"Content_%s"}', random_string(5), random_string(10))
            return wrk.format("POST", "/api/articles/", json_headers, body)
        end

    elseif workflow == 2 then
        -- Workflow 2: Create multiple (10-20), list, or delete
        local action = math.random(1, 3)
        if action == 1 then
            -- Create multiple (returns one POST request)
            local body = string.format('{"title":"W2_%s","content":"Content_%s"}', random_string(5), random_string(10))
            return wrk.format("POST", "/api/articles/", json_headers, body)
        elseif action == 2 then
            -- List
            return wrk.format("GET", "/api/articles/", base_headers, nil)
        else
            -- Delete (only delete the oldest if we have a safe buffer)
            if #active_ids > 100 then
                local id = table.remove(active_ids, 1)
                return wrk.format("DELETE", "/api/articles/" .. id .. "/", base_headers, nil)
            end
            return wrk.format("GET", "/api/articles/", base_headers, nil)
        end

    else
        -- Workflow 3: Create many (100-200), list with random filter, or delete
        local action = math.random(1, 3)
        if action == 1 then
            -- Create
            local body = string.format('{"title":"W3_%s","content":"Content_%s"}', random_string(5), random_string(10))
            return wrk.format("POST", "/api/articles/", json_headers, body)
        elseif action == 2 then
            -- List with random filter
            local filter_char = string.char(math.random(97, 122)) -- random 'a'-'z'
            return wrk.format("GET", "/api/articles/?title=" .. filter_char, base_headers, nil)
        else
            -- Delete (only delete the oldest if we have a safe buffer)
            if #active_ids > 100 then
                local id = table.remove(active_ids, 1)
                return wrk.format("DELETE", "/api/articles/" .. id .. "/", base_headers, nil)
            end
            return wrk.format("GET", "/api/articles/", base_headers, nil)
        end
    end
end

function response(status, headers, body)
    if status == 201 then
        local id = string.match(body, '"id":%s*(%d+)')
        if id then
            table.insert(active_ids, tonumber(id))
        end
    end
end
