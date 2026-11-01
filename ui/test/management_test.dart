import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:provider/provider.dart';
import 'package:ui/auth_state.dart';
import 'package:ui/main.dart';
import 'package:ui/management_page.dart';

Widget harness(Widget child, http.Client client) => MultiProvider(
        providers: [
          ChangeNotifierProvider(
              create: (_) => AuthState()
                ..login(
                    accessToken: 'access',
                    refreshToken: 'refresh',
                    sessionId: 'session',
                    userId: 1,
                    userName: 'Admin')),
          Provider<http.Client>.value(value: client),
        ],
        child: child is GymJinniApp
            ? child
            : MaterialApp(home: Scaffold(body: child)));
http.Response response(Object value, [int code = 200]) =>
    http.Response(jsonEncode(value), code);

void main() {
  testWidgets('management navigation is shown only for a system administrator',
      (tester) async {
    var administrator = false;
    final client =
        MockClient((req) async => response(req.url.path == '/v1/users/1/roles'
            ? {
                'roles': administrator
                    ? [
                        {'name': 'super_admin'}
                      ]
                    : []
              }
            : {}));
    await tester.pumpWidget(harness(const GymJinniApp(), client));
    await tester.pumpAndSettle();
    expect(find.text('Management'), findsNothing);
    await tester.pumpWidget(const SizedBox());
    administrator = true;
    await tester.pumpWidget(harness(const GymJinniApp(), client));
    await tester.pumpAndSettle();
    expect(find.text('Management'), findsOneWidget);
    await tester.tap(find.text('Management'));
    await tester.pumpAndSettle();
    expect(find.text('System administration'), findsOneWidget);
  });

  testWidgets('user directory pagination and denied responses are visible',
      (tester) async {
    final client = MockClient((req) async {
      if (req.url.queryParameters['offset'] == '50')
        return response({'message': 'Access revoked'}, 403);
      return response({
        'users': [
          {'id': 1, 'name': 'First', 'email': 'first@example.com'}
        ],
        'nextOffset': 50
      });
    });
    await tester
        .pumpWidget(harness(const ManagementResource(kind: 'users'), client));
    await tester.pumpAndSettle();
    expect(find.text('First'), findsOneWidget);
    await tester.tap(find.text('Next'));
    await tester.pumpAndSettle();
    expect(find.textContaining('Access revoked'), findsOneWidget);
  });

  testWidgets('role grant sends UTC expiry and displays continuity denial',
      (tester) async {
    Map<String, dynamic>? sent;
    final client = MockClient((req) async {
      if (req.method == 'POST') {
        sent = jsonDecode(req.body) as Map<String, dynamic>;
        return response(
            {'message': 'Keep another permanent administrator'}, 400);
      }
      if (req.url.path == '/v1/roles')
        return response({
          'roles': [
            {'id': 1, 'name': 'super_admin'}
          ]
        });
      return response({'userRoles': []});
    });
    await tester.pumpWidget(harness(
        const UserRolesDialog(user: {'id': 1, 'name': 'Admin'}), client));
    await tester.pumpAndSettle();
    await tester.tap(find.byType(DropdownButtonFormField<int>));
    await tester.pumpAndSettle();
    await tester.tap(find.text('super_admin').last);
    await tester.pumpAndSettle();
    await tester.enterText(find.byType(TextField), '2035-01-01T12:00:00Z');
    await tester.tap(find.text('Grant role'));
    await tester.pumpAndSettle();
    expect(sent?['expires_at'], {
      'year': 2035,
      'month': 1,
      'day': 1,
      'hours': 12,
      'minutes': 0,
      'seconds': 0
    });
    expect(find.textContaining('Keep another permanent administrator'),
        findsOneWidget);
  });

  testWidgets('reserved role permissions cannot be toggled', (tester) async {
    final client = MockClient((req) async => response({
          'permissions': [
            {'id': 1, 'name': 'manage_roles'}
          ]
        }));
    await tester.pumpWidget(harness(
        const RolePermissionsDialog(role: {'id': 1, 'name': 'super_admin'}),
        client));
    await tester.pumpAndSettle();
    expect(
        tester
            .widget<CheckboxListTile>(find.byType(CheckboxListTile))
            .onChanged,
        isNull);
  });
}
