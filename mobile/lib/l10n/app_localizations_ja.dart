// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Japanese (`ja`).
class AppLocalizationsJa extends AppLocalizations {
  AppLocalizationsJa([String locale = 'ja']) : super(locale);

  @override
  String get pairHelp => 'LuCIの「スマホ連携」に表示されたURLとトークンを入力してください。';

  @override
  String get routerUrl => 'ルーターURL';

  @override
  String get pairToken => '連携トークン';

  @override
  String get connect => '接続';

  @override
  String get traffic => '通信';

  @override
  String get test => 'テスト';

  @override
  String get account => 'アカウント';

  @override
  String get authenticating => '認証中';

  @override
  String get switching => '切替中';

  @override
  String get connecting => '接続中';

  @override
  String get networkConfig => 'ネットワーク構成中';

  @override
  String get interfaceProcessing => 'インターフェース処理中';

  @override
  String get disconnecting => '切断処理中';

  @override
  String get connected => '接続完了';

  @override
  String get disconnected => '切断完了';

  @override
  String get routingMode => 'ルーティングモード';

  @override
  String get smartRouting => 'スマートルーティング';

  @override
  String get smartDescription => '中国向け通信だけをOpenSocksへ送信';

  @override
  String get globalRouting => 'フル中国回線ルーティング';

  @override
  String get globalDescription => 'LANのWeb通信全体をOpenSocksへ送信';

  @override
  String get changingMode => 'ルーティングモード変更中';

  @override
  String get modeChanged => 'ルーティングモードを変更しました';

  @override
  String get protected => '回国ing';

  @override
  String get notConnected => '未接続';

  @override
  String get tapToConnect => 'タップして接続';

  @override
  String get selectServer => 'サーバーを選択';

  @override
  String get routeStopped => 'ネットワーク経路は停止中です';

  @override
  String get routeHealthy => 'ネットワーク経路は正常です';

  @override
  String get routeRestoring => 'ネットワーク経路を復元中';

  @override
  String get notSignedIn => '未ログイン';

  @override
  String daysLeft(Object days) {
    return '残り$days日';
  }

  @override
  String serversCount(int count) {
    return 'サーバー ($count)';
  }

  @override
  String historyCount(int count) {
    return '履歴 ($count)';
  }

  @override
  String get serverList => 'サーバー一覧';

  @override
  String get measuringAllPing => '全サーバーのPingを測定しています…';

  @override
  String get serversUnavailable => 'サーバーを取得できませんでした';

  @override
  String get reconnected => '再接続しました';

  @override
  String get chinaServiceTraffic => '中国サービス別トラフィック';

  @override
  String cumulativeTraffic(Object up, Object down, Object total) {
    return '累積 ↑ $up / ↓ $down / Σ $total';
  }

  @override
  String get chinaRouteTest => '中国経路・IP健全度テスト';

  @override
  String get health => '健全度';

  @override
  String get tapToTest => 'タップして測定';

  @override
  String get notSelected => '未選択';

  @override
  String get changeServer => 'サーバー変更';

  @override
  String get searchTestNode => '測定ノード名・地域・事業者を検索';

  @override
  String serverFetchError(Object error) {
    return 'サーバー取得エラー: $error';
  }

  @override
  String get completed => '完了しました';

  @override
  String get email => 'メール';

  @override
  String get phone => '電話番号';

  @override
  String get expires => '有効期限';

  @override
  String get remainingDays => '残り日数';

  @override
  String get credentials => '認証情報';

  @override
  String get saved => '保存済み';

  @override
  String get notSaved => '未保存';

  @override
  String get loginManagement => 'ログイン管理';

  @override
  String get username => 'メールアドレス / ユーザー名';

  @override
  String get password => 'パスワード';

  @override
  String get login => 'ログイン';

  @override
  String get anonymousRegister => '匿名無料登録';

  @override
  String get logout => 'ログアウト';

  @override
  String get freeAccount => 'FREE ACCOUNT';

  @override
  String get vipAccount => 'VIP ACCOUNT';
}
