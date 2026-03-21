---
name: Role-Based Access Control Service Implementation
overview: ""
todos:
  - id: bccc17f3-8c97-498f-909e-3345cdbc6337
    content: Create csr_service directory structure and initialize Go module
    status: pending
  - id: f7ffb1d3-505d-4fb0-af40-023d7b0e5b75
    content: Create DBML schema for roles, permissions, role_permission, and user_role tables
    status: pending
  - id: ebf0b250-6ce1-483e-a8da-0926c3e0e172
    content: Write SQLC queries for all CRUD operations and authorization checks
    status: pending
  - id: 74276b53-dd4f-4c7a-a3e2-89439dfd2771
    content: Create migration files for schema creation and seed data
    status: pending
  - id: 269dedf1-8425-4233-ac12-53f0d191af86
    content: Define protobuf messages and RBACService with all required RPCs
    status: pending
  - id: 33f590fa-5545-48f0-851b-1c726e70b497
    content: Create buf.yaml, buf.work.yaml, and buf.gen.yaml for code generation
    status: pending
  - id: d6f8152a-4ca3-4b7e-89af-8b1a5f0aee86
    content: Implement ImRBACServiceServer with all RPC handlers and business logic
    status: pending
  - id: 4e2fadd4-6688-4041-8500-9d6bd055580a
    content: Implement gRPC authorization interceptor and permission mapping
    status: pending
  - id: 54e4d58e-4f19-484d-8995-15b6971fc62f
    content: Create Makefile with targets for building, code generation, and migrations
    status: pending
  - id: aef5c897-7105-476d-96cf-83e0cba15b33
    content: Create service.go main entry point with gRPC and HTTP gateway servers
    status: pending
  - id: ad7adb59-a17e-4d7b-a3c8-c62bff2c367d
    content: Create gRPC and HTTP test clients to validate all functionality
    status: pending
  - id: bd52f8f8-e459-4d7c-9878-e21faf7a1351
    content: Add README with setup instructions and integration guide
    status: pending
isProject: false
---

# Role-Based Access Control Service Implementation

## Overview

Create a new standalone `csr_service` directory at the root level (parallel to `service/` and `ui/`) that implements a comprehensive RBAC system with 5 roles (Super Admin, Gym Owner, Trainer, Customer, Guest), fine-grained permissions, multi-role support per user, and authorization middleware.

## Database Design

### Create DBML Schema (`csr_service/rbac/db/rbac.dbml`)

Define PostgreSQL schemas with the following tables:

**rbac.role table:**

- id (serial, PK)
- name (varchar, unique) - e.g., "super_admin", "gym_owner", "trainer", "customer", "guest"
- description (varchar)
- created_at (timestamptz)

**rbac.permission table:**

- id (serial, PK)
- name (varchar, unique) - e.g., "manage_users", "create_class", "book_class", "cancel_booking", "view_reports"
- resource (varchar) - e.g., "user", "class", "booking", "report"
- action (varchar) - e.g., "create", "read", "update", "delete", "manage"
- description (varchar)
- created_at (timestamptz)

**rbac.role_permission table (many-to-many):**

- role_id (int, FK to rbac.role, composite PK)
- permission_id (int, FK to rbac.permission, composite PK)
- created_at (timestamptz)

**rbac.user_role table (many-to-many):**

- user_id (int, FK to user.user from existing service, composite PK)
- role_id (int, FK to rbac.role, composite PK)
- granted_by (int, FK to user.user) - who assigned this role
- granted_at (timestamptz)
- expires_at (timestamptz, nullable) - optional role expiration

### Create SQLC Queries (`csr_service/rbac/db/rbac.sqlc.sql`)

Define queries for:

- Role CRUD: CreateRole, GetRole, ListRoles, UpdateRole, DeleteRole
- Permission CRUD: CreatePermission, GetPermission, ListPermissions, UpdatePermission, DeletePermission
- Role-Permission: AssignPermissionToRole, RevokePermissionFromRole, GetRolePermissions
- User-Role: AssignRoleToUser, RevokeRoleFromUser, GetUserRoles, GetUsersWithRole
- Authorization: GetUserPermissions (joins user_role -> role_permission -> permission)

### Create Migration Files

- `csr_service/mig/000001_init_schema_rbac.up.sql` - Create rbac schema and all tables
- `csr_service/mig/000001_init_schema_rbac.down.sql` - Drop tables and schema
- `csr_service/mig/000002_seed_default_roles_permissions.up.sql` - Insert default roles and permissions
- `csr_service/mig/000002_seed_default_roles_permissions.down.sql` - Clean up seed data

## Protocol Buffer Definitions

### Create RBAC Proto (`csr_service/rbac/pb/rbac/v1alpha/rbac.proto`)

Define messages and services:

**Messages:**

- Role (id, name, description, created_at)
- Permission (id, name, resource, action, description, created_at)
- RolePermission (role, permission)
- UserRole (user_id, role, granted_by, granted_at, expires_at)
- AuthorizationRequest (user_id, permission_name or resource+action)
- AuthorizationResponse (authorized bool, reason string)

**Service: RBACService**

- CreateRole, GetRole, ListRoles, UpdateRole, DeleteRole
- CreatePermission, GetPermission, ListPermissions, UpdatePermission, DeletePermission
- AssignPermissionToRole, RevokePermissionFromRole, GetRolePermissions
- AssignRoleToUser, RevokeRoleFromUser, GetUserRoles, GetUsersWithRole
- CheckPermission (authorization check)
- GetUserPermissions (list all permissions for a user)

### Create Buf Configuration

- `csr_service/rbac/pb/buf.yaml` - Module definition
- `csr_service/rbac/pb/buf.lock` - Dependency lock file
- `csr_service/buf.work.yaml` - Workspace with rbac/pb and validator/pb
- `csr_service/buf.gen.yaml` - Code generation config (same as service/buf.gen.yaml pattern)

## Service Implementation

### RBAC Service Server (`csr_service/rbac/rbac.go`)

Implement `ImRBACServiceServer` struct:

- Database connection initialization (similar to `service/user/user.go`)
- Implement all RPC methods defined in protobuf
- Business logic for role/permission management
- Authorization check logic (CheckPermission method)

### Authorization Middleware (`csr_service/middleware/auth.go`)

Create gRPC interceptor:

- Extract user context from request (user_id from session/token)
- Extract method name to determine required permission
- Call CheckPermission to verify authorization
- Return PermissionDenied error if unauthorized
- Integration point for existing service gRPC server

### Permission Mapping (`csr_service/middleware/permissions.go`)

Define permission mapping:

- Map gRPC method names to required permissions
- e.g., "/user.v1alpha.UserService/CreateUser" -> "manage_users"
- Public methods that don't require authentication

## Build System Integration

### Makefile (`csr_service/Makefile`)

Create Makefile with targets:

- `service`: Build csr_service binary
- `bufgen`: Generate protobuf code
- `dbsql`: Generate SQLC code
- `migup/migdn`: Run migrations
- `test`: Test service with gRPC client
- Separate DBURL for csr_service or shared with main service

### Go Module (`csr_service/go.mod`)

Initialize as `github.com/ghfli/gym-jinni/csr_service` with dependencies:

- Same dependencies as service/go.mod
- grpc-ecosystem/go-grpc-middleware for interceptors
- jackc/pgx/v4 for PostgreSQL

### Main Service Entry (`csr_service/service.go`)

Create main.go similar to `service/service.go`:

- Start gRPC server on different port (e.g., 8082)
- Start HTTP gateway on different port (e.g., 8083)
- Register RBACService
- Apply validator middleware

## Testing and Documentation

### gRPC Client (`csr_service/cmd/grpcclt.go`)

Test client to:

- Create roles (Super Admin, Gym Owner, Trainer, Customer, Guest)
- Create permissions
- Assign permissions to roles
- Assign roles to users
- Test authorization checks

### HTTP Client Script (`csr_service/cmd/httpclt.sh`)

Curl commands to test REST endpoints via gateway

### SQLC Config (`csr_service/rbac/db/sqlc.yaml`)

Configure SQLC code generation:

- Schema path: `../../gen/sql/rbac.dbml.sql`
- Queries: `rbac.sqlc.sql`
- Output: `../../gen/go/rbac/v1alpha`
- Package: `rbacv1alpha`

## Default Roles and Permissions Seed Data

### Roles:

1. **super_admin** - Full system access
2. **gym_owner** - Manage gym, trainers, classes, reports
3. **trainer** - Manage own classes, view customers
4. **customer** - Book/cancel classes, view own bookings
5. **guest** - View public information only

### Permission Examples:

- User management: manage_users, create_user, update_user, delete_user, view_user
- Class management: manage_classes, create_class, update_class, delete_class, view_class
- Booking: book_class, cancel_booking, view_booking
- Reports: view_reports, generate_reports
- System: manage_roles, manage_permissions

### Role-Permission Assignments:

- super_admin: ALL permissions
- gym_owner: All except system management permissions
- trainer: create_class, update_class, view_class, view_user, view_booking
- customer: book_class, cancel_booking, view_booking, view_class
- guest: view_class (public classes only)

## Integration Notes

The csr_service can be:

1. Run standalone on separate ports
2. Imported as middleware into existing `service/` for authorization
3. Used by other services via gRPC calls to check permissions

Future integration would update `service/service.go` to import and use the authorization middleware from csr_service.