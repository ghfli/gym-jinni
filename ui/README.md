# Gym-Jinni Flutter client

Web is the primary target. Dart must satisfy `>=3.0.0 <4.0.0` in [pubspec.yaml](pubspec.yaml); other platform scaffolds exist but do not establish verified mobile/desktop support.

## Run, build and check

Run from `ui/`:

```bash
flutter pub get
flutter run -d chrome --dart-define=API_BASE=http://127.0.0.1:39081
flutter test
flutter analyze
flutter build web --dart-define=API_BASE=http://127.0.0.1:39081
```

Use port `8081` instead for a standalone backend. `lib/api_config.dart` defaults to `http://127.0.0.1:39081`. The container builds with an empty `API_BASE`; its nginx proxies same-origin `/v1/` to `main-service:8081`. When serving a web build elsewhere, configure a reachable API or a matching proxy. Do not assume a browser on another machine can reach the deployment through its own `127.0.0.1`.

## Screens and supported flows

| Screen | Implemented actions |
| --- | --- |
| Register / Log in | Create customer account; obtain session and access/refresh tokens |
| Home | Welcome and navigation shortcuts |
| Classes | Browse classes, show a class dialog, book |
| My Bookings | List own bookings and cancel confirmed bookings |
| Activity | Log workout/run/class activity, list records and totals |
| Profile | Show current name/ID; request server logout |

These are internal tabs/dialogs within one app URL, not named deep-link routes. Super administrators currently see these same screens; role/permission management, class editing, gyms, schedules, payment administration and reports are API-only. See [administrator setup](../service/RBAC.md), [API inventory](../docs/business-process-validation.md) and [UI backlog](../spec.md#10-flutter-client).

## Sessions and HTTP

`AuthState` stores credentials/session ID in memory. `SessionClient` retries a protected request once after a 401, shares concurrent renewal requests, and prevents late responses restoring a logged-out session. Server logout revokes the owned session; a failed logout leaves the session available for retry and displays an error. Tokens are not persisted in secure platform storage yet; browser reload requires login.

The client supports canonical protobuf enum names, nested statistics, lowerCamelCase fields and int64 string values where covered by its decoders, with compatibility for legacy numeric booking statuses/snake_case fields. Explicit class/booking permission errors are visible. Remaining screen error states are tracked in the specification. Backend authorization remains authoritative.

## Tests and extension points

`test/widget_test.dart` covers registration/login/navigation, booking cancellation, enum/stat decoding, permission errors and logout. `test/auth_state_test.dart` covers renewal, revocation, logout failure and late-response behavior. Both suites use injected HTTP clients and the required Provider auth state. Twelve tests passed in the dated remediation run; consult the validation report before interpreting historical failures.

Most widgets still live in `lib/main.dart`; split future features into feature files. Add meaningful widget/integration cases for new screens, including denial and network/expiry states. Node.js is optional host tooling and is not a Flutter runtime dependency; see [development guide](../docs/development.md).
