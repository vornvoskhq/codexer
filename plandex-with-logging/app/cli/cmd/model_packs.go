package cmd

import (
	"fmt"
	"os"
	"plandex-cli/api"
	"plandex-cli/auth"
	"plandex-cli/term"

	shared "plandex-shared"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

var customModelPacksOnly bool

var modelPacksCmd = &cobra.Command{
	Use:   "model-packs",
	Short: "List all model packs",
	Run:   listModelPacks,
}

var createModelPackCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a model pack",
	Run:   customModelsNotImplemented,
}

var deleteModelPackCmd = &cobra.Command{
	Use:     "delete",
	Aliases: []string{"rm"},
	Short:   "Delete a model pack by name or index",
	Run:     customModelsNotImplemented,
}

var updateModelPackCmd = &cobra.Command{
	Use:   "update",
	Short: "Update a model pack by name",
	Run:   customModelsNotImplemented,
}

var showModelPackCmd = &cobra.Command{
	Use:   "show [name]",
	Short: "Show a model pack by name",
	Args:  cobra.MaximumNArgs(1),
	Run:   showModelPack,
}

func init() {
	RootCmd.AddCommand(modelPacksCmd)
	modelPacksCmd.AddCommand(createModelPackCmd)
	modelPacksCmd.AddCommand(deleteModelPackCmd)
	modelPacksCmd.AddCommand(updateModelPackCmd)
	modelPacksCmd.AddCommand(showModelPackCmd)
	modelPacksCmd.Flags().BoolVarP(&customModelPacksOnly, "custom", "c", false, "Only show custom model packs")
	modelPacksCmd.Flags().BoolVarP(&allProperties, "all", "a", false, "Show all properties")
}

func listModelPacks(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()

	term.StartSpinner("")
	builtInModelPacks := shared.BuiltInModelPacks

	if auth.Current.IsCloud {
		filtered := []*shared.ModelPack{}
		for _, mp := range builtInModelPacks {
			if mp.LocalProvider == "" {
				filtered = append(filtered, mp)
			}
		}
		builtInModelPacks = filtered
	}

	customModelPacks, err := api.Client.ListModelPacks()
	term.StopSpinner()

	if err != nil {
		term.OutputErrorAndExit("Error fetching model packs: %v", err)
		return
	}

	if !customModelPacksOnly {
		color.New(color.Bold, term.ColorHiCyan).Println("🏠 Built-in Model Packs")
		table := tablewriter.NewWriter(os.Stdout)
		table.SetAutoWrapText(true)
		table.SetRowLine(true)
		table.SetHeader([]string{"Name", "Description"})
		for _, set := range builtInModelPacks {
			table.Append([]string{set.Name, set.Description})
		}
		table.Render()
		fmt.Println()
	}

	if len(customModelPacks) > 0 {
		color.New(color.Bold, term.ColorHiCyan).Println("🛠️  Custom Model Packs")
		table := tablewriter.NewWriter(os.Stdout)
		table.SetAutoWrapText(true)
		table.SetRowLine(true)
		table.SetHeader([]string{"#", "Name", "Description"})
		for i, set := range customModelPacks {
			table.Append([]string{fmt.Sprintf("%d", i+1), set.Name, set.Description})
		}
		table.Render()

		fmt.Println()
	} else if customModelPacksOnly {
		fmt.Println("🤷‍♂️ No custom model packs")
		fmt.Println()
	}

	if customModelPacksOnly && len(customModelPacks) > 0 {
		term.PrintCmds("", "model-packs show", "models custom")
	} else if len(customModelPacks) > 0 {
		term.PrintCmds("", "model-packs --custom", "model-packs show", "models custom")
	} else {
		term.PrintCmds("", "model-packs show", "models custom")
	}

}

func showModelPack(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()

	term.StartSpinner("")
	customModelPacks, apiErr := api.Client.ListModelPacks()
	if apiErr != nil {
		term.OutputErrorAndExit("Error fetching models: %v", apiErr)
	}
	customModels, err := api.Client.ListCustomModels()
	if err != nil {
		term.OutputErrorAndExit("Error fetching custom models: %v", err)
	}
	customModelsById := make(map[shared.ModelId]*shared.CustomModel)
	for _, m := range customModels {
		customModelsById[m.ModelId] = m
	}

	term.StopSpinner()

	modelPacks := []*shared.ModelPack{}
	modelPacks = append(modelPacks, customModelPacks...)

	builtInModelPacks := shared.BuiltInModelPacks
	if auth.Current.IsCloud {
		filtered := []*shared.ModelPack{}
		for _, mp := range builtInModelPacks {
			if mp.LocalProvider == "" {
				filtered = append(filtered, mp)
			}
		}
		builtInModelPacks = filtered
	}
	modelPacks = append(modelPacks, builtInModelPacks...)

	var name string
	if len(args) > 0 {
		name = args[0]
	}

	var modelPack *shared.ModelPack

	if name == "" {
		opts := make([]string, len(modelPacks))
		for i, mp := range modelPacks {
			opts[i] = mp.Name
		}

		selected, err := term.SelectFromList("Select a model pack:", opts)
		if err != nil {
			term.OutputErrorAndExit("Error selecting model pack: %v", err)
		}

		for _, mp := range modelPacks {
			if mp.Name == selected {
				modelPack = mp
				break
			}
		}
	} else {
		for _, mp := range modelPacks {
			if mp.Name == name {
				modelPack = mp
				break
			}
		}
	}

	if modelPack == nil {
		term.OutputErrorAndExit("Model pack not found")
		return
	}

	renderModelPack(modelPack, customModelsById, allProperties)

	fmt.Println()

	term.PrintCmds("", "set-model", "set-model default", "models custom")
}

// func getModelRoleConfig(customModels []*shared.CustomModel, modelRole shared.ModelRole) shared.ModelRoleConfig {
// 	_, modelConfig := getModelWithRoleConfig(customModels, modelRole)
// 	return modelConfig
// }

// func getModelWithRoleConfig(customModels []*shared.CustomModel, modelRole shared.ModelRole) (*shared.CustomModel, shared.ModelRoleConfig) {
// 	role := string(modelRole)

// 	modelId := getModelIdForRole(customModels, modelRole)

// 	temperatureStr, err := term.GetUserStringInputWithDefault("Temperature for "+role+":", fmt.Sprintf("%.1f", shared.DefaultConfigByRole[modelRole].Temperature))
// 	if err != nil {
// 		term.OutputErrorAndExit("Error reading temperature: %v", err)
// 	}
// 	temperature, err := strconv.ParseFloat(temperatureStr, 32)
// 	if err != nil {
// 		term.OutputErrorAndExit("Invalid number for temperature: %v", err)
// 	}

// 	topPStr, err := term.GetUserStringInputWithDefault("Top P for "+role+":", fmt.Sprintf("%.1f", shared.DefaultConfigByRole[modelRole].TopP))
// 	if err != nil {
// 		term.OutputErrorAndExit("Error reading top P: %v", err)
// 	}
// 	topP, err := strconv.ParseFloat(topPStr, 32)
// 	if err != nil {
// 		term.OutputErrorAndExit("Invalid number for top P: %v", err)
// 	}

// 	var reservedOutputTokens int
// 	if modelRole == shared.ModelRoleBuilder || modelRole == shared.ModelRolePlanner || modelRole == shared.ModelRoleWholeFileBuilder {
// 		reservedOutputTokensStr, err := term.GetUserStringInputWithDefault("Reserved output tokens for "+role+":", fmt.Sprintf("%d", model.ReservedOutputTokens))
// 		if err != nil {
// 			term.OutputErrorAndExit("Error reading reserved output tokens: %v", err)
// 		}
// 		reservedOutputTokens, err = strconv.Atoi(reservedOutputTokensStr)
// 		if err != nil {
// 			term.OutputErrorAndExit("Invalid number for reserved output tokens: %v", err)
// 		}
// 	}

// 	return model, shared.ModelRoleConfig{
// 		ModelId:              model.ModelId,
// 		Role:                 modelRole,
// 		Temperature:          float32(temperature),
// 		TopP:                 float32(topP),
// 		ReservedOutputTokens: reservedOutputTokens,
// 	}
// }

// func getPlannerRoleConfig(customModels []*shared.CustomModel) shared.PlannerRoleConfig {
// 	model, modelConfig := getModelWithRoleConfig(customModels, shared.ModelRolePlanner)

// 	return shared.PlannerRoleConfig{
// 		ModelRoleConfig: modelConfig,
// 		PlannerModelConfig: shared.PlannerModelConfig{
// 			MaxConvoTokens: model.DefaultMaxConvoTokens,
// 		},
// 	}
// }

// func getModelIdForRole(customModels []*shared.CustomModel, role shared.ModelRole) shared.ModelId {
// 	color.New(color.Bold).Printf("Select a model for the %s role 👇\n", role)
// 	return lib.SelectModelIdForRole(customModels, role)
// }
