package shared

import (
	"strings"
)

type Permission string

const (
	PermissionDeleteOrg             Permission = "delete_org"
	PermissionManageEmailDomainAuth Permission = "manage_email_domain_auth"
	PermissionManageBilling         Permission = "manage_billing"
	PermissionInviteUser            Permission = "invite_user"
	PermissionRemoveUser            Permission = "remove_user"
	PermissionSetUserRole           Permission = "set_user_role"
	PermissionListOrgRoles          Permission = "list_org_roles"
	PermissionCreateProject         Permission = "create_project"
	PermissionRenameAnyProject      Permission = "rename_any_project"
	PermissionDeleteAnyProject      Permission = "delete_any_project"
	PermissionCreatePlan            Permission = "create_plan"
	PermissionManageAnyPlanShares   Permission = "manage_any_plan_shares"
	PermissionRenameAnyPlan         Permission = "rename_any_plan"
	PermissionDeleteAnyPlan         Permission = "delete_any_plan"
	PermissionUpdateAnyPlan         Permission = "update_any_plan"
	PermissionArchiveAnyPlan        Permission = "archive_any_plan"
)

type Permissions map[string]bool

func (perms Permissions) HasPermission(permission Permission) bool {
	for p := range perms {
		split := strings.Split(p, "|")
		perm := Permission(split[0])

		if perm == permission {
			return true
		}
	}
	return false
}

func (perms Permissions) HasPermissionForResource(permission Permission, resourceId string) bool {
	for p := range perms {
		split := strings.Split(p, "|")
		perm := Permission(split[0])
		resId := split[1]

		if perm == permission && resId == resourceId {
			return true
		}
	}
	return false
}
