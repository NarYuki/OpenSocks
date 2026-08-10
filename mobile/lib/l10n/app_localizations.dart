import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:intl/intl.dart' as intl;

import 'app_localizations_en.dart';
import 'app_localizations_ja.dart';
import 'app_localizations_zh.dart';

// ignore_for_file: type=lint

/// Callers can lookup localized strings with an instance of AppLocalizations
/// returned by `AppLocalizations.of(context)`.
///
/// Applications need to include `AppLocalizations.delegate()` in their app's
/// `localizationDelegates` list, and the locales they support in the app's
/// `supportedLocales` list. For example:
///
/// ```dart
/// import 'l10n/app_localizations.dart';
///
/// return MaterialApp(
///   localizationsDelegates: AppLocalizations.localizationsDelegates,
///   supportedLocales: AppLocalizations.supportedLocales,
///   home: MyApplicationHome(),
/// );
/// ```
///
/// ## Update pubspec.yaml
///
/// Please make sure to update your pubspec.yaml to include the following
/// packages:
///
/// ```yaml
/// dependencies:
///   # Internationalization support.
///   flutter_localizations:
///     sdk: flutter
///   intl: any # Use the pinned version from flutter_localizations
///
///   # Rest of dependencies
/// ```
///
/// ## iOS Applications
///
/// iOS applications define key application metadata, including supported
/// locales, in an Info.plist file that is built into the application bundle.
/// To configure the locales supported by your app, you’ll need to edit this
/// file.
///
/// First, open your project’s ios/Runner.xcworkspace Xcode workspace file.
/// Then, in the Project Navigator, open the Info.plist file under the Runner
/// project’s Runner folder.
///
/// Next, select the Information Property List item, select Add Item from the
/// Editor menu, then select Localizations from the pop-up menu.
///
/// Select and expand the newly-created Localizations item then, for each
/// locale your application supports, add a new item and select the locale
/// you wish to add from the pop-up menu in the Value field. This list should
/// be consistent with the languages listed in the AppLocalizations.supportedLocales
/// property.
abstract class AppLocalizations {
  AppLocalizations(String locale)
    : localeName = intl.Intl.canonicalizedLocale(locale.toString());

  final String localeName;

  static AppLocalizations of(BuildContext context) {
    return Localizations.of<AppLocalizations>(context, AppLocalizations)!;
  }

  static const LocalizationsDelegate<AppLocalizations> delegate =
      _AppLocalizationsDelegate();

  /// A list of this localizations delegate along with the default localizations
  /// delegates.
  ///
  /// Returns a list of localizations delegates containing this delegate along with
  /// GlobalMaterialLocalizations.delegate, GlobalCupertinoLocalizations.delegate,
  /// and GlobalWidgetsLocalizations.delegate.
  ///
  /// Additional delegates can be added by appending to this list in
  /// MaterialApp. This list does not have to be used at all if a custom list
  /// of delegates is preferred or required.
  static const List<LocalizationsDelegate<dynamic>> localizationsDelegates =
      <LocalizationsDelegate<dynamic>>[
        delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
      ];

  /// A list of this localizations delegate's supported locales.
  static const List<Locale> supportedLocales = <Locale>[
    Locale('en'),
    Locale('ja'),
    Locale('zh'),
  ];

  /// No description provided for @pairHelp.
  ///
  /// In ja, this message translates to:
  /// **'LuCIの「スマホ連携」に表示されたURLとトークンを入力してください。'**
  String get pairHelp;

  /// No description provided for @routerUrl.
  ///
  /// In ja, this message translates to:
  /// **'ルーターURL'**
  String get routerUrl;

  /// No description provided for @pairToken.
  ///
  /// In ja, this message translates to:
  /// **'連携トークン'**
  String get pairToken;

  /// No description provided for @connect.
  ///
  /// In ja, this message translates to:
  /// **'接続'**
  String get connect;

  /// No description provided for @traffic.
  ///
  /// In ja, this message translates to:
  /// **'通信'**
  String get traffic;

  /// No description provided for @test.
  ///
  /// In ja, this message translates to:
  /// **'テスト'**
  String get test;

  /// No description provided for @account.
  ///
  /// In ja, this message translates to:
  /// **'アカウント'**
  String get account;

  /// No description provided for @authenticating.
  ///
  /// In ja, this message translates to:
  /// **'認証中'**
  String get authenticating;

  /// No description provided for @switching.
  ///
  /// In ja, this message translates to:
  /// **'切替中'**
  String get switching;

  /// No description provided for @connecting.
  ///
  /// In ja, this message translates to:
  /// **'接続中'**
  String get connecting;

  /// No description provided for @networkConfig.
  ///
  /// In ja, this message translates to:
  /// **'ネットワーク構成中'**
  String get networkConfig;

  /// No description provided for @interfaceProcessing.
  ///
  /// In ja, this message translates to:
  /// **'インターフェース処理中'**
  String get interfaceProcessing;

  /// No description provided for @disconnecting.
  ///
  /// In ja, this message translates to:
  /// **'切断処理中'**
  String get disconnecting;

  /// No description provided for @connected.
  ///
  /// In ja, this message translates to:
  /// **'接続完了'**
  String get connected;

  /// No description provided for @disconnected.
  ///
  /// In ja, this message translates to:
  /// **'切断完了'**
  String get disconnected;

  /// No description provided for @routingMode.
  ///
  /// In ja, this message translates to:
  /// **'ルーティングモード'**
  String get routingMode;

  /// No description provided for @smartRouting.
  ///
  /// In ja, this message translates to:
  /// **'スマートルーティング'**
  String get smartRouting;

  /// No description provided for @smartDescription.
  ///
  /// In ja, this message translates to:
  /// **'中国向け通信だけをOpenSocksへ送信'**
  String get smartDescription;

  /// No description provided for @globalRouting.
  ///
  /// In ja, this message translates to:
  /// **'フル中国回線ルーティング'**
  String get globalRouting;

  /// No description provided for @globalDescription.
  ///
  /// In ja, this message translates to:
  /// **'LANのWeb通信全体をOpenSocksへ送信'**
  String get globalDescription;

  /// No description provided for @changingMode.
  ///
  /// In ja, this message translates to:
  /// **'ルーティングモード変更中'**
  String get changingMode;

  /// No description provided for @modeChanged.
  ///
  /// In ja, this message translates to:
  /// **'ルーティングモードを変更しました'**
  String get modeChanged;

  /// No description provided for @protected.
  ///
  /// In ja, this message translates to:
  /// **'回国ing'**
  String get protected;

  /// No description provided for @notConnected.
  ///
  /// In ja, this message translates to:
  /// **'未接続'**
  String get notConnected;

  /// No description provided for @tapToConnect.
  ///
  /// In ja, this message translates to:
  /// **'タップして接続'**
  String get tapToConnect;

  /// No description provided for @selectServer.
  ///
  /// In ja, this message translates to:
  /// **'サーバーを選択'**
  String get selectServer;

  /// No description provided for @routeStopped.
  ///
  /// In ja, this message translates to:
  /// **'ネットワーク経路は停止中です'**
  String get routeStopped;

  /// No description provided for @routeHealthy.
  ///
  /// In ja, this message translates to:
  /// **'ネットワーク経路は正常です'**
  String get routeHealthy;

  /// No description provided for @routeRestoring.
  ///
  /// In ja, this message translates to:
  /// **'ネットワーク経路を復元中'**
  String get routeRestoring;

  /// No description provided for @notSignedIn.
  ///
  /// In ja, this message translates to:
  /// **'未ログイン'**
  String get notSignedIn;

  /// No description provided for @daysLeft.
  ///
  /// In ja, this message translates to:
  /// **'残り{days}日'**
  String daysLeft(Object days);

  /// No description provided for @serversCount.
  ///
  /// In ja, this message translates to:
  /// **'サーバー ({count})'**
  String serversCount(int count);

  /// No description provided for @historyCount.
  ///
  /// In ja, this message translates to:
  /// **'履歴 ({count})'**
  String historyCount(int count);

  /// No description provided for @serverList.
  ///
  /// In ja, this message translates to:
  /// **'サーバー一覧'**
  String get serverList;

  /// No description provided for @measuringAllPing.
  ///
  /// In ja, this message translates to:
  /// **'全サーバーのPingを測定しています…'**
  String get measuringAllPing;

  /// No description provided for @serversUnavailable.
  ///
  /// In ja, this message translates to:
  /// **'サーバーを取得できませんでした'**
  String get serversUnavailable;

  /// No description provided for @reconnected.
  ///
  /// In ja, this message translates to:
  /// **'再接続しました'**
  String get reconnected;

  /// No description provided for @chinaServiceTraffic.
  ///
  /// In ja, this message translates to:
  /// **'中国サービス別トラフィック'**
  String get chinaServiceTraffic;

  /// No description provided for @cumulativeTraffic.
  ///
  /// In ja, this message translates to:
  /// **'累積 ↑ {up} / ↓ {down} / Σ {total}'**
  String cumulativeTraffic(Object up, Object down, Object total);

  /// No description provided for @chinaRouteTest.
  ///
  /// In ja, this message translates to:
  /// **'中国経路・IP健全度テスト'**
  String get chinaRouteTest;

  /// No description provided for @health.
  ///
  /// In ja, this message translates to:
  /// **'健全度'**
  String get health;

  /// No description provided for @tapToTest.
  ///
  /// In ja, this message translates to:
  /// **'タップして測定'**
  String get tapToTest;

  /// No description provided for @notSelected.
  ///
  /// In ja, this message translates to:
  /// **'未選択'**
  String get notSelected;

  /// No description provided for @changeServer.
  ///
  /// In ja, this message translates to:
  /// **'サーバー変更'**
  String get changeServer;

  /// No description provided for @searchTestNode.
  ///
  /// In ja, this message translates to:
  /// **'測定ノード名・地域・事業者を検索'**
  String get searchTestNode;

  /// No description provided for @serverFetchError.
  ///
  /// In ja, this message translates to:
  /// **'サーバー取得エラー: {error}'**
  String serverFetchError(Object error);

  /// No description provided for @retry.
  ///
  /// In ja, this message translates to:
  /// **'再試行'**
  String get retry;

  /// No description provided for @trafficErrorTitle.
  ///
  /// In ja, this message translates to:
  /// **'通信量を取得できませんでした'**
  String get trafficErrorTitle;

  /// No description provided for @trafficErrorMessage.
  ///
  /// In ja, this message translates to:
  /// **'ルーターとの通信を確認して、もう一度読み込んでください。'**
  String get trafficErrorMessage;

  /// No description provided for @chinaRouteErrorTitle.
  ///
  /// In ja, this message translates to:
  /// **'中国経路テストに失敗しました'**
  String get chinaRouteErrorTitle;

  /// No description provided for @chinaRouteErrorMessage.
  ///
  /// In ja, this message translates to:
  /// **'中国回線への接続状態を確認して、もう一度テストしてください。'**
  String get chinaRouteErrorMessage;

  /// No description provided for @speedtestCnErrorTitle.
  ///
  /// In ja, this message translates to:
  /// **'SpeedTest.cnサーバーを取得できませんでした'**
  String get speedtestCnErrorTitle;

  /// No description provided for @speedtestCnErrorMessage.
  ///
  /// In ja, this message translates to:
  /// **'SpeedTest.cnの中国ノードAPIへ接続できませんでした。時間を置いて再試行してください。'**
  String get speedtestCnErrorMessage;

  /// No description provided for @ooklaErrorTitle.
  ///
  /// In ja, this message translates to:
  /// **'Ooklaサーバーを取得できませんでした'**
  String get ooklaErrorTitle;

  /// No description provided for @ooklaErrorMessage.
  ///
  /// In ja, this message translates to:
  /// **'Ooklaの中国サーバー一覧へ接続できませんでした。時間を置いて再試行してください。'**
  String get ooklaErrorMessage;

  /// No description provided for @routerConnectionError.
  ///
  /// In ja, this message translates to:
  /// **'ルーターへ接続できません。Wi-FiまたはTailscale接続を確認してください。'**
  String get routerConnectionError;

  /// No description provided for @routerTimeoutError.
  ///
  /// In ja, this message translates to:
  /// **'応答が時間内に返りませんでした。'**
  String get routerTimeoutError;

  /// No description provided for @invalidResponseError.
  ///
  /// In ja, this message translates to:
  /// **'ルーターから正しい応答を受信できませんでした。'**
  String get invalidResponseError;

  /// No description provided for @serverResponseError.
  ///
  /// In ja, this message translates to:
  /// **'ルーター側の処理に失敗しました。'**
  String get serverResponseError;

  /// No description provided for @completed.
  ///
  /// In ja, this message translates to:
  /// **'完了しました'**
  String get completed;

  /// No description provided for @email.
  ///
  /// In ja, this message translates to:
  /// **'メール'**
  String get email;

  /// No description provided for @phone.
  ///
  /// In ja, this message translates to:
  /// **'電話番号'**
  String get phone;

  /// No description provided for @expires.
  ///
  /// In ja, this message translates to:
  /// **'有効期限'**
  String get expires;

  /// No description provided for @remainingDays.
  ///
  /// In ja, this message translates to:
  /// **'残り日数'**
  String get remainingDays;

  /// No description provided for @credentials.
  ///
  /// In ja, this message translates to:
  /// **'認証情報'**
  String get credentials;

  /// No description provided for @saved.
  ///
  /// In ja, this message translates to:
  /// **'保存済み'**
  String get saved;

  /// No description provided for @notSaved.
  ///
  /// In ja, this message translates to:
  /// **'未保存'**
  String get notSaved;

  /// No description provided for @loginManagement.
  ///
  /// In ja, this message translates to:
  /// **'ログイン管理'**
  String get loginManagement;

  /// No description provided for @username.
  ///
  /// In ja, this message translates to:
  /// **'メールアドレス / ユーザー名'**
  String get username;

  /// No description provided for @password.
  ///
  /// In ja, this message translates to:
  /// **'パスワード'**
  String get password;

  /// No description provided for @login.
  ///
  /// In ja, this message translates to:
  /// **'ログイン'**
  String get login;

  /// No description provided for @anonymousRegister.
  ///
  /// In ja, this message translates to:
  /// **'匿名無料登録'**
  String get anonymousRegister;

  /// No description provided for @logout.
  ///
  /// In ja, this message translates to:
  /// **'ログアウト'**
  String get logout;

  /// No description provided for @freeAccount.
  ///
  /// In ja, this message translates to:
  /// **'FREE ACCOUNT'**
  String get freeAccount;

  /// No description provided for @vipAccount.
  ///
  /// In ja, this message translates to:
  /// **'VIP ACCOUNT'**
  String get vipAccount;
}

class _AppLocalizationsDelegate
    extends LocalizationsDelegate<AppLocalizations> {
  const _AppLocalizationsDelegate();

  @override
  Future<AppLocalizations> load(Locale locale) {
    return SynchronousFuture<AppLocalizations>(lookupAppLocalizations(locale));
  }

  @override
  bool isSupported(Locale locale) =>
      <String>['en', 'ja', 'zh'].contains(locale.languageCode);

  @override
  bool shouldReload(_AppLocalizationsDelegate old) => false;
}

AppLocalizations lookupAppLocalizations(Locale locale) {
  // Lookup logic when only language code is specified.
  switch (locale.languageCode) {
    case 'en':
      return AppLocalizationsEn();
    case 'ja':
      return AppLocalizationsJa();
    case 'zh':
      return AppLocalizationsZh();
  }

  throw FlutterError(
    'AppLocalizations.delegate failed to load unsupported locale "$locale". This is likely '
    'an issue with the localizations generation tool. Please file an issue '
    'on GitHub with a reproducible sample app and the gen-l10n configuration '
    'that was used.',
  );
}
