package middleware

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

func TestEveryRPCRequiresExplicitPolicy(t *testing.T) {
	paths, err := filepath.Glob("../*/pb/*/*/*.proto")
	if err != nil {
		t.Fatal(err)
	}
	packagePattern := regexp.MustCompile(`package\s+([^;]+);`)
	servicePattern := regexp.MustCompile(`service\s+(\w+)\s*\{`)
	rpcPattern := regexp.MustCompile(`rpc\s+(\w+)\s*\(`)
	declared := map[string]bool{}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		pkg := packagePattern.FindSubmatch(data)
		service := servicePattern.FindSubmatch(data)
		if len(service) == 0 {
			continue
		}
		for _, rpc := range rpcPattern.FindAllSubmatch(data, -1) {
			method := "/" + string(pkg[1]) + "." + string(service[1]) + "/" + string(rpc[1])
			declared[method] = true
			if _, exists := PermissionMapping[method]; !exists {
				t.Errorf("RPC lacks explicit policy: %s", method)
			}
		}
	}
	if len(declared) == 0 {
		t.Fatal("no contracts discovered")
	}
	for method := range PermissionMapping {
		if !declared[method] {
			t.Errorf("policy references missing RPC: %s", method)
		}
	}
	for method := range PublicMethods {
		if !declared[method] {
			t.Errorf("public policy references missing RPC: %s", method)
		}
	}
	if IsPublicMethod("/rbac.v1alpha.RBACService/AssignRoleToUser") {
		t.Fatal("role assignment must not be public")
	}
}
