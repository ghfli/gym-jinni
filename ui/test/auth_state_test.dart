import 'dart:async';
import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:ui/auth_state.dart';

AuthState signedIn() => AuthState()
  ..login(
      accessToken: 'old',
      refreshToken: 'refresh',
      sessionId: 'session',
      userId: 1,
      userName: 'Athlete');

void main() {
  test('expired access is renewed once and the original request is retried',
      () async {
    final auth = signedIn();
    var renewals = 0;
    var reads = 0;
    final client = SessionClient(auth, MockClient((request) async {
      if (request.url.path.endsWith('renew_token')) {
        renewals++;
        expect(jsonDecode(request.body)['refresh_tkn'], 'refresh');
        return http.Response('{"accessTkn":"new"}', 200);
      }
      reads++;
      if (reads == 1) return http.Response('{}', 401);
      expect(request.headers['authorization'], 'Bearer new');
      expect(request.headers['grpc-metadata-authorization'], 'Bearer new');
      return http.Response('{}', 200);
    }));
    final response = await client.get(Uri.parse('http://localhost/v1/classes'));
    expect(response.statusCode, 200);
    expect(reads, 2);
    expect(renewals, 1);
    client.close();
  });

  test('simultaneous renewals share a single request', () async {
    final auth = signedIn();
    final response = Completer<http.Response>();
    var calls = 0;
    final client = MockClient((_) {
      calls++;
      return response.future;
    });
    final first = auth.renew(client);
    final second = auth.renew(client);
    response.complete(http.Response('{"access_tkn":"new"}', 200));
    expect(await first, isTrue);
    expect(await second, isTrue);
    expect(calls, 1);
    client.close();
  });

  test('revoked refresh clears the local session', () async {
    final auth = signedIn();
    expect(await auth.renew(MockClient((_) async => http.Response('{}', 401))),
        isFalse);
    expect(auth.isLoggedIn, isFalse);
    expect(auth.sessionId, isNull);
  });

  test('failed logout keeps session available for retry', () async {
    final auth = signedIn();
    await expectLater(
        auth.logout(MockClient((_) async => http.Response('{}', 503))),
        throwsA(isA<http.ClientException>()));
    expect(auth.isLoggedIn, isTrue);
  });

  test('slow renewal cannot restore a logged-out session', () async {
    final auth = signedIn();
    final response = Completer<http.Response>();
    final client = MockClient((request) async {
      if (request.url.path.endsWith('renew_token')) return response.future;
      return http.Response('{}', 200);
    });
    final renewing = auth.renew(client);
    await auth.logout(client);
    response.complete(http.Response('{"accessTkn":"new"}', 200));
    expect(await renewing, isFalse);
    expect(auth.isLoggedIn, isFalse);
    client.close();
  });
}
