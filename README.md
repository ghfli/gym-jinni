# gym-jinni
a service for gym owners, trainers and customers to manage their daily gym
activities

Please join this project to implement what
[vagaro.com](https://www.vagaro.com),
[gymmaster.com](https://www.gymmaster.com) and
[strava.com](https://www.strava.com) provide.  Start from implementing minimum
viable key features, work towards providing other major ones, and eventually
design something new.

Click this link to review the initial/ongoing [system design document](
https://www.notion.so/d0ca654781a9407199fe1c1abf6e99c7?v=7d033ab95d3e49f4a0b43f211f970a6e)
on notion.

Gym-Jinni uses a modular monolith: one Go service exposes gRPC (`:8080`) and an HTTP/JSON gateway (`:8081`) over PostgreSQL, with ten domain modules. Flutter provides the web client. Local deployment uses k3d/Ansible and Docker by default.

## Documentation

| Guide | Contents |
| --- | --- |
| [Development and tooling](docs/development.md) | Prerequisites, generation, tests, signing keys and optional Node.js tooling |
| [Backend](service/README.md) | API architecture, database migrations and service execution |
| [Accounts and administrator setup](service/RBAC.md) | First-admin bootstrap, permissions and management APIs |
| [Flutter UI](ui/README.md) | Current screens, API configuration, sessions and widget tests |
| [Deployment](k8s-setup/README.md) | Local cluster, CI, credentials, TLS readiness and deployment gaps |
| [Cluster quick start](k8s-setup/QUICKSTART.md) | Minimal source-grounded setup sequence |
| [Specification](spec.md) | Implemented/partial/planned requirements and delivery status |
| [HTTP and workflow inventory](docs/business-process-validation.md) | All 70 HTTP operations, six screens and dated validation evidence |
| [Contributor instructions](AGENTS.md) | Repository conventions and verification requirements |

Historical VM proposals remain in [memo.md](memo.md); the actual VM configuration is [Vagrantfile](Vagrantfile) and [setup-alx.sh](setup-alx.sh). Historical notes are not a tested alternative setup path.

## Build and verify

From the repository root, with Docker access, Go, curl and build/image dependency access:

```bash
service/scripts/validate-business.sh
```

This uses disposable PostgreSQL and a freshly generated service image, verifies API/bootstrap flows and migration round trips, then cleans up its resources. It does not use the deployed database. For local source builds, generate both bindings first:

```bash
cd service
make bufgen dbsql
make buflint
go test ./...
go build -o cmd/service .
```

Set `DBURL` and a dedicated `JWT_SECRET` of at least 32 bytes before starting the service; only explicit `APP_ENV=development` permits the development fallback. See [backend instructions](service/README.md). Generated bindings are ignored and should not be edited or committed.

For the UI, point `API_BASE` at a running backend:

```bash
cd ui
flutter pub get
flutter run -d chrome --dart-define=API_BASE=http://127.0.0.1:8081
flutter test
flutter analyze
```

Use `http://127.0.0.1:39081` for a local k3d gateway. The deployed web image uses same-origin `/v1/` requests through nginx. The default UI source URL also uses port `39081`.

## Application and management

Register/login, Home, Classes, My Bookings, Activity and Profile are implemented. A database operator provisions the first Super administrator with `go run ./cmd/bootstrap-admin -user-id N` from `service/`; subsequent grants use the authenticated RBAC API. There are currently no trainer/owner/admin management pages. Payments are stored records and notifications are an inbox; provider settlement, external delivery and other remaining workflows are tracked in the specification.

When configured, the local cluster publishes web UI at `http://localhost:39300`, HTTP API at `http://localhost:39081/v1/`, and gRPC at `localhost:39080`. Local deployment provisions/reuses the required signing Secret and blocks startup on migration errors; deployment-specific database/domain/image configuration still requires review; consult the deployment guide before running the full setup.

## CI and deployment

GitHub builds on pushes/PRs to `main`, `master`, and `csr`; non-PR runs publish images. Deployments require a manual workflow run on `main`/`master` with `deploy` enabled and a valid base64 `KUBECONFIG` secret for a reachable cluster. The actions use native Node.js 24. GitLab builds with Podman and also deploys manually. The workstation Node.js version is separate from the Actions runtime; see the development guide for portable host tooling guidance.

## Contributing

Read [AGENTS.md](AGENTS.md) and the specification before changing contracts, migrations or permissions. Keep commits focused and report actual verification results and limitations. Contact `newstart.infotech@gmail.com` or the project Slack at [gymjinni.slack.com](https://gymjinni.slack.com) for project discussions.

System administrator bootstrap and browser management are documented in [system administration](docs/system-administration.md).
