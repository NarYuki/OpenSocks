module("luci.controller.opensocks", package.seeall)

local http = luci.http
local uci = luci.model.uci.cursor()
local jsonc = luci.jsonc

function index()
    entry({"admin", "services", "opensocks"},
          template("opensocks/status"), _("OpenSocks"), 80).dependent = false
    entry({"admin", "services", "opensocks", "status"}, call("action_status")).leaf = true
    entry({"admin", "services", "opensocks", "lines"}, call("action_lines")).leaf = true
    entry({"admin", "services", "opensocks", "connect"}, call("action_connect")).leaf = true
    entry({"admin", "services", "opensocks", "disconnect"}, call("action_disconnect")).leaf = true
    entry({"admin", "services", "opensocks", "login"}, call("action_login")).leaf = true
    entry({"admin", "services", "opensocks", "register"}, call("action_register")).leaf = true
    entry({"admin", "services", "opensocks", "logout"}, call("action_logout")).leaf = true
    entry({"admin", "services", "opensocks", "settings"}, call("action_settings")).leaf = true
    entry({"admin", "services", "opensocks", "history"}, call("action_history")).leaf = true
    entry({"admin", "services", "opensocks", "reconnect"}, call("action_reconnect")).leaf = true
    entry({"admin", "services", "opensocks", "setup"}, call("action_setup")).leaf = true
    entry({"admin", "services", "opensocks", "traffic"}, call("action_traffic")).leaf = true
    entry({"admin", "services", "opensocks", "regiontest"}, call("action_regiontest")).leaf = true
    entry({"admin", "services", "opensocks", "speed_servers"}, call("action_speed_servers")).leaf = true
    entry({"admin", "services", "opensocks", "speed_run"}, call("action_speed_run")).leaf = true
    entry({"admin", "services", "opensocks", "speedcn_servers"}, call("action_speedcn_servers")).leaf = true
    entry({"admin", "services", "opensocks", "speedcn_run"}, call("action_speedcn_run")).leaf = true
    entry({"admin", "services", "opensocks", "mobile_pairing"}, call("action_mobile_pairing")).leaf = true
end

local function daemon_base()
    local port = tonumber(uci:get("opensocks", "settings", "control_port")) or 9091
    if port < 1 or port > 65535 then port = 9091 end
    return "http://127.0.0.1:" .. tostring(port)
end

local function shquote(s)
    return "'" .. tostring(s):gsub("'", "'\\''") .. "'"
end

local function stringify(v)
    if jsonc.stringify then
        return jsonc.stringify(v)
    end
    local json = require "luci.json"
    return json.encode(v)
end

-- Proxy a JSON request to the local daemon (server-side, avoids CORS).
local function daemon_request(method, path, payload, timeout)
    local cmd = "/usr/bin/curl -sS -m " .. tostring(timeout or 30) .. " -X " .. method
    if payload then
        cmd = cmd .. " -d " .. shquote(payload)
    end
    cmd = cmd .. " " .. shquote(daemon_base() .. path)
    local out = luci.sys.exec(cmd)
    local ok, parsed = pcall(jsonc.parse, out)
    if ok and parsed then
        return parsed
    end
    return { ok = false, error = (out and #out > 0 and out) or "daemon unreachable" }
end

local function write_json(data)
    http.prepare_content("application/json")
    http.write_json(data)
end

local function require_json_body()
    local body = http.content()
    if not body or body == "" then
        return nil
    end
    local ok, parsed = pcall(jsonc.parse, body)
    if not ok then
        return nil
    end
    return parsed
end

function action_mobile_pairing()
    write_json(daemon_request("GET", "/mobile/pairing", nil))
end

function action_setup()
    local req = require_json_body() or {}
    local mode = req.mode == "global" and "global" or "smart"
    write_json(daemon_request("POST", "/setup", stringify({ mode = mode }), 180))
end

function action_status()
    write_json(daemon_request("GET", "/status", nil))
end

function action_traffic()
    write_json(daemon_request("GET", "/traffic", nil))
end

function action_regiontest()
    write_json(daemon_request("GET", "/regiontest", nil, 30))
end

function action_speed_servers()
    write_json(daemon_request("GET", "/speedtest/servers", nil, 30))
end

function action_speed_run()
    local req = require_json_body() or {}
    write_json(daemon_request("POST", "/speedtest/run", stringify({ id = req.id }), 90))
end

function action_speedcn_servers()
    write_json(daemon_request("GET", "/speedtestcn/servers", nil, 30))
end

function action_speedcn_run()
    local req = require_json_body() or {}
    write_json(daemon_request("POST", "/speedtestcn/run", stringify({ id = req.id }), 90))
end

function action_lines()
    local sort = http.formvalue("sort")
    local path = "/lines"
    if sort == "ping" then path = path .. "?sort=ping" end
    write_json(daemon_request("GET", path, nil))
end

function action_history()
    write_json(daemon_request("GET", "/history", nil))
end

function action_reconnect()
    local req = require_json_body() or {}
    write_json(daemon_request("POST", "/reconnect", stringify({ id = req.id })))
end

function action_connect()
    local req = require_json_body() or {}
    local line_id = tonumber(req.line_id) or -1
    write_json(daemon_request("POST", "/connect", stringify({ line_id = line_id }), 90))
end

function action_disconnect()
    write_json(daemon_request("POST", "/disconnect", "{}"))
end

function action_login()
    local req = require_json_body() or {}
    local auth_by = req.auth_by
    local username = req.username
    local email = req.email
    if not auth_by then
        if email or (username and username:find("@", 1, true)) then
            auth_by = "email"
            email = email or username
            username = nil
        else
            auth_by = "username"
        end
    end
    local payload = {
        auth_by = auth_by,
        auth_type = req.auth_type or "password",
        username = username,
        password = req.password,
        email = email,
        phone = req.phone,
        sms_code = req.sms_code,
    }
    write_json(daemon_request("POST", "/login", stringify(payload)))
end

function action_register()
    write_json(daemon_request("POST", "/register", "{}"))
end

function action_logout()
    write_json(daemon_request("POST", "/logout", "{}"))
end

function action_settings()
    local req = require_json_body()
    if req then
        write_json(daemon_request("POST", "/settings", stringify({
            mode = req.mode,
            tun = req.tun,
            free_only = req.free_only,
            auto_connect = req.auto_connect,
            auto_route = req.auto_route,
            session_count = tonumber(req.session_count),
            region = req.region,
            exclude_regions = req.exclude_regions,
            include_domains = req.include_domains,
            exclude_domains = req.exclude_domains,
            include_cidrs = req.include_cidrs,
            exclude_cidrs = req.exclude_cidrs,
        })))
    else
        local data = daemon_request("GET", "/settings", nil)
        local pairing = daemon_request("GET", "/mobile/pairing", nil)
        if pairing and not pairing.error then
            data.mobileURL = pairing.url
            data.mobileToken = pairing.token
            data.mobileEnabled = pairing.enabled
        end
        write_json(data)
    end
end
