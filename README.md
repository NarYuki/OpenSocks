# OpenSocks for OpenWrt

OpenSocks is a low-memory OpenWrt client and LuCI dashboard for China-route
proxy connections. It uses `shadowsocks-libev` and nftables instead of a large
all-in-one proxy core, making it suitable for memory-constrained routers.

The Flutter Android/iOS controller is available in [`mobile/`](mobile/). It
provides the same router controls through a token-authenticated mobile API,
with a VPN-style power button and a 100 ms live speed-test display.

## Features

- Smart China routing and an all-TCP China proxy mode
- One-tap OpenWrt network integration
- Persistent server selection, connection history, and reconnect actions
- Saved credentials with automatic sign-in after session expiry
- VIP/Free server filtering and latency sorting
- Automatic routing restoration after router or daemon restarts
- Per-service China traffic accounting with one-second LuCI updates
- Passive DNS suffix learning for service subdomains and CDN hosts
- Ookla China-server and SpeedTest.cn measurements from LuCI
- English, Japanese, and Simplified Chinese interface support
- Low-memory `ss-redir` data path with nftables interval sets

## Repository layout

```text
openwrt/
├── opensocks/           Go daemon and OpenWrt service package
└── luci-app-opensocks/  LuCI controller, dashboard, ACL, and translations
```

## Build

Copy both package directories into an OpenWrt buildroot or add this repository
as a package feed, then run:

```sh
./scripts/feeds update -a
./scripts/feeds install -a
make package/opensocks/compile V=s
make package/luci-app-opensocks/compile V=s
```

The daemon can also be tested on a development host:

```sh
cd openwrt/opensocks/src
go test ./...
go build .
```

## Runtime dependencies

- `shadowsocks-libev-ss-redir`
- `shadowsocks-libev-ss-local`
- `nftables-json`
- `ca-bundle`
- `uci`
- `luci-base`, `luci-compat`, and `curl` for the LuCI package

After installing both generated packages:

```sh
/etc/init.d/opensocks enable
/etc/init.d/opensocks start
```

Open `https://<router>/cgi-bin/luci/admin/services/opensocks`. Configure LuCI
HTTPS before entering account credentials.

Credentials and sessions are created only at runtime on the router and are not
part of this repository. Do not commit `/etc/opensocks`, runtime UCI exports,
logs, or captured API responses.

Additional OpenWrt-specific details are in
[openwrt/README.md](openwrt/README.md).

## License

GPL-3.0-or-later. See [LICENSE](LICENSE).
