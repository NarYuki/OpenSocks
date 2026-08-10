# OpenSocks for OpenWrt

[中文](README.md) | [English](README.en.md) | [日本語](README.ja.md)

OpenSocksは中国向け回線をOpenWrtから利用するための省メモリデーモン、LuCI管理画面、スマホクライアントです。大規模な統合プロキシコアを使わず、`shadowsocks-libev`とnftablesで動作します。

## 主な機能

- スマート：中国サービスはTransocks中国回線、X・YouTube・Google等は元のWAN回線
- フル中国回線：LAN/Wi-Fiの公開先TCP/UDPをすべてTransocks中国回線へ転送
- TCP REDIRECTとUDP TPROXY（王者栄耀のPVP UDP通信を含む）
- 選択サーバー固定、セッション更新、経路の自動復元
- AES-256-GCMによる認証情報の暗号化保存
- Ping順サーバー一覧、接続履歴、再接続
- 中国サービス別通信量、リアルタイム速度、累積通信量
- 中国国内Ookla／SpeedTest.cn測定
- LuCI／Android／iOSの中国語・英語・日本語対応
- 約1.8KBのminimal IPK。バイナリは`/tmp`、設定と状態は`/etc`に保存

## インストール

opkgリポジトリの利用を推奨します。

```sh
echo 'src/gz opensocks https://rel.n4t.su/opkg' >> /etc/opkg/customfeeds.conf
opkg update
opkg install opensocks-minimal luci-app-opensocks
```

[GitHub Releases](https://github.com/NarYuki/OpenSocks/releases/latest)からIPKを直接ダウンロードしてインストールすることもできます。インストール、ログイン、機能、スマホ連携の詳しい説明:

- [日本語 Wiki](https://github.com/NarYuki/OpenSocks/wiki/Home-ja)
- [English Wiki](https://github.com/NarYuki/OpenSocks/wiki/Home-en)
- [中文 Wiki](https://github.com/NarYuki/OpenSocks/wiki)

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
