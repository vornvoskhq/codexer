package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"plandex-server/db"
	"plandex-server/hooks"

	shared "plandex-shared"

	"github.com/jmoiron/sqlx"
)

func ListOrgsHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for ListOrgsHandler")

	auth := Authenticate(w, r, false)
	if auth == nil {
		return
	}

	orgs, err := db.GetAccessibleOrgsForUser(auth.User)

	if err != nil {
		log.Printf("Error listing orgs: %v\n", err)
		http.Error(w, "Error listing orgs: "+err.Error(), http.StatusInternalServerError)
		return
	}

	apiOrgs, apiErr := toApiOrgs(orgs)

	if apiErr != nil {
		log.Printf("Error converting orgs to api: %v\n", apiErr)
		writeApiError(w, *apiErr)
		return
	}

	bytes, err := json.Marshal(apiOrgs)

	if err != nil {
		log.Printf("Error marshalling response: %v\n", err)
		http.Error(w, "Error marshalling response: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Successfully listed orgs")

	w.Write(bytes)
}

func CreateOrgHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for CreateOrgHandler")

	if os.Getenv("IS_CLOUD") != "" {
		writeApiError(w, shared.ApiError{
			Type:   shared.ApiErrorTypeOther,
			Status: http.StatusForbidden,
			Msg:    "Plandex Cloud orgs can only be created by starting a trial",
		})
		return
	}

	auth := Authenticate(w, r, false)
	if auth == nil {
		return
	}

	// read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading request body: %v\n", err)
		http.Error(w, "Error reading request body: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var req shared.CreateOrgRequest
	err = json.Unmarshal(body, &req)
	if err != nil {
		log.Printf("Error unmarshalling request: %v\n", err)
		http.Error(w, "Error unmarshalling request: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var apiErr *shared.ApiError
	var org *db.Org
	err = db.WithTx(r.Context(), "create org", func(tx *sqlx.Tx) error {
		var err error
		var domain *string
		if req.AutoAddDomainUsers {
			if shared.IsEmailServiceDomain(auth.User.Domain) {
				log.Printf("Invalid domain: %v\n", auth.User.Domain)
				return fmt.Errorf("invalid domain: %v", auth.User.Domain)
			}

			domain = &auth.User.Domain
		}

		// create a new org
		org, err = db.CreateOrg(&req, auth.AuthToken.UserId, domain, tx)

		if err != nil {
			log.Printf("Error creating org: %v\n", err)
			return fmt.Errorf("error creating org: %v", err)
		}

		if org.AutoAddDomainUsers && org.Domain != nil {
			err = db.AddOrgDomainUsers(org.Id, *org.Domain, tx)

			if err != nil {
				log.Printf("Error adding org domain users: %v\n", err)
				return fmt.Errorf("error adding org domain users: %v", err)
			}
		}

		_, apiErr = hooks.ExecHook(hooks.CreateOrg, hooks.HookParams{
			Auth: auth,
			Tx:   tx,

			CreateOrgHookRequestParams: &hooks.CreateOrgHookRequestParams{
				Org: org,
			},
		})

		return nil
	})

	if apiErr != nil {
		writeApiError(w, *apiErr)
		return
	}

	if err != nil {
		log.Printf("Error creating org: %v\n", err)
		http.Error(w, "Error creating org: "+err.Error(), http.StatusInternalServerError)
		return
	}

	resp := shared.CreateOrgResponse{
		Id: org.Id,
	}

	bytes, err := json.Marshal(resp)

	if err != nil {
		log.Printf("Error marshalling response: %v\n", err)
		http.Error(w, "Error marshalling response: "+err.Error(), http.StatusInternalServerError)
		return
	}

	err = SetAuthCookieIfBrowser(w, r, auth.User, "", org.Id)
	if err != nil {
		log.Printf("Error setting auth cookie: %v\n", err)
		http.Error(w, "Error setting auth cookie: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Successfully created org")

	w.Write(bytes)
}

func GetOrgSessionHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for GetOrgSessionHandler")

	auth := Authenticate(w, r, true)

	if auth == nil {
		return
	}

	org, apiErr := getApiOrg(auth.OrgId)

	if apiErr != nil {
		log.Printf("Error converting org to api: %v\n", apiErr)
		writeApiError(w, *apiErr)
		return
	}

	bytes, err := json.Marshal(org)

	if err != nil {
		log.Printf("Error marshalling response: %v\n", err)
		http.Error(w, "Error marshalling response: "+err.Error(), http.StatusInternalServerError)
		return
	}

	err = SetAuthCookieIfBrowser(w, r, auth.User, "", org.Id)
	if err != nil {
		log.Printf("Error setting auth cookie: %v\n", err)
		http.Error(w, "Error setting auth cookie: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(bytes)

	log.Println("Successfully got org session")
}

func ListOrgRolesHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for ListOrgRolesHandler")

	auth := Authenticate(w, r, true)
	if auth == nil {
		return
	}

	org, err := db.GetOrg(auth.OrgId)
	if err != nil {
		log.Printf("Error getting org: %v\n", err)
		http.Error(w, "Error getting org: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if org.IsTrial {
		writeApiError(w, shared.ApiError{
			Type:   shared.ApiErrorTypeTrialActionNotAllowed,
			Status: http.StatusForbidden,
			Msg:    "Trial user can't list org roles",
		})
		return
	}

	if !auth.HasPermission(shared.PermissionListOrgRoles) {
		log.Println("User cannot list org roles")
		http.Error(w, "User cannot list org roles", http.StatusForbidden)
		return
	}

	roles, err := db.ListOrgRoles(auth.OrgId)

	if err != nil {
		log.Printf("Error listing org roles: %v\n", err)
		http.Error(w, "Error listing org roles: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var apiRoles []*shared.OrgRole
	for _, role := range roles {
		apiRoles = append(apiRoles, role.ToApi())
	}

	bytes, err := json.Marshal(apiRoles)

	if err != nil {
		log.Printf("Error marshalling response: %v\n", err)
		http.Error(w, "Error marshalling response: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Successfully listed org roles")

	w.Write(bytes)
}
