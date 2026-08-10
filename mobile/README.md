# OpenSocks Mobile

Flutter製のAndroid/iOS管理アプリです。

- VPNアプリ型の円形電源ボタンによる接続・切断
- サーバー一覧、全サーバーPing測定、切替、履歴からの再接続
- 中国向けリアルタイム速度、累積通信量、サービス別ランキング
- 出口地域、通信事業者、IP、ASN、IP健全度テスト
- Ookla／SpeedTest.cn統合速度測定（100msリアルタイムUI）
- アカウント操作

## ペアリング

LuCIの「OpenSocks → スマホアプリ」で「連携情報を表示」を押し、表示されたルーターURLと連携トークンをアプリへ入力します。

外出先から利用する場合は、先にTailscaleなどでルーターへ接続してください。

## ビルド

```sh
flutter pub get
flutter analyze
flutter test
flutter build apk --release
```
