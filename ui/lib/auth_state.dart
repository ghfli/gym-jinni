import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:http/http.dart' as http;

import 'api_config.dart';

class AuthState extends ChangeNotifier {
  String? _accessToken;
  String? _refreshToken;
  String? _sessionId;
  int? _userId;
  String? _userName;
  Future<bool>? _renewal;

  bool get isLoggedIn => _accessToken?.isNotEmpty == true;
  String? get accessToken => _accessToken;
  String? get refreshToken => _refreshToken;
  String? get sessionId => _sessionId;
  int? get userId => _userId;
  String? get userName => _userName;

  Map<String, String> get authHeaders => {
        if (isLoggedIn) 'Authorization': 'Bearer $_accessToken',
        if (isLoggedIn) 'Grpc-Metadata-authorization': 'Bearer $_accessToken',
      };

  void login({
    required String accessToken,
    required String refreshToken,
    required String sessionId,
    required int userId,
    required String userName,
  }) {
    if (accessToken.isEmpty || refreshToken.isEmpty || sessionId.isEmpty) {
      throw const FormatException('Incomplete login response');
    }
    _accessToken = accessToken;
    _refreshToken = refreshToken;
    _sessionId = sessionId;
    _userId = userId;
    _userName = userName;
    notifyListeners();
  }

  void _clear() {
    _accessToken = null;
    _refreshToken = null;
    _sessionId = null;
    _userId = null;
    _userName = null;
    notifyListeners();
  }

  Future<void> logout(http.Client client) async {
    final response = await client.post(
      Uri.parse('${ApiConfig.baseUrl}/v1/logout_user'),
      headers: {'Content-Type': 'application/json', ...authHeaders},
      body: jsonEncode({'session_id': _sessionId}),
    );
    if (response.statusCode >= 300 && response.statusCode != 401) {
      throw http.ClientException('Logout failed (${response.statusCode})');
    }
    _clear();
  }

  Future<bool> renew(http.Client client) async {
    if (_renewal != null) return _renewal!;
    final pending = _renew(client);
    _renewal = pending;
    try {
      return await pending;
    } finally {
      _renewal = null;
    }
  }

  Future<bool> _renew(http.Client client) async {
    final previousRefresh = _refreshToken;
    final previousSession = _sessionId;
    final response = await client.post(
      Uri.parse('${ApiConfig.baseUrl}/v1/renew_token'),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({'refresh_tkn': previousRefresh}),
    );
    // A slow response must not restore a session after logout or a new login.
    if (_sessionId != previousSession || !isLoggedIn) return false;
    if (response.statusCode == 401) {
      _clear();
      return false;
    }
    if (response.statusCode >= 300) {
      throw http.ClientException(
          'Session renewal failed (${response.statusCode})');
    }
    final data = jsonDecode(response.body) as Map<String, dynamic>;
    final token = (data['accessTkn'] ?? data['access_tkn'])?.toString();
    if (token == null || token.isEmpty) {
      throw const FormatException('Incomplete renewal response');
    }
    _accessToken = token;
    notifyListeners();
    return true;
  }
}

// Retry once after a 401, sharing one renewal across simultaneous page loads.
class SessionClient extends http.BaseClient {
  SessionClient(this.auth, this.delegate);
  final AuthState auth;
  final http.Client delegate;

  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    final bytes = await request.finalize().toBytes();
    http.Request copy() => http.Request(request.method, request.url)
      ..headers.addAll(request.headers)
      ..headers.addAll(auth.authHeaders)
      ..followRedirects = request.followRedirects
      ..maxRedirects = request.maxRedirects
      ..persistentConnection = request.persistentConnection
      ..bodyBytes = bytes;
    final response = await delegate.send(copy());
    final sessionRequest = request.url.path.endsWith('/login_user') ||
        request.url.path.endsWith('/create_user') ||
        request.url.path.endsWith('/renew_token');
    if (response.statusCode != 401 || !auth.isLoggedIn || sessionRequest) {
      return response;
    }
    if (!await auth.renew(delegate)) return response;
    await response.stream.drain<void>();
    return delegate.send(copy());
  }

  @override
  void close() => delegate.close();
}
