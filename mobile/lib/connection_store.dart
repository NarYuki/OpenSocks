import 'package:flutter_secure_storage/flutter_secure_storage.dart';

class ConnectionStore {
  static const _storage = FlutterSecureStorage();
  static Future<(String?, String?)> load() async => (
    await _storage.read(key: 'router_url'),
    await _storage.read(key: 'api_token'),
  );
  static Future<void> save(String url, String token) async {
    await _storage.write(key: 'router_url', value: url);
    await _storage.write(key: 'api_token', value: token);
  }

  static Future<void> clear() => _storage.deleteAll();
}
