// Package middleware — RBAC primitives (M2-D TDD RED stub).
//
// The ProjectRoleResolver interface decouples this package from the
// repository layer: the middleware asks "what role does user X have on
// project Y?" and the service layer supplies an implementation backed by
// (project.owner_id == user_id) or the project_members table. This keeps
// tests trivial — supply a fake resolver, no DB needed.
//
// Implementation lands once rbac_test.go fails for the expected reason
// (undefined: RequireProjectRole, ProjectRoleResolver).
package middleware