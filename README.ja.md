# OpenSocks for OpenWrt

<p align="center">
  <img src="assets/opensocks-icon-monochrome.png" alt="OpenSocks" width="192">
</p>

[中文](README.md) | [English](README.en.md) | [日本語](README.ja.md)

OpenSocksはTransocks（穿梭）の中国向け回国回線サービスをOpenWrtから利用するためのオープンソースクライアントで、LuCI管理画面とスマホ操作アプリを提供します。サーバー回線の利用にはTransocksアカウントが必要です。大規模な統合プロキシコアを使わず、`shadowsocks-libev`とnftablesで動作します。本プロジェクトは独立実装であり、Transocks公式クライアントではありません。

## 主な機能

- スマート：中国サービスはTransocks中国回線、X・YouTube・Google等は元のWAN回線
- フル中国回線：LAN/Wi-Fiの公開先TCP/UDPをすべてTransocks中国回線へ転送
- TCP REDIRECTとUDP TPROXY
- 選択サーバー固定、セッション更新、経路の自動復元
- Ping順サーバー一覧、接続履歴、再接続
- 中国サービス別通信量、リアルタイム速度、累積通信量
- 中国国内Ookla／SpeedTest.cn測定
- LuCI／Android／iOSの中国語・英語・日本語対応
- **容量の少ない端末向け:** 約1.8KBのminimalランチャーが実行ファイルを`/tmp`へ自動展開し、再起動後も自動復旧

## インストール

opkgリポジトリの利用を推奨します。

```sh
wget -O /etc/opkg/keys/d24a5e234001294c https://rel.n4t.su/opkg/d24a5e234001294c
echo 'src/gz opensocks https://rel.n4t.su/opkg' >> /etc/opkg/customfeeds.conf
opkg update
opkg install opensocks-minimal luci-app-opensocks
```

[GitHub Releases](https://github.com/NarYuki/OpenSocks/releases/latest)からIPKを直接ダウンロードしてインストールすることもできます。インストール、ログイン、機能、スマホ連携の詳しい説明:

- [日本語 Wiki](https://github.com/NarYuki/OpenSocks/wiki/Home-ja)
- [English Wiki](https://github.com/NarYuki/OpenSocks/wiki/Home-en)
- [中文 Wiki](https://github.com/NarYuki/OpenSocks/wiki)

スマホアプリは、Android版を[最新Release](https://github.com/NarYuki/OpenSocks/releases/latest)から、iOS版を[TestFlight](https://testflight.apple.com/join/eT82PtM1)からインストールできます。

## 開発時の確認

```sh
cd openwrt/opensocks/src
go test ./...
go vet ./...

cd ../../../mobile
flutter analyze
flutter test
```

ライセンスはGPL-3.0-or-laterです。詳細は[LICENSE](LICENSE)を参照してください。
