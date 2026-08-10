import 'dart:convert';
import 'dart:io';

class OpenSocksApi {
  OpenSocksApi(this.baseUrl, this.token);
  final String baseUrl;
  final String token;

  Future<Map<String, dynamic>> request(
    String path, {
    String method = 'GET',
    Map<String, dynamic>? body,
    int timeout = 35,
  }) async {
    final client = HttpClient()..connectionTimeout = const Duration(seconds: 8);
    try {
      final uri = Uri.parse(
        '${baseUrl.replaceAll(RegExp(r'/+$'), '')}/${path.replaceAll(RegExp(r'^/+'), '')}',
      );
      final req = await client
          .openUrl(method, uri)
          .timeout(Duration(seconds: timeout));
      req.headers.set(HttpHeaders.authorizationHeader, 'Bearer $token');
      req.headers.contentType = ContentType.json;
      if (body != null) req.write(jsonEncode(body));
      final res = await req.close().timeout(Duration(seconds: timeout));
      final text = await utf8.decoder.bind(res).join();
      dynamic decoded;
      try {
        decoded = jsonDecode(text);
      } catch (_) {
        throw Exception(
          'HTTP ${res.statusCode}: ${text.substring(0, text.length.clamp(0, 160))}',
        );
      }
      if (res.statusCode < 200 || res.statusCode >= 300 || decoded is! Map) {
        throw Exception(
          decoded is Map
              ? (decoded['error'] ?? 'HTTP ${res.statusCode}')
              : 'HTTP ${res.statusCode}',
        );
      }
      final data = Map<String, dynamic>.from(decoded);
      if (data['error'] != null) throw Exception(data['error']);
      return data;
    } finally {
      client.close(force: true);
    }
  }

  Future<Map<String, dynamic>> get(String path, {int timeout = 35}) =>
      request(path, timeout: timeout);
  Future<Map<String, dynamic>> post(
    String path, [
    Map<String, dynamic>? body,
    int timeout = 90,
  ]) => request(path, method: 'POST', body: body ?? {}, timeout: timeout);
}
