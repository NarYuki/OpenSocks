import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:opensocks_mobile/main.dart';
import 'package:opensocks_mobile/api.dart';
import 'package:opensocks_mobile/l10n/app_localizations.dart';
import 'package:opensocks_mobile/l10n/app_localizations_en.dart';
import 'package:opensocks_mobile/l10n/app_localizations_zh.dart';

class _FakeApi extends OpenSocksApi {
  _FakeApi() : super('http://127.0.0.1', 'test');

  final events = <String>[];
  bool failSettings = false;
  final status = <String, dynamic>{
    'running': true,
    'routingApplied': true,
    'mode': 'smart',
    'sessionCount': 2,
    'lineName': 'Test China server',
  };

  @override
  Future<Map<String, dynamic>> get(String path, {int timeout = 35}) async {
    if (path == 'status') return Map<String, dynamic>.from(status);
    if (path == 'traffic') {
      return {'up_bps': 0, 'down_bps': 0, 'up_bytes': 0, 'down_bytes': 0};
    }
    return <String, dynamic>{};
  }

  @override
  Future<Map<String, dynamic>> post(
    String path, [
    Map<String, dynamic>? body,
    int timeout = 90,
  ]) async {
    events.add('post:$path:${body?['mode']}:${body?['session_count']}');
    if (path == 'settings') {
      if (failSettings) throw Exception('forced settings failure');
      status['mode'] = body?['mode'];
      status['sessionCount'] = body?['session_count'];
    }
    return {'ok': true};
  }

  @override
  void close() {}
}

void main() {
  test('traffic units scale correctly', () {
    expect(rate(125000), '1.0 Mbps');
    expect(bytes(1024), '1.0 KiB');
  });

  test(
    'English and Chinese localization files are generated independently',
    () {
      expect(AppLocalizationsEn().connect, 'Connect');
      expect(AppLocalizationsZh().connect, '连接');
      expect(AppLocalizationsEn().routeHealthy, 'Network route is healthy');
      expect(AppLocalizationsZh().routeHealthy, '网络路由正常');
    },
  );

  testWidgets(
    'routing sheet applies route and session before restoring navigation',
    (tester) async {
      final api = _FakeApi();
      final selectedTab = ValueNotifier<int>(0);
      addTearDown(selectedTab.dispose);

      await tester.pumpWidget(
        MaterialApp(
          locale: const Locale('en'),
          localizationsDelegates: const [
            AppLocalizations.delegate,
            GlobalMaterialLocalizations.delegate,
            GlobalCupertinoLocalizations.delegate,
            GlobalWidgetsLocalizations.delegate,
          ],
          supportedLocales: AppLocalizations.supportedLocales,
          home: Scaffold(
            body: OverviewPage(
              api: api,
              selectedTab: selectedTab,
              onOverlayVisibilityChanged: (visible) =>
                  api.events.add('overlay:$visible'),
            ),
          ),
        ),
      );
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));

      expect(find.byIcon(Icons.route_rounded), findsOneWidget);
      await tester.tap(find.byIcon(Icons.route_rounded));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Full China routing'));
      await tester.ensureVisible(find.text('Triple mode'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Triple mode'));
      await tester.ensureVisible(find.text('Apply'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Apply'));
      await tester.pumpAndSettle();

      expect(api.events, contains('post:settings:global:3'));
      expect(
        api.events.indexOf('post:settings:global:3'),
        lessThan(api.events.lastIndexOf('overlay:false')),
      );
      expect(find.text('Full China routing · Triple mode'), findsOneWidget);

      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pump();
    },
  );

  testWidgets(
    'routing sheet restores navigation and keeps status after API failure',
    (tester) async {
      final api = _FakeApi()..failSettings = true;
      final selectedTab = ValueNotifier<int>(0);
      addTearDown(selectedTab.dispose);

      await tester.pumpWidget(
        MaterialApp(
          locale: const Locale('en'),
          localizationsDelegates: const [
            AppLocalizations.delegate,
            GlobalMaterialLocalizations.delegate,
            GlobalCupertinoLocalizations.delegate,
            GlobalWidgetsLocalizations.delegate,
          ],
          supportedLocales: AppLocalizations.supportedLocales,
          home: Scaffold(
            body: OverviewPage(
              api: api,
              selectedTab: selectedTab,
              onOverlayVisibilityChanged: (visible) =>
                  api.events.add('overlay:$visible'),
            ),
          ),
        ),
      );
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));
      await tester.tap(find.byIcon(Icons.route_rounded));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Full China routing'));
      await tester.ensureVisible(find.text('Apply'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Apply'));
      await tester.pumpAndSettle();

      expect(api.events, contains('post:settings:global:2'));
      expect(api.events.last, 'overlay:false');
      expect(find.text('Smart routing · Dual mode'), findsOneWidget);
      expect(find.text('Apply'), findsNothing);

      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pump();
    },
  );
}
