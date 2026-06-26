import 'package:flutter/foundation.dart';

class AuthState extends ChangeNotifier {
  String? _accessToken;
  String? _refreshToken;
  int? _userId;
  String? _userName;

  bool get isLoggedIn => _accessToken != null;
  String? get accessToken => _accessToken;
  String? get refreshToken => _refreshToken;
  int? get userId => _userId;
  String? get userName => _userName;

  Map<String, String> get authHeaders => {
        if (_accessToken != null) 'Authorization': 'Bearer $_accessToken',
      };

  void login({
    required String accessToken,
    required String refreshToken,
    required int userId,
    required String userName,
  }) {
    _accessToken = accessToken;
    _refreshToken = refreshToken;
    _userId = userId;
    _userName = userName;
    notifyListeners();
  }

  void logout() {
    _accessToken = null;
    _refreshToken = null;
    _userId = null;
    _userName = null;
    notifyListeners();
  }
}
