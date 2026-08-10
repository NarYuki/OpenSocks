# OpenSocks for OpenWrt

<p align="center">
  <img src="assets/opensocks-icon-monochrome.png" alt="OpenSocks" width="192">
</p>

[中文](README.md) | [English](README.en.md) | [日本語](README.ja.md)

OpenSocks is an open-source OpenWrt client for the Transocks China return-route service, with a LuCI dashboard and mobile controller. A Transocks account is required to use its server lines. It uses `shadowsocks-libev` and nftables instead of a large all-in-one proxy core. This is an independent implementation, not an official Transocks client.

## Highlights

- Smart routing sends Chinese services through the Transocks China line while X, YouTube, Google, and other non-Chinese services use the original WAN
- Full China routing sends all public TCP/UDP from LAN and Wi-Fi through the Transocks China line
- TCP REDIRECT and UDP TPROXY
- Persistent server selection, session recovery, and automatic route restoration
- Latency-sorted servers, connection history, and reconnect actions
- Per-service China traffic, live rates, and persistent totals
- Ookla China and SpeedTest.cn testing
- Chinese, English, and Japanese LuCI/mobile interfaces
- **Low-storage option:** an approximately 1.8KB minimal launcher expands the daemon into `/tmp` and restores it automatically after reboots

## Installation

Using the opkg repository is recommended:

```sh
wget -O /etc/opkg/keys/d24a5e234001294c https://rel.n4t.su/opkg/d24a5e234001294c
echo 'src/gz opensocks https://rel.n4t.su/opkg' >> /etc/opkg/customfeeds.conf
opkg update
opkg install opensocks-minimal luci-app-opensocks
```

You can also download IPKs from [GitHub Releases](https://github.com/NarYuki/OpenSocks/releases/latest) and install them directly. Full installation, login, feature, and mobile-pairing documentation:

- [English Wiki](https://github.com/NarYuki/OpenSocks/wiki/Home-en)
- [中文 Wiki](https://github.com/NarYuki/OpenSocks/wiki)
- [日本語 Wiki](https://github.com/NarYuki/OpenSocks/wiki/Home-ja)

Mobile app: download the Android APK from the [latest release](https://github.com/NarYuki/OpenSocks/releases/latest), or install the iOS app through [TestFlight](https://testflight.apple.com/join/eT82PtM1).

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
