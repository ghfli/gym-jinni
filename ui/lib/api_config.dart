/// Base URL for the Go grpc-gateway HTTP server (`service/service.go`).
///
/// Override at run time, for example:
/// `flutter run -d chrome --dart-define=API_BASE=http://127.0.0.1:8081`
class ApiConfig {
  ApiConfig._();

  static const String baseUrl = String.fromEnvironment(
    'API_BASE',
    defaultValue: 'http://127.0.0.1:8081',
  );
}
