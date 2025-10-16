package rbac

import (
	"context"
	"database/sql"
	"fmt"
	. "github.com/ghfli/gym-jinni/csr_service/gen/go/rbac/v1alpha"
	"github.com/golang/protobuf/ptypes/timestamp"
	_ "github.com/jackc/pgx/v4/stdlib"
	"google.golang.org/genproto/googleapis/type/datetime"
	"log"
	"os"
	"time"
)

type ImRBACServiceServer struct {
	UnimplementedRBACServiceServer
	db *sql.DB
	q  *Queries
}

func NewImRBACServiceServer() (*ImRBACServiceServer, error) {
	dburl := os.Getenv("DBURL")
	log.Println("DBURL", dburl)
	db, err := sql.Open("pgx", dburl)
	if err != nil {
		log.Printf("Failed to open DBURL %s: %v", dburl, err)
		return &ImRBACServiceServer{}, err
	}
	return &ImRBACServiceServer{
		db: db,
		q:  New(db),
	}, nil
}

// Helper function to convert time.Time to google.type.DateTime
func timeToDateTime(t time.Time) *datetime.DateTime {
	return &datetime.DateTime{
		Year:    int32(t.Year()),
		Month:   int32(t.Month()),
		Day:     int32(t.Day()),
		Hours:   int32(t.Hour()),
		Minutes: int32(t.Minute()),
		Seconds: int32(t.Second()),
		Nanos:   int32(t.Nanosecond()),
	}
}

// Helper function to convert google.type.DateTime to time.Time
func dateTimeToTime(dt *datetime.DateTime) time.Time {
	if dt == nil {
		return time.Time{}
	}
	return time.Date(
		int(dt.Year), time.Month(dt.Month), int(dt.Day),
		int(dt.Hours), int(dt.Minutes), int(dt.Seconds),
		int(dt.Nanos), time.UTC,
	)
}

// Helper function to convert sql.NullTime to google.type.DateTime pointer
func nullTimeToDateTime(nt sql.NullTime) *datetime.DateTime {
	if !nt.Valid {
		return nil
	}
	return timeToDateTime(nt.Time)
}

// Role Management

func (s *ImRBACServiceServer) CreateRole(ctx context.Context, req *CreateRoleRequest) (*CreateRoleResponse, error) {
	log.Println("CreateRole request:", req)
	
	role, err := s.q.CreateRole(ctx, CreateRoleParams{
		Name:        req.Name,
		Description: sql.NullString{String: req.Description, Valid: req.Description != ""},
	})
	if err != nil {
		log.Println("Failed to CreateRole:", err)
		return nil, err
	}
	
	return &CreateRoleResponse{
		Role: &Role{
			Id:          role.ID,
			Name:        role.Name,
			Description: role.Description.String,
			CreatedAt:   timeToDateTime(role.CreatedAt),
		},
	}, nil
}

func (s *ImRBACServiceServer) GetRole(ctx context.Context, req *GetRoleRequest) (*GetRoleResponse, error) {
	log.Println("GetRole request:", req)
	
	role, err := s.q.GetRole(ctx, req.Id)
	if err != nil {
		log.Println("Failed to GetRole:", err)
		return nil, err
	}
	
	return &GetRoleResponse{
		Role: &Role{
			Id:          role.ID,
			Name:        role.Name,
			Description: role.Description.String,
			CreatedAt:   timeToDateTime(role.CreatedAt),
		},
	}, nil
}

func (s *ImRBACServiceServer) GetRoleByName(ctx context.Context, req *GetRoleByNameRequest) (*GetRoleByNameResponse, error) {
	log.Println("GetRoleByName request:", req)
	
	role, err := s.q.GetRoleByName(ctx, req.Name)
	if err != nil {
		log.Println("Failed to GetRoleByName:", err)
		return nil, err
	}
	
	return &GetRoleByNameResponse{
		Role: &Role{
			Id:          role.ID,
			Name:        role.Name,
			Description: role.Description.String,
			CreatedAt:   timeToDateTime(role.CreatedAt),
		},
	}, nil
}

func (s *ImRBACServiceServer) ListRoles(ctx context.Context, req *ListRolesRequest) (*ListRolesResponse, error) {
	log.Println("ListRoles request")
	
	roles, err := s.q.ListRoles(ctx)
	if err != nil {
		log.Println("Failed to ListRoles:", err)
		return nil, err
	}
	
	protoRoles := make([]*Role, len(roles))
	for i, role := range roles {
		protoRoles[i] = &Role{
			Id:          role.ID,
			Name:        role.Name,
			Description: role.Description.String,
			CreatedAt:   timeToDateTime(role.CreatedAt),
		}
	}
	
	return &ListRolesResponse{Roles: protoRoles}, nil
}

func (s *ImRBACServiceServer) UpdateRole(ctx context.Context, req *UpdateRoleRequest) (*UpdateRoleResponse, error) {
	log.Println("UpdateRole request:", req)
	
	err := s.q.UpdateRole(ctx, UpdateRoleParams{
		ID:          req.Id,
		Name:        req.Name,
		Description: sql.NullString{String: req.Description, Valid: req.Description != ""},
	})
	if err != nil {
		log.Println("Failed to UpdateRole:", err)
		return nil, err
	}
	
	// Fetch updated role
	role, err := s.q.GetRole(ctx, req.Id)
	if err != nil {
		log.Println("Failed to fetch updated role:", err)
		return nil, err
	}
	
	return &UpdateRoleResponse{
		Role: &Role{
			Id:          role.ID,
			Name:        role.Name,
			Description: role.Description.String,
			CreatedAt:   timeToDateTime(role.CreatedAt),
		},
	}, nil
}

func (s *ImRBACServiceServer) DeleteRole(ctx context.Context, req *DeleteRoleRequest) (*DeleteRoleResponse, error) {
	log.Println("DeleteRole request:", req)
	
	err := s.q.DeleteRole(ctx, req.Id)
	if err != nil {
		log.Println("Failed to DeleteRole:", err)
		return &DeleteRoleResponse{Success: false}, err
	}
	
	return &DeleteRoleResponse{Success: true}, nil
}

// Permission Management

func (s *ImRBACServiceServer) CreatePermission(ctx context.Context, req *CreatePermissionRequest) (*CreatePermissionResponse, error) {
	log.Println("CreatePermission request:", req)
	
	permission, err := s.q.CreatePermission(ctx, CreatePermissionParams{
		Name:        req.Name,
		Resource:    req.Resource,
		Action:      req.Action,
		Description: sql.NullString{String: req.Description, Valid: req.Description != ""},
	})
	if err != nil {
		log.Println("Failed to CreatePermission:", err)
		return nil, err
	}
	
	return &CreatePermissionResponse{
		Permission: &Permission{
			Id:          permission.ID,
			Name:        permission.Name,
			Resource:    permission.Resource,
			Action:      permission.Action,
			Description: permission.Description.String,
			CreatedAt:   timeToDateTime(permission.CreatedAt),
		},
	}, nil
}

func (s *ImRBACServiceServer) GetPermission(ctx context.Context, req *GetPermissionRequest) (*GetPermissionResponse, error) {
	log.Println("GetPermission request:", req)
	
	permission, err := s.q.GetPermission(ctx, req.Id)
	if err != nil {
		log.Println("Failed to GetPermission:", err)
		return nil, err
	}
	
	return &GetPermissionResponse{
		Permission: &Permission{
			Id:          permission.ID,
			Name:        permission.Name,
			Resource:    permission.Resource,
			Action:      permission.Action,
			Description: permission.Description.String,
			CreatedAt:   timeToDateTime(permission.CreatedAt),
		},
	}, nil
}

func (s *ImRBACServiceServer) GetPermissionByName(ctx context.Context, req *GetPermissionByNameRequest) (*GetPermissionByNameResponse, error) {
	log.Println("GetPermissionByName request:", req)
	
	permission, err := s.q.GetPermissionByName(ctx, req.Name)
	if err != nil {
		log.Println("Failed to GetPermissionByName:", err)
		return nil, err
	}
	
	return &GetPermissionByNameResponse{
		Permission: &Permission{
			Id:          permission.ID,
			Name:        permission.Name,
			Resource:    permission.Resource,
			Action:      permission.Action,
			Description: permission.Description.String,
			CreatedAt:   timeToDateTime(permission.CreatedAt),
		},
	}, nil
}

func (s *ImRBACServiceServer) ListPermissions(ctx context.Context, req *ListPermissionsRequest) (*ListPermissionsResponse, error) {
	log.Println("ListPermissions request")
	
	permissions, err := s.q.ListPermissions(ctx)
	if err != nil {
		log.Println("Failed to ListPermissions:", err)
		return nil, err
	}
	
	protoPermissions := make([]*Permission, len(permissions))
	for i, perm := range permissions {
		protoPermissions[i] = &Permission{
			Id:          perm.ID,
			Name:        perm.Name,
			Resource:    perm.Resource,
			Action:      perm.Action,
			Description: perm.Description.String,
			CreatedAt:   timeToDateTime(perm.CreatedAt),
		}
	}
	
	return &ListPermissionsResponse{Permissions: protoPermissions}, nil
}

func (s *ImRBACServiceServer) UpdatePermission(ctx context.Context, req *UpdatePermissionRequest) (*UpdatePermissionResponse, error) {
	log.Println("UpdatePermission request:", req)
	
	err := s.q.UpdatePermission(ctx, UpdatePermissionParams{
		ID:          req.Id,
		Name:        req.Name,
		Resource:    req.Resource,
		Action:      req.Action,
		Description: sql.NullString{String: req.Description, Valid: req.Description != ""},
	})
	if err != nil {
		log.Println("Failed to UpdatePermission:", err)
		return nil, err
	}
	
	// Fetch updated permission
	permission, err := s.q.GetPermission(ctx, req.Id)
	if err != nil {
		log.Println("Failed to fetch updated permission:", err)
		return nil, err
	}
	
	return &UpdatePermissionResponse{
		Permission: &Permission{
			Id:          permission.ID,
			Name:        permission.Name,
			Resource:    permission.Resource,
			Action:      permission.Action,
			Description: permission.Description.String,
			CreatedAt:   timeToDateTime(permission.CreatedAt),
		},
	}, nil
}

func (s *ImRBACServiceServer) DeletePermission(ctx context.Context, req *DeletePermissionRequest) (*DeletePermissionResponse, error) {
	log.Println("DeletePermission request:", req)
	
	err := s.q.DeletePermission(ctx, req.Id)
	if err != nil {
		log.Println("Failed to DeletePermission:", err)
		return &DeletePermissionResponse{Success: false}, err
	}
	
	return &DeletePermissionResponse{Success: true}, nil
}

// Role-Permission Assignment

func (s *ImRBACServiceServer) AssignPermissionToRole(ctx context.Context, req *AssignPermissionToRoleRequest) (*AssignPermissionToRoleResponse, error) {
	log.Println("AssignPermissionToRole request:", req)
	
	err := s.q.AssignPermissionToRole(ctx, AssignPermissionToRoleParams{
		RoleID:       req.RoleId,
		PermissionID: req.PermissionId,
	})
	if err != nil {
		log.Println("Failed to AssignPermissionToRole:", err)
		return &AssignPermissionToRoleResponse{Success: false}, err
	}
	
	return &AssignPermissionToRoleResponse{Success: true}, nil
}

func (s *ImRBACServiceServer) RevokePermissionFromRole(ctx context.Context, req *RevokePermissionFromRoleRequest) (*RevokePermissionFromRoleResponse, error) {
	log.Println("RevokePermissionFromRole request:", req)
	
	err := s.q.RevokePermissionFromRole(ctx, RevokePermissionFromRoleParams{
		RoleID:       req.RoleId,
		PermissionID: req.PermissionId,
	})
	if err != nil {
		log.Println("Failed to RevokePermissionFromRole:", err)
		return &RevokePermissionFromRoleResponse{Success: false}, err
	}
	
	return &RevokePermissionFromRoleResponse{Success: true}, nil
}

func (s *ImRBACServiceServer) GetRolePermissions(ctx context.Context, req *GetRolePermissionsRequest) (*GetRolePermissionsResponse, error) {
	log.Println("GetRolePermissions request:", req)
	
	permissions, err := s.q.GetRolePermissions(ctx, req.RoleId)
	if err != nil {
		log.Println("Failed to GetRolePermissions:", err)
		return nil, err
	}
	
	protoPermissions := make([]*Permission, len(permissions))
	for i, perm := range permissions {
		protoPermissions[i] = &Permission{
			Id:          perm.ID,
			Name:        perm.Name,
			Resource:    perm.Resource,
			Action:      perm.Action,
			Description: perm.Description.String,
			CreatedAt:   timeToDateTime(perm.CreatedAt),
		}
	}
	
	return &GetRolePermissionsResponse{Permissions: protoPermissions}, nil
}

// User-Role Assignment

func (s *ImRBACServiceServer) AssignRoleToUser(ctx context.Context, req *AssignRoleToUserRequest) (*AssignRoleToUserResponse, error) {
	log.Println("AssignRoleToUser request:", req)
	
	var expiresAt sql.NullTime
	if req.ExpiresAt != nil {
		expiresAt = sql.NullTime{
			Time:  dateTimeToTime(req.ExpiresAt),
			Valid: true,
		}
	}
	
	err := s.q.AssignRoleToUser(ctx, AssignRoleToUserParams{
		UserID:    req.UserId,
		RoleID:    req.RoleId,
		GrantedBy: sql.NullInt32{Int32: req.GrantedBy, Valid: req.GrantedBy > 0},
		ExpiresAt: expiresAt,
	})
	if err != nil {
		log.Println("Failed to AssignRoleToUser:", err)
		return &AssignRoleToUserResponse{Success: false}, err
	}
	
	return &AssignRoleToUserResponse{Success: true}, nil
}

func (s *ImRBACServiceServer) RevokeRoleFromUser(ctx context.Context, req *RevokeRoleFromUserRequest) (*RevokeRoleFromUserResponse, error) {
	log.Println("RevokeRoleFromUser request:", req)
	
	err := s.q.RevokeRoleFromUser(ctx, RevokeRoleFromUserParams{
		UserID: req.UserId,
		RoleID: req.RoleId,
	})
	if err != nil {
		log.Println("Failed to RevokeRoleFromUser:", err)
		return &RevokeRoleFromUserResponse{Success: false}, err
	}
	
	return &RevokeRoleFromUserResponse{Success: true}, nil
}

func (s *ImRBACServiceServer) GetUserRoles(ctx context.Context, req *GetUserRolesRequest) (*GetUserRolesResponse, error) {
	log.Println("GetUserRoles request:", req)
	
	roles, err := s.q.GetUserRoles(ctx, req.UserId)
	if err != nil {
		log.Println("Failed to GetUserRoles:", err)
		return nil, err
	}
	
	protoRoles := make([]*Role, len(roles))
	for i, role := range roles {
		protoRoles[i] = &Role{
			Id:          role.ID,
			Name:        role.Name,
			Description: role.Description.String,
			CreatedAt:   timeToDateTime(role.CreatedAt),
		}
	}
	
	return &GetUserRolesResponse{Roles: protoRoles}, nil
}

func (s *ImRBACServiceServer) GetUserRoleDetails(ctx context.Context, req *GetUserRoleDetailsRequest) (*GetUserRoleDetailsResponse, error) {
	log.Println("GetUserRoleDetails request:", req)
	
	roleDetails, err := s.q.GetUserRoleDetails(ctx, req.UserId)
	if err != nil {
		log.Println("Failed to GetUserRoleDetails:", err)
		return nil, err
	}
	
	userRoles := make([]*UserRole, len(roleDetails))
	for i, rd := range roleDetails {
		userRoles[i] = &UserRole{
			UserId: req.UserId,
			Role: &Role{
				Id:          rd.RoleID,
				Name:        rd.RoleName,
				Description: rd.RoleDescription.String,
			},
			GrantedBy: rd.GrantedBy.Int32,
			GrantedAt: timeToDateTime(rd.GrantedAt),
			ExpiresAt: nullTimeToDateTime(rd.ExpiresAt),
		}
	}
	
	return &GetUserRoleDetailsResponse{UserRoles: userRoles}, nil
}

func (s *ImRBACServiceServer) GetUsersWithRole(ctx context.Context, req *GetUsersWithRoleRequest) (*GetUsersWithRoleResponse, error) {
	log.Println("GetUsersWithRole request:", req)
	
	users, err := s.q.GetUsersWithRole(ctx, req.RoleId)
	if err != nil {
		log.Println("Failed to GetUsersWithRole:", err)
		return nil, err
	}
	
	userRoles := make([]*UserRole, len(users))
	for i, u := range users {
		userRoles[i] = &UserRole{
			UserId:    u.UserID,
			GrantedBy: u.GrantedBy.Int32,
			GrantedAt: timeToDateTime(u.GrantedAt),
			ExpiresAt: nullTimeToDateTime(u.ExpiresAt),
		}
	}
	
	return &GetUsersWithRoleResponse{UserRoles: userRoles}, nil
}

// Authorization

func (s *ImRBACServiceServer) CheckPermission(ctx context.Context, req *CheckPermissionRequest) (*CheckPermissionResponse, error) {
	log.Println("CheckPermission request:", req)
	
	result, err := s.q.CheckUserPermission(ctx, CheckUserPermissionParams{
		UserID: req.UserId,
		Name:   req.PermissionName,
	})
	if err != nil {
		log.Println("Failed to CheckPermission:", err)
		return &CheckPermissionResponse{
			Authorized: false,
			Reason:     fmt.Sprintf("Error checking permission: %v", err),
		}, nil
	}
	
	reason := "Permission granted"
	if !result {
		reason = fmt.Sprintf("User %d does not have permission '%s'", req.UserId, req.PermissionName)
	}
	
	return &CheckPermissionResponse{
		Authorized: result,
		Reason:     reason,
	}, nil
}

func (s *ImRBACServiceServer) CheckPermissionByResourceAction(ctx context.Context, req *CheckPermissionByResourceActionRequest) (*CheckPermissionByResourceActionResponse, error) {
	log.Println("CheckPermissionByResourceAction request:", req)
	
	result, err := s.q.CheckUserPermissionByResourceAction(ctx, CheckUserPermissionByResourceActionParams{
		UserID:   req.UserId,
		Resource: req.Resource,
		Action:   req.Action,
	})
	if err != nil {
		log.Println("Failed to CheckPermissionByResourceAction:", err)
		return &CheckPermissionByResourceActionResponse{
			Authorized: false,
			Reason:     fmt.Sprintf("Error checking permission: %v", err),
		}, nil
	}
	
	reason := "Permission granted"
	if !result {
		reason = fmt.Sprintf("User %d does not have permission for resource '%s' action '%s'", req.UserId, req.Resource, req.Action)
	}
	
	return &CheckPermissionByResourceActionResponse{
		Authorized: result,
		Reason:     reason,
	}, nil
}

func (s *ImRBACServiceServer) GetUserPermissions(ctx context.Context, req *GetUserPermissionsRequest) (*GetUserPermissionsResponse, error) {
	log.Println("GetUserPermissions request:", req)
	
	permissions, err := s.q.GetUserPermissions(ctx, req.UserId)
	if err != nil {
		log.Println("Failed to GetUserPermissions:", err)
		return nil, err
	}
	
	protoPermissions := make([]*Permission, len(permissions))
	for i, perm := range permissions {
		protoPermissions[i] = &Permission{
			Id:          perm.ID,
			Name:        perm.Name,
			Resource:    perm.Resource,
			Action:      perm.Action,
			Description: perm.Description.String,
			CreatedAt:   timeToDateTime(perm.CreatedAt),
		}
	}
	
	return &GetUserPermissionsResponse{Permissions: protoPermissions}, nil
}

