package db

import (
	"context"
	"database/sql"
	"fmt"

	shared "plandex-shared"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

func CreateBranch(repo *GitRepo, plan *Plan, parentBranch *Branch, name string, tx *sqlx.Tx) (*Branch, error) {

	query := `INSERT INTO branches (org_id, owner_id, plan_id, parent_branch_id, name, status, context_tokens, convo_tokens) 
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	RETURNING id, created_at, updated_at`

	var (
		contextTokens  int
		convoTokens    int
		parentBranchId *string
	)

	if parentBranch != nil {
		parentBranchId = &parentBranch.Id

		contextTokens = parentBranch.ContextTokens
		convoTokens = parentBranch.ConvoTokens
	}

	branch := &Branch{
		OrgId:          plan.OrgId,
		OwnerId:        plan.OwnerId,
		PlanId:         plan.Id,
		ParentBranchId: parentBranchId,
		Name:           name,
		Status:         shared.PlanStatusDraft,
	}

	var err error

	if tx == nil {
		err = Conn.QueryRow(
			query,
			branch.OrgId,
			branch.OwnerId,
			branch.PlanId,
			branch.ParentBranchId,
			branch.Name,
			branch.Status,
			contextTokens,
			convoTokens,
		).Scan(
			&branch.Id,
			&branch.CreatedAt,
			&branch.UpdatedAt,
		)
	} else {
		err = tx.QueryRow(
			query,
			branch.OrgId,
			branch.OwnerId,
			branch.PlanId,
			branch.ParentBranchId,
			branch.Name,
			branch.Status,
			contextTokens,
			convoTokens,
		).Scan(
			&branch.Id,
			&branch.CreatedAt,
			&branch.UpdatedAt,
		)
	}

	if err != nil {
		return nil, fmt.Errorf("error creating branch: %v", err)
	}

	// Create the git branch (except for main, which is created by default on repo init)
	if name != "main" {
		// parentBranchName := "main"
		// if parentBranch != nil {
		// 	parentBranchName = parentBranch.Name
		// }

		err = repo.GitCreateBranch(name)

		if err != nil {
			return nil, fmt.Errorf("error creating git branch: %v", err)
		}
	}

	err = IncActiveBranches(plan.Id, 1, tx)

	if err != nil {
		return nil, fmt.Errorf("error incrementing active branches: %v", err)
	}

	return branch, nil
}

func GetDbBranch(planId, name string) (*Branch, error) {
	var branch Branch
	err := Conn.Get(&branch, "SELECT * FROM branches WHERE plan_id = $1 AND name = $2", planId, name)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, fmt.Errorf("error getting branch: %v", err)
	}

	return &branch, nil
}

func ListPlanBranches(repo *GitRepo, planId string) ([]*Branch, error) {
	var branches []*Branch
	err := Conn.Select(&branches, "SELECT * FROM branches WHERE plan_id = $1 ORDER BY created_at", planId)

	if err != nil {
		return nil, fmt.Errorf("error listing branches: %v", err)
	}

	// log.Println("branches: ", spew.Sdump(branches))

	gitBranches, err := repo.GitListBranches()

	if err != nil {
		return nil, fmt.Errorf("error listing git branches: %v", err)
	}

	// log.Println("gitBranches: ", spew.Sdump(gitBranches))

	var nameSet = make(map[string]bool)
	for _, name := range gitBranches {
		nameSet[name] = true
	}

	var res []*Branch
	for _, branch := range branches {
		if nameSet[branch.Name] {
			res = append(res, branch)
		}
	}

	return res, nil
}

func ListBranchesForPlans(orgId string, planIds []string) ([]*Branch, error) {
	var branches []*Branch
	err := Conn.Select(&branches, "SELECT * FROM branches WHERE plan_id = ANY($1) ORDER BY created_at", pq.Array(planIds))

	if err != nil {
		return nil, fmt.Errorf("error listing branches: %v", err)
	}

	return branches, nil
}

func DeleteBranch(ctx context.Context, repo *GitRepo, planId, branch string) error {
	return WithTx(ctx, "delete branch", func(tx *sqlx.Tx) error {

		_, err := tx.Exec("DELETE FROM branches WHERE plan_id = $1 AND name = $2", planId, branch)

		if err != nil {
			return fmt.Errorf("error deleting branch: %v", err)
		}

		err = IncActiveBranches(planId, -1, tx)

		if err != nil {
			return fmt.Errorf("error decrementing active branches: %v", err)
		}

		err = repo.GitDeleteBranch(branch)

		if err != nil {
			return fmt.Errorf("error deleting branch dir: %v", err)
		}

		return err
	})
}
