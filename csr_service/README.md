# CSR Service - Role-Based Access Control (RBAC) Service

A comprehensive RBAC service for the gym-jinni project that provides role and permission management with fine-grained authorization capabilities.

## Overview

The CSR (Customer Service & RBAC) service implements a complete role-based access control system with:

- **5 Default Roles**: Super Admin, Gym Owner, Trainer, Customer, Guest
- **Fine-grained Permissions**: Resource-action based permission model
- **Multi-role Support**: Users can have multiple roles simultaneously
- **Role Expiration**: Roles can have optional expiration dates
- **Authorization Middleware**: gRPC interceptors for automatic permission checking
- **REST API**: HTTP gateway for easy integration

## Architecture

### Database Schema

The service uses PostgreSQL with the following schema in the `rbac` namespace:

- **rbac.role**: Stores role definitions
- **rbac.permission**: Stores permission definitions (resource + action)
- **rbac.role_permission**: Many-to-many mapping of roles to permissions
- **rbac.user_role**: Many-to-many mapping of users to roles with metadata

### Technology Stack

- **Language**: Go 1.18
- **Database**: PostgreSQL
- **API**: gRPC with HTTP/REST gateway
- **Code Generation**: Protocol Buffers (buf), SQLC
- **Validation**: protobuf validators
- **Migration**: golang-migrate

## Directory Structure

```
csr_service/
├── rbac/
│   ├── db/
│   │   ├── rbac.dbml          # Database schema definition
│   │   ├── rbac.sqlc.sql      # SQLC queries
│   │   └── sqlc.yaml          # SQLC configuration
│   ├── pb/
│   │   └── rbac/v1alpha/
│   │       └── rbac.proto     # Protobuf definitions
│   └── rbac.go                # Service implementation
├── middleware/
│   ├── auth.go                # Authorization interceptor
│   └── permissions.go         # Permission mappings
├── mig/
│   ├── 000001_init_schema_rbac.up.sql    # Schema migration
│   ├── 000001_init_schema_rbac.down.sql
│   ├── 000002_seed_default_roles_permissions.up.sql   # Seed data
│   └── 000002_seed_default_roles_permissions.down.sql
├── cmd/
│   ├── grpcclt.go            # gRPC test client
│   └── httpclt.sh            # HTTP test script
├── validator/pb/             # Validator proto definitions
├── service.go                # Main service entry point
├── Makefile                  # Build automation
├── go.mod                    # Go module definition
└── README.md                 # This file
```

## Default Roles and Permissions

### Roles

1. **super_admin** - Full system access
2. **gym_owner** - Manage gym, trainers, classes, reports
3. **trainer** - Manage own classes, view customers
4. **customer** - Book/cancel classes, view own bookings
5. **guest** - View public information only

### Permission Structure

Permissions follow a `resource.action` model:

**User Management:**
- `manage_users`, `create_user`, `view_user`, `update_user`, `delete_user`

**Class Management:**
- `manage_classes`, `create_class`, `view_class`, `update_class`, `delete_class`

**Booking:**
- `manage_bookings`, `book_class`, `view_booking`, `cancel_booking`

**Reports:**
- `view_reports`, `generate_reports`, `manage_reports`

**System/RBAC:**
- `manage_roles`, `manage_permissions`, `view_roles`, `view_permissions`

### Role-Permission Assignments

| Role | Permissions |
|------|------------|
| super_admin | ALL permissions |
| gym_owner | All except system management |
| trainer | create_class, update_class, view_class, view_user, view_booking |
| customer | book_class, cancel_booking, view_booking, view_class |
| guest | view_class |

## Setup and Installation

### Prerequisites

- Go 1.18+
- PostgreSQL
- Docker (for PostgreSQL)
- buf (for protobuf generation)
- sqlc (for SQL code generation)
- dbml2sql (for schema generation)
- golang-migrate (for migrations)

### Step 1: Setup PostgreSQL

```bash
# Create PostgreSQL container (if not exists)
make pgcr

# Or use existing PostgreSQL from the main service
# The service shares the same database as the main gym-jinni service
```

### Step 2: Generate Code

```bash
# Generate database schema from DBML
make dbsch

# Generate Go code from SQL queries
make dbsql

# Generate Go code from protobuf definitions
make bufgen

# Or run all code generation at once
make service
```

### Step 3: Run Migrations

```bash
# Run all migrations (creates schema and seeds data)
make migup

# To rollback
make migdn
```

### Step 4: Build and Run Service

```bash
# Build the service
make service

# Run the service (starts gRPC on 8082 and HTTP on 8083)
export DBURL="postgresql://root:gj@127.0.0.1:5432/gj?sslmode=disable&search_path=public"
./cmd/service
```

## Usage

### gRPC Client

```bash
# Build and run the test client
make grpcclt
./cmd/grpcclt
```

### HTTP/REST API

```bash
# Run the HTTP test script
chmod +x cmd/httpclt.sh
./cmd/httpclt.sh
```

### Example API Calls

**List all roles:**
```bash
curl http://127.0.0.1:8083/v1/roles
```

**Get role by name:**
```bash
curl http://127.0.0.1:8083/v1/roles/by-name/customer
```

**Assign role to user:**
```bash
curl -X POST http://127.0.0.1:8083/v1/users/1/roles/4 \
  -H "Content-Type: application/json" \
  -d '{"user_id": 1, "role_id": 4, "granted_by": 1}'
```

**Check permission:**
```bash
curl -X POST http://127.0.0.1:8083/v1/check-permission \
  -H "Content-Type: application/json" \
  -d '{"user_id": 1, "permission_name": "book_class"}'
```

**Get user permissions:**
```bash
curl http://127.0.0.1:8083/v1/users/1/permissions
```

## Integration with Existing Services

### Option 1: Standalone Service

Run csr_service as a separate microservice and call it via gRPC from other services.

```go
// In your service
import rbacv1alpha "github.com/ghfli/gym-jinni/csr_service/gen/go/rbac/v1alpha"

conn, _ := grpc.Dial("127.0.0.1:8082", grpc.WithInsecure())
rbacClient := rbacv1alpha.NewRBACServiceClient(conn)

// Check permission
resp, _ := rbacClient.CheckPermission(ctx, &rbacv1alpha.CheckPermissionRequest{
    UserId: 1,
    PermissionName: "create_class",
})
```

### Option 2: Middleware Integration

Import the authorization middleware into your existing service:

```go
// In service/service.go
import (
    "github.com/ghfli/gym-jinni/csr_service/middleware"
    rbacv1alpha "github.com/ghfli/gym-jinni/csr_service/gen/go/rbac/v1alpha"
)

// Create RBAC client
rbacConn, _ := grpc.Dial("127.0.0.1:8082", grpc.WithInsecure())
rbacClient := rbacv1alpha.NewRBACServiceClient(rbacConn)

// Add middleware to server
server := grpc.NewServer(
    grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
        grpc_validator.UnaryServerInterceptor(),
        middleware.AuthInterceptor(rbacClient),  // Add this
    )))
```

### Authentication Note

The current implementation extracts user ID from gRPC metadata header `user-id`. In production, you should:

1. Implement JWT token validation
2. Extract user ID from validated token
3. Update `extractUserID()` function in `middleware/auth.go`

Example metadata usage:
```go
// In your client
md := metadata.Pairs("user-id", "1")
ctx := metadata.NewOutgoingContext(context.Background(), md)
```

## Development

### Running Tests

```bash
# Run all tests
make test

# This will:
# 1. Build service and client
# 2. Start the service in background
# 3. Run gRPC client tests
# 4. Run HTTP client tests
# 5. Stop the service
```

### Adding New Permissions

1. Add permission to `mig/000002_seed_default_roles_permissions.up.sql`
2. Assign to roles as needed
3. Update `middleware/permissions.go` to map gRPC methods to permissions
4. Run `make migup` to apply changes

### Adding New Roles

1. Add role to `mig/000002_seed_default_roles_permissions.up.sql`
2. Assign permissions to the role
3. Run `make migup` to apply changes

## Makefile Targets

- `make service` - Build the service binary
- `make grpcclt` - Build the gRPC test client
- `make test` - Run all tests
- `make bufgen` - Generate protobuf code
- `make dbsql` - Generate SQLC code
- `make dbsch` - Generate SQL schema from DBML
- `make migup` - Run database migrations
- `make migdn` - Rollback database migrations
- `make pgcr` - Create PostgreSQL container
- `make pgup` - Start PostgreSQL container
- `make pgdn` - Stop PostgreSQL container
- `make clean` - Clean generated files

## API Reference

### gRPC Service: RBACService

Full API documentation is available in the protobuf definition at `rbac/pb/rbac/v1alpha/rbac.proto`.

**Role Management:**
- `CreateRole`, `GetRole`, `GetRoleByName`, `ListRoles`, `UpdateRole`, `DeleteRole`

**Permission Management:**
- `CreatePermission`, `GetPermission`, `GetPermissionByName`, `ListPermissions`, `UpdatePermission`, `DeletePermission`

**Role-Permission Assignment:**
- `AssignPermissionToRole`, `RevokePermissionFromRole`, `GetRolePermissions`

**User-Role Assignment:**
- `AssignRoleToUser`, `RevokeRoleFromUser`, `GetUserRoles`, `GetUserRoleDetails`, `GetUsersWithRole`

**Authorization:**
- `CheckPermission`, `CheckPermissionByResourceAction`, `GetUserPermissions`

## Troubleshooting

### Database Connection Issues

Ensure the DBURL environment variable is set correctly:
```bash
export DBURL="postgresql://root:gj@127.0.0.1:5432/gj?sslmode=disable&search_path=public"
```

### Port Conflicts

The service uses:
- Port 8082 for gRPC
- Port 8083 for HTTP gateway

Change these with flags:
```bash
./cmd/service -grpc-server-endpoint=127.0.0.1:9082 -gateway-port=:9083
```

### Migration Errors

If migrations fail, check the current version:
```bash
make migvn
```

Force to a specific version if needed:
```bash
make migfr
```

## Contributing

When adding new features:

1. Update DBML schema if database changes are needed
2. Update protobuf definitions for API changes
3. Regenerate code with `make bufgen` and `make dbsql`
4. Update permission mappings in `middleware/permissions.go`
5. Add tests to `cmd/grpcclt.go`
6. Update this README

## License

Same as gym-jinni project.

