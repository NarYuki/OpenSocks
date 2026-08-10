import 'package:flutter_test/flutter_test.dart';
import 'package:opensocks_mobile/main.dart';
import 'package:opensocks_mobile/l10n/app_localizations_en.dart';
import 'package:opensocks_mobile/l10n/app_localizations_zh.dart';

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
}
