import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:provider/provider.dart';
import 'package:ui/auth_state.dart';
import 'package:ui/main.dart';

AuthState signedIn() => AuthState()
  ..login(
      accessToken: 'access',
      sessionId: 'session',
      refreshToken: 'refresh',
      userId: 1,
      userName: 'Athlete');

Widget harness(Widget child, AuthState auth, http.Client client) =>
    MultiProvider(
      providers: [
        ChangeNotifierProvider.value(value: auth),
        Provider<http.Client>.value(value: client),
      ],
      child: child is GymJinniApp ? child : MaterialApp(home: child),
    );

http.Response jsonResponse(Object body, [int status = 200]) =>
    http.Response(jsonEncode(body), status,
        headers: {'content-type': 'application/json'});

void main() {
  testWidgets('shows registration and login with required providers',
      (tester) async {
    await tester.pumpWidget(harness(const GymJinniApp(), AuthState(),
        MockClient((_) async => jsonResponse({}))));
    expect(find.text('Register'), findsOneWidget);
    expect(find.text('Create account'), findsOneWidget);
    await tester.tap(find.text('Log in').first);
    await tester.pumpAndSettle();
    expect(find.text('Log in'), findsWidgets);
  });

  testWidgets('login reaches home and navigation opens classes',
      (tester) async {
    final auth = AuthState();
    final client = MockClient((request) async {
      if (request.url.path == '/v1/login_user') {
        final body = jsonDecode(request.body) as Map<String, dynamic>;
        expect(body['user']['email'], 'athlete@example.com');
        return jsonResponse({
          'user': {'id': 1, 'name': 'Athlete'},
          'accessTkn': 'access',
          'refreshTkn': 'refresh',
          'sessionId': 'session'
        });
      }
      expect(request.headers['authorization'], 'Bearer access');
      return jsonResponse({});
    });
    await tester.pumpWidget(harness(const GymJinniApp(), auth, client));
    await tester.tap(find.text('Log in').first);
    await tester.pumpAndSettle();
    final fields = find.byType(TextField).hitTestable();
    await tester.enterText(fields.at(0), 'athlete@example.com');
    await tester.enterText(fields.at(1), 'password');
    await tester.tap(find.widgetWithText(FilledButton, 'Log in'));
    await tester.pumpAndSettle();
    expect(find.text('Welcome, Athlete!'), findsOneWidget);
    await tester.tap(find.text('Classes').last);
    await tester.pumpAndSettle();
    expect(find.byType(ClassListPage), findsOneWidget);
  });

  testWidgets('numeric booking status supports cancellation and refresh',
      (tester) async {
    var cancelled = false;
    final client = MockClient((request) async {
      if (request.method == 'POST') {
        expect(request.url.path, '/v1/bookings/7/cancel');
        cancelled = true;
        return jsonResponse({});
      }
      return jsonResponse({
        'bookings': [
          {'id': 7, 'class_id': 2, 'status': cancelled ? 2 : 1}
        ]
      });
    });
    await tester
        .pumpWidget(harness(const MyBookingsPage(), signedIn(), client));
    await tester.pumpAndSettle();
    expect(find.text('Confirmed'), findsOneWidget);
    await tester.tap(find.text('Cancel'));
    await tester.pumpAndSettle();
    expect(cancelled, isTrue);
    expect(find.text('Cancelled'), findsOneWidget);
  });

  testWidgets('UI-05 canonical protobuf enum permits cancellation',
      (tester) async {
    final client = MockClient((_) async => jsonResponse({
          'bookings': [
            {'id': 7, 'classId': 2, 'status': 'BOOKING_STATUS_CONFIRMED'}
          ]
        }));
    await tester
        .pumpWidget(harness(const MyBookingsPage(), signedIn(), client));
    await tester.pumpAndSettle();
    expect(find.text('Confirmed'), findsOneWidget);
    expect(find.text('Cancel'), findsOneWidget);
  });

  testWidgets('UI-05 nested activity statistics display actual totals',
      (tester) async {
    final client = MockClient(
        (request) async => request.url.path.endsWith('activity_stats')
            ? jsonResponse({
                'stats': {
                  'total_count': '2',
                  'total_duration': '40',
                  'total_calories': '200'
                }
              })
            : jsonResponse({'activities': []}));
    await tester.pumpWidget(harness(const ActivityPage(), signedIn(), client));
    await tester.pumpAndSettle();
    expect(find.text('40'), findsOneWidget);
    expect(find.text('200'), findsOneWidget);
  });

  testWidgets('UI-02 permission denial is visible rather than empty classes',
      (tester) async {
    await tester.pumpWidget(harness(
        const ClassListPage(),
        signedIn(),
        MockClient(
            (_) async => jsonResponse({'message': 'Permission denied'}, 403))));
    await tester.pumpAndSettle();
    expect(find.textContaining('403'), findsOneWidget);
  });

  testWidgets('UI-06 logout revokes server session and returns to auth',
      (tester) async {
    var revoked = false;
    final auth = signedIn();
    final client = MockClient((request) async {
      if (request.url.path == '/v1/logout_user') {
        expect(jsonDecode(request.body)['session_id'], 'session');
        revoked = true;
      }
      return jsonResponse({});
    });
    await tester.pumpWidget(harness(const GymJinniApp(), auth, client));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Profile').last);
    await tester.pumpAndSettle();
    await tester.tap(find.text('Log out'));
    await tester.pumpAndSettle();
    expect(auth.isLoggedIn, isFalse);
    expect(find.text('Register'), findsOneWidget);
    expect(revoked, isTrue);
  });
}
