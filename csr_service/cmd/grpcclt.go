package main

import (
	"context"
	"fmt"
	"github.com/ghfli/gym-jinni/csr_service/gen/go/rbac/v1alpha"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	connectTo := "127.0.0.1:8082"
	conn, err := grpc.Dial(connectTo, grpc.WithBlock(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", connectTo, err)
	}
	defer conn.Close()
	log.Println("Connected to", connectTo)

	client := rbacv1alpha.NewRBACServiceClient(conn)
	ctx := context.Background()

	// Test 1: List all roles (should include seeded roles)
	log.Println("\n=== Test 1: List all roles ===")
	rolesResp, err := client.ListRoles(ctx, &rbacv1alpha.ListRolesRequest{})
	if err != nil {
		return fmt.Errorf("failed to ListRoles: %w", err)
	}
	log.Printf("Found %d roles:", len(rolesResp.Roles))
	for _, role := range rolesResp.Roles {
		log.Printf("  - %s (ID: %d): %s", role.Name, role.Id, role.Description)
	}

	// Test 2: List all permissions
	log.Println("\n=== Test 2: List all permissions ===")
	permsResp, err := client.ListPermissions(ctx, &rbacv1alpha.ListPermissionsRequest{})
	if err != nil {
		return fmt.Errorf("failed to ListPermissions: %w", err)
	}
	log.Printf("Found %d permissions:", len(permsResp.Permissions))
	for _, perm := range permsResp.Permissions {
		log.Printf("  - %s (ID: %d): %s.%s", perm.Name, perm.Id, perm.Resource, perm.Action)
	}

	// Test 3: Get a specific role by name
	log.Println("\n=== Test 3: Get 'customer' role by name ===")
	customerRole, err := client.GetRoleByName(ctx, &rbacv1alpha.GetRoleByNameRequest{
		Name: "customer",
	})
	if err != nil {
		return fmt.Errorf("failed to GetRoleByName: %w", err)
	}
	log.Printf("Customer role: %+v", customerRole.Role)

	// Test 4: Get permissions for customer role
	log.Println("\n=== Test 4: Get permissions for customer role ===")
	customerPerms, err := client.GetRolePermissions(ctx, &rbacv1alpha.GetRolePermissionsRequest{
		RoleId: customerRole.Role.Id,
	})
	if err != nil {
		return fmt.Errorf("failed to GetRolePermissions: %w", err)
	}
	log.Printf("Customer role has %d permissions:", len(customerPerms.Permissions))
	for _, perm := range customerPerms.Permissions {
		log.Printf("  - %s: %s.%s", perm.Name, perm.Resource, perm.Action)
	}

	// Test 5: Create a custom role
	log.Println("\n=== Test 5: Create custom 'test_role' ===")
	testRole, err := client.CreateRole(ctx, &rbacv1alpha.CreateRoleRequest{
		Name:        "test_role",
		Description: "A test role for demonstration",
	})
	if err != nil {
		log.Printf("Warning: Failed to create test_role (may already exist): %v", err)
	} else {
		log.Printf("Created test role: %+v", testRole.Role)
	}

	// Test 6: Create a custom permission
	log.Println("\n=== Test 6: Create custom 'test_permission' ===")
	testPerm, err := client.CreatePermission(ctx, &rbacv1alpha.CreatePermissionRequest{
		Name:        "test_permission",
		Resource:    "test_resource",
		Action:      "test_action",
		Description: "A test permission for demonstration",
	})
	if err != nil {
		log.Printf("Warning: Failed to create test_permission (may already exist): %v", err)
	} else {
		log.Printf("Created test permission: %+v", testPerm.Permission)
	}

	// Test 7: Assign role to user (assuming user ID 1 exists)
	log.Println("\n=== Test 7: Assign customer role to user ID 1 ===")
	assignResp, err := client.AssignRoleToUser(ctx, &rbacv1alpha.AssignRoleToUserRequest{
		UserId:    1,
		RoleId:    customerRole.Role.Id,
		GrantedBy: 1, // Self-assigned for test
	})
	if err != nil {
		log.Printf("Warning: Failed to assign role to user: %v", err)
	} else {
		log.Printf("Assigned role to user: success=%v", assignResp.Success)
	}

	// Test 8: Get user roles
	log.Println("\n=== Test 8: Get roles for user ID 1 ===")
	userRoles, err := client.GetUserRoles(ctx, &rbacv1alpha.GetUserRolesRequest{
		UserId: 1,
	})
	if err != nil {
		log.Printf("Warning: Failed to get user roles: %v", err)
	} else {
		log.Printf("User 1 has %d roles:", len(userRoles.Roles))
		for _, role := range userRoles.Roles {
			log.Printf("  - %s", role.Name)
		}
	}

	// Test 9: Get user permissions
	log.Println("\n=== Test 9: Get all permissions for user ID 1 ===")
	userPerms, err := client.GetUserPermissions(ctx, &rbacv1alpha.GetUserPermissionsRequest{
		UserId: 1,
	})
	if err != nil {
		log.Printf("Warning: Failed to get user permissions: %v", err)
	} else {
		log.Printf("User 1 has %d permissions:", len(userPerms.Permissions))
		for _, perm := range userPerms.Permissions {
			log.Printf("  - %s: %s.%s", perm.Name, perm.Resource, perm.Action)
		}
	}

	// Test 10: Check if user has specific permission
	log.Println("\n=== Test 10: Check if user 1 has 'book_class' permission ===")
	checkResp, err := client.CheckPermission(ctx, &rbacv1alpha.CheckPermissionRequest{
		UserId:         1,
		PermissionName: "book_class",
	})
	if err != nil {
		return fmt.Errorf("failed to CheckPermission: %w", err)
	}
	log.Printf("Permission check result: authorized=%v, reason=%s", checkResp.Authorized, checkResp.Reason)

	// Test 11: Check permission by resource and action
	log.Println("\n=== Test 11: Check if user 1 can 'create' 'class' ===")
	checkResp2, err := client.CheckPermissionByResourceAction(ctx, &rbacv1alpha.CheckPermissionByResourceActionRequest{
		UserId:   1,
		Resource: "class",
		Action:   "create",
	})
	if err != nil {
		return fmt.Errorf("failed to CheckPermissionByResourceAction: %w", err)
	}
	log.Printf("Permission check result: authorized=%v, reason=%s", checkResp2.Authorized, checkResp2.Reason)

	log.Println("\n=== All tests completed successfully ===")
	return nil
}

