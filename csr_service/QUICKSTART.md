# CSR Service - Quick Start Guide

## Prerequisites Check

Ensure you have these tools installed:

```bash
# Check Go
go version  # Should be 1.18 or higher

# Check buf
buf --version

# Check sqlc
sqlc version

# Check dbml2sql
dbml2sql --version

# Check migrate
migrate -version

# Check PostgreSQL/Docker
docker ps
```

## 5-Minute Quick Start

### Step 1: Navigate to csr_service

```bash
cd csr_service
```

### Step 2: Setup Database (Choose One)

**Option A: Fresh PostgreSQL Container**
```bash
make setup
```
This will:
- Create PostgreSQL container
- Create database
- Run all migrations
- Seed default roles and permissions

**Option B: Use Existing PostgreSQL**
```bash
# Start PostgreSQL if not running
make pgup

# Run migrations only
make migup
```

### Step 3: Generate Code

```bash
# Generate all code (protobuf + SQLC)
make bufgen dbsql

# Or just build everything at once
make service
```

### Step 4: Set Database URL

```bash
export DBURL="postgresql://root:gj@127.0.0.1:5432/gj?sslmode=disable&search_path=public"
```

### Step 5: Start the Service

```bash
./cmd/service
```

You should see:
```
gRPC server listening on 127.0.0.1:8082
gRPC gateway server listening on :8083
```

### Step 6: Test the Service

**Open a new terminal** and run:

```bash
cd csr_service

# Test with gRPC client
make grpcclt
./cmd/grpcclt

# Test with HTTP
chmod +x cmd/httpclt.sh
./cmd/httpclt.sh
```

## Verify Installation

### Check Roles

```bash
curl http://127.0.0.1:8083/v1/roles | jq '.roles[].name'
```

Expected output:
```
"customer"
"guest"
"gym_owner"
"super_admin"
"trainer"
```

### Check Permissions

```bash
curl http://127.0.0.1:8083/v1/permissions | jq '.permissions | length'
```

Expected: `23` permissions

### Check Role Permissions

```bash
# Get customer role permissions
curl http://127.0.0.1:8083/v1/roles/4/permissions | jq '.permissions[].name'
```

Expected output:
```
"book_class"
"cancel_booking"
"view_booking"
"view_class"
```

## Common Operations

### Assign Role to User

```bash
curl -X POST http://127.0.0.1:8083/v1/users/1/roles/4 \
  -H "Content-Type: application/json" \
  -d '{"user_id": 1, "role_id": 4, "granted_by": 1}'
```

### Check User Permissions

```bash
curl http://127.0.0.1:8083/v1/users/1/permissions | jq '.permissions[].name'
```

### Verify User Has Permission

```bash
curl -X POST http://127.0.0.1:8083/v1/check-permission \
  -H "Content-Type: application/json" \
  -d '{"user_id": 1, "permission_name": "book_class"}' | jq '.'
```

Expected output:
```json
{
  "authorized": true,
  "reason": "Permission granted"
}
```

## Troubleshooting

### Issue: "Connection refused"

**Problem**: Service not running or wrong port

**Solution**:
```bash
# Check if service is running
ps aux | grep service

# Check port
netstat -an | grep 8082
```

### Issue: "Database connection failed"

**Problem**: PostgreSQL not running or wrong credentials

**Solution**:
```bash
# Start PostgreSQL
make pgup

# Verify connection
docker exec -it postgres psql -U root -d gj -c "SELECT 1;"

# Check DBURL
echo $DBURL
```

### Issue: "Table does not exist"

**Problem**: Migrations not run

**Solution**:
```bash
# Check migration version
make migvn

# Run migrations
make migup
```

### Issue: "Code generation failed"

**Problem**: buf or sqlc not configured correctly

**Solution**:
```bash
# Update buf dependencies
make bufmup

# Regenerate schema
make dbsch

# Regenerate everything
make bufgen dbsql
```

### Issue: "No roles found"

**Problem**: Seed data not loaded

**Solution**:
```bash
# Check migration version (should be 2)
make migvn

# If version is 1, run second migration
make migup1
```

## Directory Structure Reference

```
csr_service/
├── cmd/
│   ├── service          # Service binary (after build)
│   ├── grpcclt          # Test client binary (after build)
│   ├── grpcclt.go       # Test client source
│   └── httpclt.sh       # HTTP test script
├── gen/                 # Generated code (created after bufgen/dbsql)
│   ├── go/
│   │   └── rbac/v1alpha/
│   └── sql/
├── mig/                 # Migration files
├── rbac/                # Service implementation
├── middleware/          # Authorization middleware
└── validator/           # Validator proto
```

## Next Steps

After successful setup:

1. **Read README.md** for detailed documentation
2. **Review rbac.proto** to understand the full API
3. **Explore middleware/** to see authorization implementation
4. **Test with your own data** using the gRPC or HTTP clients
5. **Integrate with main service** following the README guide

## Quick Reference Commands

```bash
# Build
make service          # Build service binary
make grpcclt          # Build test client

# Database
make pgup            # Start PostgreSQL
make pgdn            # Stop PostgreSQL
make migup           # Run migrations
make migdn           # Rollback migrations

# Code Generation
make bufgen          # Generate protobuf code
make dbsql           # Generate SQLC code
make dbsch           # Generate SQL schema from DBML

# Testing
make test            # Run all tests
./cmd/grpcclt        # Run gRPC tests
./cmd/httpclt.sh     # Run HTTP tests

# Cleanup
make clean           # Remove generated files and binaries
make teardn          # Full teardown (migrations + database + container)
```

## Getting Help

- Check **README.md** for comprehensive documentation
- Review **IMPLEMENTATION_SUMMARY.md** for architecture overview
- Look at **rbac/pb/rbac/v1alpha/rbac.proto** for API reference
- Examine **cmd/grpcclt.go** for usage examples

## Success Indicators

You know everything is working when:

✅ Service starts without errors
✅ Both ports 8082 and 8083 are listening
✅ `curl http://127.0.0.1:8083/v1/roles` returns 5 roles
✅ `curl http://127.0.0.1:8083/v1/permissions` returns 23 permissions
✅ Test clients complete successfully
✅ Permission checks return expected results

Happy coding! 🚀

