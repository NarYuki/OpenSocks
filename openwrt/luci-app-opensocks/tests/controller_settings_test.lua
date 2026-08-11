local request = {
    mode = "global",
    tun = false,
    free_only = true,
    auto_connect = true,
    auto_route = true,
    session_count = 3,
    slot2_line_id = 321,
    slot3_line_id = 654,
    region = "",
    exclude_regions = "",
    include_domains = "",
    exclude_domains = "",
    include_cidrs = "",
    exclude_cidrs = "",
}

local forwarded
local response

package.seeall = package.seeall or {}
module = module or function() end
luci = {
    http = {
        content = function() return "request" end,
        prepare_content = function() end,
        write_json = function(value) response = value end,
        formvalue = function() return nil end,
    },
    model = { uci = { cursor = function()
        return { get = function() return nil end }
    end } },
    jsonc = {
        parse = function(value)
            if value == "request" then return request end
            if value == "daemon-response" then return { ok = true } end
            return nil
        end,
        stringify = function(value)
            forwarded = value
            return '{"session_count":3}'
        end,
    },
    sys = { exec = function(command)
        assert(command:find('session_count', 1, true), "serialized settings were not forwarded")
        return "daemon-response"
    end },
}

dofile("openwrt/luci-app-opensocks/luasrc/controller/opensocks.lua")
action_settings()

assert(forwarded, "settings payload was not serialized")
assert(forwarded.mode == "global", "routing mode was not forwarded")
assert(forwarded.session_count == 3, "session count was not forwarded")
assert(forwarded.slot2_line_id == 321, "session 2 server was not forwarded")
assert(forwarded.slot3_line_id == 654, "session 3 server was not forwarded")
assert(response and response.ok == true, "daemon response was not returned")
print("PASS LuCI settings forwarding")
