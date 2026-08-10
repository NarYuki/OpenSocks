# OpenWrt 用パッケージ

「穿梭 Transocks」のコア仕様に基づく OpenWrt ルーター向け VPN パッケージ + LuCI 管理 UI。

```
openwrt/
├── opensocks/          デーモン + 軽量 ss-redir/nftables エンジン
└── luci-app-opensocks/ LuCI Web ダッシュ (ステータス・ログイン・Free回線選択)
```

## 機能

- **ログイン**: メール/電話/ユーザー名 + パスワード。セッション切れ時は保存済み認証情報で自動再ログイン
- **Free サーバー**: ログイン不要のデバイス登録 (`registerByDevice`) で無料回線へ接続可能。
  `free_only` 設定で無料回線のみ表示
- **エンジン**: 約100KBの `shadowsocks-libev-ss-redir`。mihomoは使用しない
- **モード**: smart (中国CIDR→PROXY / 他→DIRECT) または全TCP通信の中国プロキシ経由
- **Web ダッシュ操作**: 接続、ping順ソート、履歴確認、履歴からの再接続
- **ワンタップ構成**: LANブリッジを自動検出し、smartまたは全TCP通信のプロキシ経由を即時構成
- **LuCI表示言語**: 英語（既定）、日本語、簡体中国語
- **自動接続**: ブート時にサーバー推奨回線へ自動接続 (auto_connect)

## ビルド方法

### 1. OpenWrt SDK / buildroot を用意

```bash
./scripts/feeds update -a && ./scripts/feeds install -a

# パッケージを feeds として追加するか、直接配置:
#   package/opensocks/       ← openwrt/opensocks をコピー
#   package/luci-app-opensocks/ ← openwrt/luci-app-opensocks をコピー
./scripts/feeds install opensocks luci-app-opensocks  # feeds.conf 経由の場合
make package/opensocks/compile
make package/luci-app-opensocks/compile
```

`luci-app-opensocks` は `luci` (feeds/luci) が必須。

### 2. デーモン単体のビルド (動作確認用)

```bash
cd opensocks/src
CGO_ENABLED=0 go build -o opensocks .   # ホスト環境でビルド
```

## インストール後

```bash
opkg update && opkg install shadowsocks-libev-ss-redir nftables-json opensocks luci-app-opensocks
/etc/init.d/opensocks enable
/etc/init.d/opensocks start
```

LuCI HTTPSを設定してから、ブラウザで
`https://<router>/cgi-bin/luci/admin/services/opensocks` を開く。

### 手動設定 (/etc/config/opensocks)

```
config settings
    option mode 'smart'         # smart | global
    option tun '0'              # 軽量版では常にnftables TCPリダイレクト
    option free_only '1'        # 無料回線のみ表示・自動接続
    option auto_connect '1'     # ブート時自動接続
    option region ''            # リージョンIDフィルタ (任意)
    option api_domain 'https://abscf2.fobwifi.com'
    option control_port '9091'  # ローカル制御API
```

## FAQ

### Free サーバーはどう動く?

`GET /api/2/line` が返す回線リストに `isFree=true` の無料回線が含まれます (auto-free 回線は
id=-1)。無料回線はアカウント不要で、端末登録 API (`/api/1/app/user/register` device フロー) で
取得したトークンだけで接続できます。VIP 回線はログインが必要です。

### なぜ mihomo を使わない?

RAM 120MBの実機ではmihomoがルーター全体を不安定化したためです。現在はAPIが返すSS接続に必要な
`ss-redir`だけを使い、中国CIDRリストをnftablesのinterval setへ読み込みます。実機RSSは
OpenSocks約11.6MB + ss-redir約1.4MBでした。TCPはREDIRECT、UDPはTPROXYで中国経路へ送り、
王者荣耀のPVP通信も含めて処理します。Trojan/GTS/TUNには未対応です。

容量が限られる機種向けには`opensocks-minimal`も用意しています。このIPKはランチャーだけを
overlayへ保存し、署名済みリリースの`opensocks-linux-mipsle.gz`をSHA-256検証して`/tmp`へ展開します。
実行ファイルは再起動時に再取得しますが、UCI設定は`/etc/config/opensocks`、暗号化済み認証情報・
履歴・通信統計は`/etc/opensocks`に残るため失われません。

### ルーターを再起動すると?

`auto_connect` が有効なら、デーモンが起動時に保存済みトークンで Free 回線へ自動接続します。

## ライセンス

GPL-3.0 (mihomo / clash エコシステムとの整合性のため)。
