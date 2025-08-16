package cmd

import (
	"fmt"
	"os"
	"plandex-cli/auth"
	"plandex-cli/lib"
	"plandex-cli/term"
	"plandex-cli/types"

	"github.com/sashabaranov/go-openai"
	"github.com/spf13/cobra"
)

var (
	recursive       bool
	namesOnly       bool
	note            string
	forceSkipIgnore bool
	imageDetail     string
	defsOnly        bool
)

var contextLoadCmd = &cobra.Command{
	Use:     "load [files-or-urls...]",
	Aliases: []string{"l", "add"},
	Short:   "Load context from various inputs",
	Long:    `Load context from a file path, a directory, a URL, an image, a note, or piped data.`,
	Run:     contextLoad,
}

func init() {
	contextLoadCmd.Flags().StringVarP(&note, "note", "n", "", "Add a note to the context")
	contextLoadCmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "Search directories recursively")
	contextLoadCmd.Flags().BoolVar(&namesOnly, "tree", false, "Load directory tree with file names only")
	contextLoadCmd.Flags().BoolVarP(&forceSkipIgnore, "force", "f", false, "Load files even when ignored by .gitignore or .plandexignore")
	contextLoadCmd.Flags().StringVarP(&imageDetail, "detail", "d", "high", "Image detail level (high or low)")
	contextLoadCmd.Flags().BoolVar(&defsOnly, "map", false, "Load file maps (function/method/class signatures, variable names, types, etc.)")
	RootCmd.AddCommand(contextLoadCmd)
}

func contextLoad(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()
	lib.MustResolveProject()

	if lib.CurrentPlanId == "" {
		term.OutputNoCurrentPlanErrorAndExit()
		return
	}

	lib.MustLoadContext(args, &types.LoadContextParams{
		Note:            note,
		Recursive:       recursive,
		NamesOnly:       namesOnly,
		ForceSkipIgnore: forceSkipIgnore,
		ImageDetail:     openai.ImageURLDetail(imageDetail),
		DefsOnly:        defsOnly,
		SessionId:       os.Getenv("PLANDEX_REPL_SESSION_ID"),
	})

	fmt.Println()
	term.PrintCmds("", "ls", "tell", "debug")
}
