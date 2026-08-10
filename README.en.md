# OpenSocks for OpenWrt

[中文](README.md) | [English](README.en.md) | [日本語](README.ja.md)

OpenSocks is a low-memory OpenWrt client, LuCI dashboard, and mobile controller for China-route proxy connections. It uses `shadowsocks-libev` and nftables instead of a large all-in-one proxy core.

## Highlights

- Smart China routing and full China-line routing
- TCP REDIRECT and UDP TPROXY, including Honor of Kings PVP traffic
- Persistent server selection, session recovery, and automatic route restoration
- AES-256-GCM encrypted credential storage
- Latency-sorted servers, connection history, and reconnect actions
- Per-service China traffic, live rates, and persistent totals
- Ookla China and SpeedTest.cn testing
- Chinese, English, and Japanese LuCI/mobile interfaces
- Approximately 1.8KB minimal IPK; the binary runs from `/tmp` while settings remain under `/etc`

## Installation

Using the opkg repository is recommended:

```sh
echo 'src/gz opensocks https://rel.n4t.su/opkg' >> /etc/opkg/customfeeds.conf
opkg update
opkg install opensocks-minimal luci-app-opensocks
```

You can also download IPKs from [GitHub Releases](https://github.com/NarYuki/OpenSocks/releases/latest) and install them directly. Full installation, login, feature, and mobile-pairing documentation:

- [English Wiki](https://github.com/NarYuki/OpenSocks/wiki/Home-en)
- [中文 Wiki](https://github.com/NarYuki/OpenSocks/wiki)
- [日本語 Wiki](https://github.com/NarYuki/OpenSocks/wiki/Home-ja)

## Development checks

```sh
cd openwrt/opensocks/src
go test ./...
go vet ./...

cd ../../../mobile
flutter analyze
flutter test
```

Licensed under GPL-3.0-or-later. See [LICENSE](LICENSE).
