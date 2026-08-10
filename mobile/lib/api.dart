import 'dart:async';
import 'dart:convert';
import 'dart:io';

enum OpenSocksApiErrorKind { connection, timeout, response, server }

class OpenSocksApiException implements Exception {
  const OpenSocksApiException(this.kind, {this.statusCode, this.detail});

  final OpenSocksApiErrorKind kind;
  final int? statusCode;
  final String? detail;

  @override
  String toString() => detail ?? kind.name;
}

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
        throw OpenSocksApiException(
          OpenSocksApiErrorKind.response,
          statusCode: res.statusCode,
        );
      }
      if (res.statusCode < 200 || res.statusCode >= 300 || decoded is! Map) {
        throw OpenSocksApiException(
          OpenSocksApiErrorKind.server,
          statusCode: res.statusCode,
          detail: decoded is Map ? decoded['error']?.toString() : null,
        );
      }
      final data = Map<String, dynamic>.from(decoded);
      if (data['error'] != null) {
        throw OpenSocksApiException(
          OpenSocksApiErrorKind.server,
          statusCode: res.statusCode,
          detail: data['error'].toString(),
        );
      }
      return data;
    } on TimeoutException {
      throw const OpenSocksApiException(OpenSocksApiErrorKind.timeout);
    } on SocketException {
      throw const OpenSocksApiException(OpenSocksApiErrorKind.connection);
    } on HandshakeException {
      throw const OpenSocksApiException(OpenSocksApiErrorKind.connection);
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
