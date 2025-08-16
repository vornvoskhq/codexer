package db

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

var BaseDir string

func init() {
	home, err := os.UserHomeDir()
	if err != nil {
		panic(fmt.Errorf("error getting user home dir: %v", err))
	}

	log.Println("Plandex server home dir:", home)
	log.Println("os.Getenv(PLANDEX_BASE_DIR):", os.Getenv("PLANDEX_BASE_DIR"))
	log.Println("GOENV:", os.Getenv("GOENV"))
	if os.Getenv("GOENV") == "development" && os.Getenv("LOCAL_MODE") == "1" {
		log.Println("Local mode enabled")
	}

	BaseDir = os.Getenv("PLANDEX_BASE_DIR")

	if BaseDir == "" {
		if os.Getenv("GOENV") == "development" {
			BaseDir = filepath.Join(home, "plandex-server")
		} else {
			BaseDir = "/plandex-server"
		}
	}

	log.Printf("File system dir: %v\n", BaseDir)
}

func InitPlan(orgId, planId string) error {
	dir := getPlanDir(orgId, planId)
	err := os.MkdirAll(dir, os.ModePerm)

	if err != nil {
		return fmt.Errorf("error creating plan dir: %v", err)
	}

	for _, subdirFn := range [](func(orgId, planId string) string){
		getPlanContextDir,
		getPlanConversationDir,
		getPlanResultsDir,
		getPlanDescriptionsDir} {
		err = os.MkdirAll(subdirFn(orgId, planId), os.ModePerm)

		if err != nil {
			return fmt.Errorf("error creating plan subdir: %v", err)
		}
	}

	err = InitGitRepo(orgId, planId)

	if err != nil {
		return fmt.Errorf("error initializing git repo: %v", err)
	}

	return nil
}

func DeletePlanDir(orgId, planId string) error {
	dir := getPlanDir(orgId, planId)
	err := os.RemoveAll(dir)

	if err != nil {
		return fmt.Errorf("error deleting plan dir: %v", err)
	}

	return nil
}

func getOrgDir(orgId string) string {
	return filepath.Join(BaseDir, "orgs", orgId)
}

func getProjectDir(orgId, projectId string) string {
	return filepath.Join(getOrgDir(orgId), "projects", projectId)
}

func getProjectMapCacheDir(orgId, projectId string) string {
	return filepath.Join(getProjectDir(orgId, projectId), "map_cache")
}

func getPlanDir(orgId, planId string) string {
	return filepath.Join(getOrgDir(orgId), "plans", planId)
}

func getPlanContextDir(orgId, planId string) string {
	return filepath.Join(getPlanDir(orgId, planId), "context")
}

func getPlanConversationDir(orgId, planId string) string {
	return filepath.Join(getPlanDir(orgId, planId), "conversation")
}

func getPlanResultsDir(orgId, planId string) string {
	return filepath.Join(getPlanDir(orgId, planId), "results")
}

func getPlanAppliesDir(orgId, planId string) string {
	return filepath.Join(getPlanDir(orgId, planId), "applies")
}

func getPlanDescriptionsDir(orgId, planId string) string {
	return filepath.Join(getPlanDir(orgId, planId), "descriptions")
}
