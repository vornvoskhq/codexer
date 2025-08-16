package cmd

import (
	"fmt"
	"plandex-cli/api"
	"plandex-cli/auth"
	"plandex-cli/lib"
	"plandex-cli/term"
	"strconv"
	"strings"

	shared "plandex-shared"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

const (
	OptCreateNewBranch = "Create a new branch"
)

var confirmCreateBranch bool

var checkoutCmd = &cobra.Command{
	Use:     "checkout [name-or-index]",
	Aliases: []string{"co"},
	Short:   "Checkout an existing plan branch or create a new one",
	Run:     checkout,
	Args:    cobra.MaximumNArgs(1),
}

func init() {
	RootCmd.AddCommand(checkoutCmd)
	checkoutCmd.Flags().BoolVarP(&confirmCreateBranch, "yes", "y", false, "Confirm creating a new branch")
}

func checkout(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()
	lib.MustResolveProject()

	if lib.CurrentPlanId == "" {
		term.OutputNoCurrentPlanErrorAndExit()
	}

	branchName := ""
	willCreate := false

	var nameOrIdx string
	if len(args) > 0 {
		nameOrIdx = strings.TrimSpace(args[0])
	}

	term.StartSpinner("")
	branches, apiErr := api.Client.ListBranches(lib.CurrentPlanId)
	term.StopSpinner()

	if apiErr != nil {
		term.OutputErrorAndExit("Error getting branches: %v", apiErr)
		return
	}

	if nameOrIdx != "" {
		idx, err := strconv.Atoi(nameOrIdx)

		if err == nil {
			if idx > 0 && idx <= len(branches) {
				branchName = branches[idx-1].Name
			} else {
				term.OutputErrorAndExit("Branch %d not found", idx)
			}
		} else {
			for _, b := range branches {
				if b.Name == nameOrIdx {
					branchName = b.Name
					break
				}
			}
		}

		if branchName == "" {
			fmt.Printf("🌱 Branch %s not found\n", color.New(color.Bold, term.ColorHiCyan).Sprint(nameOrIdx))

			if confirmCreateBranch {
				fmt.Println("✅ --yes flag set, will create branch")
				branchName = nameOrIdx
				willCreate = true
			} else {
				res, err := term.ConfirmYesNo("Create it now?")

				if err != nil {
					term.OutputErrorAndExit("Error getting user input: %v", err)
				}

				if res {
					branchName = nameOrIdx
					willCreate = true
				} else {
					return
				}
			}
		}

	}

	if nameOrIdx == "" {
		opts := make([]string, len(branches))
		for i, branch := range branches {
			opts[i] = branch.Name
		}
		opts = append(opts, OptCreateNewBranch)

		selected, err := term.SelectFromList("Select a branch", opts)

		if err != nil {
			term.OutputErrorAndExit("Error selecting branch: %v", err)
			return
		}

		if selected == OptCreateNewBranch {
			branchName, err = term.GetRequiredUserStringInput("Branch name")
			if err != nil {
				term.OutputErrorAndExit("Error getting branch name: %v", err)
				return
			}
			willCreate = true
		} else {
			branchName = selected
		}
	}

	if branchName == "" {
		term.OutputErrorAndExit("Branch not found")
	}

	term.StartSpinner("")
	if willCreate {
		err := api.Client.CreateBranch(lib.CurrentPlanId, lib.CurrentBranch, shared.CreateBranchRequest{Name: branchName})

		if err != nil {
			term.OutputErrorAndExit("Error creating branch: %v", err)
			return
		}

		// fmt.Printf("✅ Created branch %s\n", color.New(color.Bold, term.ColorHiGreen).Sprint(branchName))
	}

	err := lib.WriteCurrentBranch(branchName)

	if err != nil {
		term.OutputErrorAndExit("Error setting current branch: %v", err)
		return
	}

	updatedModelSettings, err := lib.SaveLatestPlanModelSettingsIfNeeded()
	term.StopSpinner()
	if err != nil {
		term.OutputErrorAndExit("Error saving model settings: %v", err)
	}

	term.StopSpinner()

	fmt.Printf("✅ Checked out branch %s\n", color.New(color.Bold, term.ColorHiGreen).Sprint(branchName))

	if updatedModelSettings {
		fmt.Println()
		fmt.Println("🧠 Model settings file updated → ", lib.GetPlanModelSettingsPath(lib.CurrentPlanId))
	}

	fmt.Println()
	term.PrintCmds("", "load", "tell", "branches", "delete-branch")

}
