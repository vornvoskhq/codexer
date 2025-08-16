package db

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	shared "plandex-shared"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

const orgFields = "id, name, domain, auto_add_domain_users, owner_id, is_trial, created_at, updated_at"

func GetAccessibleOrgsForUser(user *User) ([]*Org, error) {
	// direct access
	var orgUsers []*OrgUser
	var orgs []*Org

	err := Conn.Select(&orgUsers, "SELECT * FROM orgs_users WHERE user_id = $1", user.Id)
	if err != nil {
		return nil, fmt.Errorf("error getting orgs for user: %v", err)
	}

	orgRoleIdByOrgId := map[string]string{}
	orgIds := []string{}
	for _, ou := range orgUsers {
		orgIds = append(orgIds, ou.OrgId)
		orgRoleIdByOrgId[ou.OrgId] = ou.OrgRoleId
	}

	if len(orgIds) > 0 {
		query := fmt.Sprintf("SELECT %s FROM orgs WHERE id = ANY($1)", orgFields)
		err = Conn.Select(&orgs, query, pq.Array(orgIds))
		if err != nil {
			return nil, fmt.Errorf("error getting orgs for user: %v", err)
		}
	} else {
		log.Println("No orgs found for user")
		return orgs, nil
	}

	// access via invitation
	invites, err := GetPendingInvitesForEmail(user.Email)
	if err != nil {
		return nil, fmt.Errorf("error getting invites for user: %v", err)
	}

	orgIds = []string{}
	for _, invite := range invites {
		orgIds = append(orgIds, invite.OrgId)
		orgRoleIdByOrgId[invite.OrgId] = invite.OrgRoleId
	}

	if len(orgIds) > 0 {
		var orgsFromInvites []*Org
		query := fmt.Sprintf("SELECT %s FROM orgs WHERE id = ANY($1)", orgFields)
		err = Conn.Select(&orgsFromInvites, query, pq.Array(orgIds))
		if err != nil {
			return nil, fmt.Errorf("error getting orgs from invites: %v", err)
		}
		orgs = append(orgs, orgsFromInvites...)
	}

	return orgs, nil
}
func GetOrg(orgId string) (*Org, error) {
	var org Org
	query := fmt.Sprintf("SELECT %s FROM orgs WHERE id = $1", orgFields)
	err := Conn.Get(&org, query, orgId)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("org not found")
		}

		return nil, fmt.Errorf("error getting org: %v", err)
	}

	return &org, nil
}

func ValidateOrgMembership(userId string, orgId string) (bool, error) {
	var count int
	err := Conn.QueryRow("SELECT COUNT(*) FROM orgs_users WHERE user_id = $1 AND org_id = $2", userId, orgId).Scan(&count)

	if err != nil {
		return false, fmt.Errorf("error validating org membership: %v", err)
	}

	return count > 0, nil
}

func CreateOrg(req *shared.CreateOrgRequest, userId string, domain *string, tx *sqlx.Tx) (*Org, error) {
	org := &Org{
		Name:               req.Name,
		Domain:             domain,
		AutoAddDomainUsers: req.AutoAddDomainUsers,
		OwnerId:            userId,
	}

	err := tx.QueryRow("INSERT INTO orgs (name, domain, auto_add_domain_users, owner_id, is_trial) VALUES ($1, $2, $3, $4, false) RETURNING id", req.Name, domain, req.AutoAddDomainUsers, userId).Scan(&org.Id)

	if err != nil {
		if IsNonUniqueErr(err) {
			// Handle the uniqueness constraint violation
			return nil, fmt.Errorf("an org with domain %s already exists", *domain)

		}

		return nil, fmt.Errorf("error creating org: %v", err)
	}

	orgOwnerRoleId, err := GetOrgOwnerRoleId()
	if err != nil {
		return nil, fmt.Errorf("error getting org owner role id: %v", err)
	}

	_, err = tx.Exec("INSERT INTO orgs_users (org_id, user_id, org_role_id) VALUES ($1, $2, $3)", org.Id, userId, orgOwnerRoleId)

	if err != nil {
		return nil, fmt.Errorf("error adding org membership: %v", err)
	}

	return org, nil
}

func GetOrgForDomain(domain string) (*Org, error) {
	var org Org
	query := fmt.Sprintf("SELECT %s FROM orgs WHERE domain = $1", orgFields)
	err := Conn.Get(&org, query, domain)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, fmt.Errorf("error getting org for domain: %v", err)
	}

	return &org, nil
}

func AddOrgDomainUsers(orgId, domain string, tx *sqlx.Tx) error {
	usersForDomain, err := GetUsersForDomain(domain)

	if err != nil {
		return fmt.Errorf("error getting users for domain: %v", err)
	}

	orgMemberRoleId, err := GetOrgMemberRoleId()

	if err != nil {
		return fmt.Errorf("error getting org member role id: %v", err)
	}

	if len(usersForDomain) > 0 {
		// create org users for each user
		var valueStrings []string
		var valueArgs []interface{}
		for i, user := range usersForDomain {
			num := i * 3
			valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, $%d)", num+1, num+2, num+3))
			valueArgs = append(valueArgs, orgId, user.Id, orgMemberRoleId)
		}

		// Join all value strings and execute a single query
		stmt := fmt.Sprintf("INSERT INTO orgs_users (org_id, user_id, org_role_id) VALUES %s ON CONFLICT ON CONSTRAINT org_user_unique DO NOTHING", strings.Join(valueStrings, ","))
		_, err = tx.Exec(stmt, valueArgs...)

		if err != nil {
			return fmt.Errorf("error adding org users: %v", err)
		}
	}

	return nil
}

func DeleteOrgUser(orgId, userId string, tx *sqlx.Tx) error {
	log.Printf("Deleting org user, org: %s | user: %s\n", orgId, userId)

	_, err := tx.Exec("DELETE FROM orgs_users WHERE org_id = $1 AND user_id = $2", orgId, userId)

	if err != nil {
		return fmt.Errorf("error deleting org member: %v", err)
	}

	return nil
}

func CreateOrgUser(orgId, userId, orgRoleId string, tx *sqlx.Tx) error {
	query := "INSERT INTO orgs_users (org_id, user_id, org_role_id) VALUES ($1, $2, $3)"
	var err error
	if tx == nil {
		_, err = Conn.Exec(query, orgId, userId, orgRoleId)
	} else {
		_, err = tx.Exec(query, orgId, userId, orgRoleId)
	}

	if err != nil {
		return fmt.Errorf("error adding org member: %v", err)
	}

	return nil
}

func ListOrgRoles(orgId string) ([]*OrgRole, error) {
	var orgRoles []*OrgRole
	err := Conn.Select(&orgRoles, "SELECT * FROM org_roles WHERE org_id IS NULL OR org_id = $1", orgId)

	if err != nil {
		return nil, fmt.Errorf("error listing org roles: %v", err)
	}

	return orgRoles, nil
}

func AddToOrgForDomain(userId, domain string, tx *sqlx.Tx) (string, error) {
	org, err := GetOrgForDomain(domain)

	if err != nil {
		return "", fmt.Errorf("error getting org for domain: %v", err)
	}

	orgOwnerRoleId, err := GetOrgOwnerRoleId()

	if err != nil {
		return "", fmt.Errorf("error getting org owner role id: %v", err)
	}

	if org != nil && org.AutoAddDomainUsers {
		err = CreateOrgUser(org.Id, userId, orgOwnerRoleId, tx)

		if err != nil {
			return "", fmt.Errorf("error adding org user: %v", err)
		}
	}

	var orgId string
	if org != nil {
		orgId = org.Id
	}

	return orgId, nil
}
