/// Base URL for the Go grpc-gateway HTTP server (`service/service.go`).
///
/// In the k8s / k3d deployment the UI is served by nginx which proxies
/// `/v1/` to the backend, so the built image passes `--dart-define=API_BASE=`
/// (empty string = same-origin). For local `flutter run -d chrome` against
/// the k3d cluster use port 39081 (the k3d LB port):
///   `flutter run -d chrome --dart-define=API_BASE=http://127.0.0.1:39081`
class ApiConfig {
  ApiConfig._();

  static const String baseUrl = String.fromEnvironment(
    'API_BASE',
    defaultValue: 'http://127.0.0.1:39081',
  );
}
