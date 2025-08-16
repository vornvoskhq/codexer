package shared

var DailyDriverModelPack ModelPack
var ReasoningModelPack ModelPack
var StrongModelPack ModelPack
var OSSModelPack ModelPack
var CheapModelPack ModelPack

var OpusPlannerModelPack ModelPack

var AnthropicModelPack ModelPack
var OpenAIModelPack ModelPack
var GoogleModelPack ModelPack

var GeminiPlannerModelPack ModelPack
var O3PlannerModelPack ModelPack
var R1PlannerModelPack ModelPack
var PerplexityPlannerModelPack ModelPack

var OllamaExperimentalModelPack ModelPack
var OllamaAdaptiveOssModelPack ModelPack
var OllamaAdaptiveDailyModelPack ModelPack

var BuiltInModelPacks = []*ModelPack{
	&DailyDriverModelPack,
	&ReasoningModelPack,
	&StrongModelPack,
	&CheapModelPack,
	&OSSModelPack,
	&OllamaExperimentalModelPack,
	&OllamaAdaptiveOssModelPack,
	&OllamaAdaptiveDailyModelPack,
	&AnthropicModelPack,
	&OpenAIModelPack,
	&GoogleModelPack,
	&GeminiPlannerModelPack,
	&OpusPlannerModelPack,
	&O3PlannerModelPack,
	&R1PlannerModelPack,
	&PerplexityPlannerModelPack,
}

var BuiltInModelPacksByName = make(map[string]*ModelPack)

var DefaultModelPack *ModelPack = &DailyDriverModelPack

func getModelRoleConfig(role ModelRole, modelId ModelId, fns ...func(*ModelRoleConfigSchema)) ModelRoleConfigSchema {
	c := ModelRoleConfigSchema{
		ModelId: modelId,
	}
	for _, f := range fns {
		f(&c)
	}
	return c
}

func getLargeContextFallback(role ModelRole, modelId ModelId, fns ...func(*ModelRoleConfigSchema)) func(*ModelRoleConfigSchema) {
	return func(c *ModelRoleConfigSchema) {
		n := getModelRoleConfig(role, modelId)
		for _, f := range fns {
			f(&n)
		}
		c.LargeContextFallback = &n
	}
}

func getErrorFallback(role ModelRole, modelId ModelId, fns ...func(*ModelRoleConfigSchema)) func(*ModelRoleConfigSchema) {
	return func(c *ModelRoleConfigSchema) {
		n := getModelRoleConfig(role, modelId)
		for _, f := range fns {
			f(&n)
		}
		c.ErrorFallback = &n
	}
}

func getStrongModelFallback(role ModelRole, modelId ModelId, fns ...func(*ModelRoleConfigSchema)) func(*ModelRoleConfigSchema) {
	return func(c *ModelRoleConfigSchema) {
		n := getModelRoleConfig(role, modelId)
		for _, f := range fns {
			f(&n)
		}
		c.StrongModel = &n
	}
}

var (
	DailyDriverSchema         ModelPackSchema
	ReasoningSchema           ModelPackSchema
	StrongSchema              ModelPackSchema
	OssSchema                 ModelPackSchema
	CheapSchema               ModelPackSchema
	OllamaExperimentalSchema  ModelPackSchema
	OllamaAdaptiveOssSchema   ModelPackSchema
	OllamaAdaptiveDailySchema ModelPackSchema
	AnthropicSchema           ModelPackSchema
	OpenAISchema              ModelPackSchema
	GoogleSchema              ModelPackSchema
	GeminiPlannerSchema       ModelPackSchema
	OpusPlannerSchema         ModelPackSchema
	R1PlannerSchema           ModelPackSchema
	PerplexityPlannerSchema   ModelPackSchema
	O3PlannerSchema           ModelPackSchema
)

var BuiltInModelPackSchemas = []*ModelPackSchema{
	&DailyDriverSchema,
	&ReasoningSchema,
	&StrongSchema,
	&CheapSchema,
	&OssSchema,
	&OllamaExperimentalSchema,
	&OllamaAdaptiveOssSchema,
	&OllamaAdaptiveDailySchema,
	&AnthropicSchema,
	&OpenAISchema,
	&GeminiPlannerSchema,
	&OpusPlannerSchema,
	&O3PlannerSchema,
	&R1PlannerSchema,
	&PerplexityPlannerSchema,
}

func init() {
	defaultBuilder := getModelRoleConfig(ModelRoleBuilder, "openai/o4-mini-medium",
		getStrongModelFallback(ModelRoleBuilder, "openai/o4-mini-high"),
	)

	DailyDriverSchema = ModelPackSchema{
		Name:        "daily-driver",
		Description: "A mix of models from Anthropic, OpenAI, and Google that balances speed, quality, and cost. Supports up to 2M context.",
		ModelPackSchemaRoles: ModelPackSchemaRoles{
			Planner: getModelRoleConfig(ModelRolePlanner, "anthropic/claude-sonnet-4",
				getLargeContextFallback(ModelRolePlanner, "google/gemini-2.5-pro",
					getLargeContextFallback(ModelRolePlanner, "google/gemini-pro-1.5"),
				),
			),
			Architect: Pointer(getModelRoleConfig(ModelRoleArchitect, "anthropic/claude-sonnet-4",
				getLargeContextFallback(ModelRoleArchitect, "google/gemini-2.5-pro",
					getLargeContextFallback(ModelRoleArchitect, "google/gemini-pro-1.5"),
				),
			)),
			Coder: Pointer(getModelRoleConfig(ModelRoleCoder, "anthropic/claude-sonnet-4",
				getLargeContextFallback(ModelRoleCoder, "openai/gpt-4.1"),
			)),
			PlanSummary:      getModelRoleConfig(ModelRolePlanSummary, "openai/o4-mini-low"),
			Builder:          defaultBuilder,
			WholeFileBuilder: Pointer(getModelRoleConfig(ModelRoleWholeFileBuilder, "openai/o4-mini-medium")),
			Namer:            getModelRoleConfig(ModelRoleName, "openai/gpt-4.1-mini"),
			CommitMsg:        getModelRoleConfig(ModelRoleCommitMsg, "openai/gpt-4.1-mini"),
			ExecStatus:       getModelRoleConfig(ModelRoleExecStatus, "openai/o4-mini-low"),
		},
	}

	ReasoningSchema = ModelPackSchema{
		Name:        "reasoning",
		Description: "Like the daily driver, but uses sonnet-4-thinking with reasoning enabled for planning and coding. Supports up to 160k input context.",
		ModelPackSchemaRoles: ModelPackSchemaRoles{
			Planner:     getModelRoleConfig(ModelRolePlanner, "anthropic/claude-sonnet-4-thinking-hidden"),
			Coder:       Pointer(getModelRoleConfig(ModelRoleCoder, "anthropic/claude-sonnet-4-thinking-hidden")),
			PlanSummary: getModelRoleConfig(ModelRolePlanSummary, "openai/o4-mini-low"),
			Builder:     defaultBuilder,
			WholeFileBuilder: Pointer(getModelRoleConfig(ModelRoleWholeFileBuilder,
				"openai/o4-mini-medium")),
			Namer:      getModelRoleConfig(ModelRoleName, "openai/gpt-4.1-mini"),
			CommitMsg:  getModelRoleConfig(ModelRoleCommitMsg, "openai/gpt-4.1-mini"),
			ExecStatus: getModelRoleConfig(ModelRoleExecStatus, "openai/o4-mini-low"),
		},
	}

	StrongSchema = ModelPackSchema{
		Name:        "strong",
		Description: "For difficult tasks where slower responses and builds are ok. Uses o3-high for architecture and planning, claude-sonnet-4 thinking for implementation. Supports up to 160k input context.",
		ModelPackSchemaRoles: ModelPackSchemaRoles{
			Planner:     getModelRoleConfig(ModelRolePlanner, "openai/o3-high"),
			Architect:   Pointer(getModelRoleConfig(ModelRoleArchitect, "openai/o3-high")),
			Coder:       Pointer(getModelRoleConfig(ModelRoleCoder, "anthropic/claude-sonnet-4-thinking-hidden")),
			PlanSummary: getModelRoleConfig(ModelRolePlanSummary, "openai/o4-mini-low"),
			Builder:     getModelRoleConfig(ModelRoleBuilder, "openai/o4-mini-high"),
			WholeFileBuilder: Pointer(getModelRoleConfig(ModelRoleWholeFileBuilder,
				"openai/o4-mini-high")),
			Namer:      getModelRoleConfig(ModelRoleName, "openai/gpt-4.1-mini"),
			CommitMsg:  getModelRoleConfig(ModelRoleCommitMsg, "openai/gpt-4.1-mini"),
			ExecStatus: getModelRoleConfig(ModelRoleExecStatus, "openai/o4-mini-medium"),
		},
	}

	CheapSchema = ModelPackSchema{
		Name:        "cheap",
		Description: "Cost-effective models that can still get the job done for easier tasks. Supports up to 160k context. Uses OpenAI's o4-mini model for planning, GPT-4.1 for coding, and GPT-4.1 Mini for lighter tasks.",
		ModelPackSchemaRoles: ModelPackSchemaRoles{
			Planner:     getModelRoleConfig(ModelRolePlanner, "openai/o4-mini-medium"),
			Coder:       Pointer(getModelRoleConfig(ModelRoleCoder, "openai/gpt-4.1")),
			PlanSummary: getModelRoleConfig(ModelRolePlanSummary, "openai/gpt-4.1-mini"),
			Builder:     getModelRoleConfig(ModelRoleBuilder, "openai/o4-mini-low"),
			WholeFileBuilder: Pointer(getModelRoleConfig(ModelRoleWholeFileBuilder,
				"openai/o4-mini-low")),
			Namer:      getModelRoleConfig(ModelRoleName, "openai/gpt-4.1-mini"),
			CommitMsg:  getModelRoleConfig(ModelRoleCommitMsg, "openai/gpt-4.1-mini"),
			ExecStatus: getModelRoleConfig(ModelRoleExecStatus, "openai/o4-mini-low"),
		},
	}

	OssSchema = ModelPackSchema{
		Name:        "oss",
		Description: "An experimental mix of the best open source models for coding. Supports up to 144k context, 33k per file.",
		ModelPackSchemaRoles: ModelPackSchemaRoles{
			Planner:     getModelRoleConfig(ModelRolePlanner, "deepseek/r1"),
			Coder:       Pointer(getModelRoleConfig(ModelRoleCoder, "deepseek/v3")),
			PlanSummary: getModelRoleConfig(ModelRolePlanSummary, "deepseek/r1-hidden"),
			Builder:     getModelRoleConfig(ModelRoleBuilder, "deepseek/r1-hidden"),
			WholeFileBuilder: Pointer(getModelRoleConfig(ModelRoleWholeFileBuilder,
				"deepseek/r1-hidden")),
			Namer:      getModelRoleConfig(ModelRoleName, "qwen/qwen3-8b-cloud"),
			CommitMsg:  getModelRoleConfig(ModelRoleCommitMsg, "qwen/qwen3-8b-cloud"),
			ExecStatus: getModelRoleConfig(ModelRoleExecStatus, "deepseek/r1-hidden"),
		},
	}

	OllamaExperimentalSchema = ModelPackSchema{
		Name:        "ollama",
		Description: "Ollama experimental local blend. Supports up to 110k context. For now, more for experimentation and benchmarking than getting work done.",
		ModelPackSchemaRoles: ModelPackSchemaRoles{
			LocalProvider: ModelProviderOllama,
			Planner:       getModelRoleConfig(ModelRolePlanner, "qwen/qwen3-32b-local"),
			PlanSummary:   getModelRoleConfig(ModelRolePlanSummary, "mistral/devstral-small"),
			Builder:       getModelRoleConfig(ModelRoleBuilder, "mistral/devstral-small"),
			WholeFileBuilder: Pointer(getModelRoleConfig(ModelRoleWholeFileBuilder,
				"mistral/devstral-small")),
			Namer:      getModelRoleConfig(ModelRoleName, "qwen/qwen3-8b-local"),
			CommitMsg:  getModelRoleConfig(ModelRoleCommitMsg, "qwen/qwen3-8b-local"),
			ExecStatus: getModelRoleConfig(ModelRoleExecStatus, "mistral/devstral-small"),
		},
	}

	// Copy daily driver schema and modify it to use ollama for lighter tasks
	OllamaAdaptiveDailySchema = cloneSchema(DailyDriverSchema)
	OllamaAdaptiveDailySchema.Name = "ollama-daily"
	OllamaAdaptiveDailySchema.Description = "Ollama adaptive/daily-driver blend. Uses 'daily-driver' for heavy lifting, local models for lighter tasks."
	OllamaAdaptiveDailySchema.LocalProvider = ModelProviderOllama
	OllamaAdaptiveDailySchema.PlanSummary = getModelRoleConfig(ModelRolePlanSummary, "mistral/devstral-small")
	OllamaAdaptiveDailySchema.CommitMsg = getModelRoleConfig(ModelRoleCommitMsg, "qwen/qwen3-8b-local")
	OllamaAdaptiveDailySchema.Namer = getModelRoleConfig(ModelRoleName, "qwen/qwen3-8b-local")

	// Copy oss schema and modify it to use ollama for lighter tasks
	OllamaAdaptiveOssSchema = cloneSchema(OssSchema)
	OllamaAdaptiveOssSchema.Name = "ollama-oss"
	OllamaAdaptiveOssSchema.Description = "Ollama adaptive/oss blend. Uses local models for planning and context selection, open source cloud models for implementation and file edits. Supports up to 110k context."
	OllamaAdaptiveOssSchema.LocalProvider = ModelProviderOllama
	OllamaAdaptiveOssSchema.PlanSummary = getModelRoleConfig(ModelRolePlanSummary, "mistral/devstral-small")
	OllamaAdaptiveOssSchema.CommitMsg = getModelRoleConfig(ModelRoleCommitMsg, "qwen/qwen3-8b-local")
	OllamaAdaptiveOssSchema.Namer = getModelRoleConfig(ModelRoleName, "qwen/qwen3-8b-local")

	OpenAISchema = ModelPackSchema{
		Name:        "openai",
		Description: "OpenAI blend. Supports up to 1M context. Uses OpenAI's GPT-4.1 model for heavy lifting, GPT-4.1 Mini for lighter tasks.",
		ModelPackSchemaRoles: ModelPackSchemaRoles{
			Planner:     getModelRoleConfig(ModelRolePlanner, "openai/gpt-4.1"),
			PlanSummary: getModelRoleConfig(ModelRolePlanSummary, "openai/o4-mini-low"),
			Builder:     defaultBuilder,
			WholeFileBuilder: Pointer(getModelRoleConfig(ModelRoleWholeFileBuilder,
				"openai/o4-mini-medium")),
			Namer:      getModelRoleConfig(ModelRoleName, "openai/gpt-4.1-mini"),
			CommitMsg:  getModelRoleConfig(ModelRoleCommitMsg, "openai/gpt-4.1-mini"),
			ExecStatus: getModelRoleConfig(ModelRoleExecStatus, "openai/o4-mini-low"),
		},
	}

	AnthropicSchema = ModelPackSchema{
		Name:        "anthropic",
		Description: "Anthropic blend. Supports up to 180k context. Uses Claude Sonnet 4 for heavy lifting, Claude 3 Haiku for lighter tasks.",
		ModelPackSchemaRoles: ModelPackSchemaRoles{
			Planner:     getModelRoleConfig(ModelRolePlanner, "anthropic/claude-sonnet-4"),
			Coder:       Pointer(getModelRoleConfig(ModelRoleCoder, "anthropic/claude-sonnet-4")),
			PlanSummary: getModelRoleConfig(ModelRolePlanSummary, "anthropic/claude-3.5-haiku"),
			Builder:     getModelRoleConfig(ModelRoleBuilder, "anthropic/claude-sonnet-4"),
			WholeFileBuilder: Pointer(getModelRoleConfig(ModelRoleWholeFileBuilder,
				"anthropic/claude-sonnet-4")),
			Namer:      getModelRoleConfig(ModelRoleName, "anthropic/claude-3.5-haiku"),
			CommitMsg:  getModelRoleConfig(ModelRoleCommitMsg, "anthropic/claude-3.5-haiku"),
			ExecStatus: getModelRoleConfig(ModelRoleExecStatus, "anthropic/claude-sonnet-4"),
		},
	}

	GoogleSchema = ModelPackSchema{
		Name:        "google",
		Description: "Uses Gemini 2.5 Pro for heavy lifting, 2.5 Flash for light tasks. Supports up to 1M input context.",
		ModelPackSchemaRoles: ModelPackSchemaRoles{
			Planner:     getModelRoleConfig(ModelRolePlanner, "google/gemini-2.5-pro"),
			Coder:       Pointer(getModelRoleConfig(ModelRoleCoder, "google/gemini-2.5-flash")),
			PlanSummary: getModelRoleConfig(ModelRolePlanSummary, "google/gemini-2.5-flash"),
			Builder:     getModelRoleConfig(ModelRoleBuilder, "google/gemini-2.5-pro"),
			Namer:       getModelRoleConfig(ModelRoleName, "google/gemini-2.5-flash"),
			CommitMsg:   getModelRoleConfig(ModelRoleCommitMsg, "google/gemini-2.5-flash"),
			ExecStatus:  getModelRoleConfig(ModelRoleExecStatus, "google/gemini-2.5-pro"),
		},
	}

	GeminiPlannerSchema = ModelPackSchema{
		Name:        "gemini-planner",
		Description: "Uses Gemini 2.5 Pro for planning, default models for other roles. Supports up to 1M input context.",
		ModelPackSchemaRoles: ModelPackSchemaRoles{
			Planner: getModelRoleConfig(ModelRolePlanner, "google/gemini-2.5-pro"),
			Coder: Pointer(getModelRoleConfig(ModelRoleCoder, "anthropic/claude-sonnet-4",
				getLargeContextFallback(ModelRoleCoder, "openai/gpt-4.1"),
			)),
			PlanSummary: getModelRoleConfig(ModelRolePlanSummary, "openai/o4-mini-low"),
			Builder:     defaultBuilder,
			WholeFileBuilder: Pointer(getModelRoleConfig(ModelRoleWholeFileBuilder,
				"openai/o4-mini-medium")),
			Namer:      getModelRoleConfig(ModelRoleName, "openai/gpt-4.1-mini"),
			CommitMsg:  getModelRoleConfig(ModelRoleCommitMsg, "openai/gpt-4.1-mini"),
			ExecStatus: getModelRoleConfig(ModelRoleExecStatus, "openai/o4-mini-low"),
		},
	}

	OpusPlannerSchema = ModelPackSchema{
		Name:        "opus-planner",
		Description: "Uses Claude Opus 4 for planning, default models for other roles. Supports up to 180k input context.",
		ModelPackSchemaRoles: ModelPackSchemaRoles{
			Planner: getModelRoleConfig(ModelRolePlanner, "anthropic/claude-opus-4"),
			Coder: Pointer(getModelRoleConfig(ModelRoleCoder, "anthropic/claude-sonnet-4",
				getLargeContextFallback(ModelRoleCoder, "openai/gpt-4.1"),
			)),
			PlanSummary: getModelRoleConfig(ModelRolePlanSummary, "openai/o4-mini-low"),
			Builder:     defaultBuilder,
			WholeFileBuilder: Pointer(getModelRoleConfig(ModelRoleWholeFileBuilder,
				"openai/o4-mini-medium")),
			Namer:      getModelRoleConfig(ModelRoleName, "openai/gpt-4.1-mini"),
			CommitMsg:  getModelRoleConfig(ModelRoleCommitMsg, "openai/gpt-4.1-mini"),
			ExecStatus: getModelRoleConfig(ModelRoleExecStatus, "openai/o4-mini-low"),
		},
	}

	O3PlannerSchema = ModelPackSchema{
		Name:        "o3-planner",
		Description: "Uses Claude Opus 4 for planning, default models for other roles. Supports up to 180k input context.",
		ModelPackSchemaRoles: ModelPackSchemaRoles{
			Planner: getModelRoleConfig(ModelRolePlanner, "anthropic/opus-4"),
			Coder: Pointer(getModelRoleConfig(ModelRoleCoder, "anthropic/claude-sonnet-4",
				getLargeContextFallback(ModelRoleCoder, "openai/gpt-4.1"),
			)),
			PlanSummary: getModelRoleConfig(ModelRolePlanSummary, "openai/o4-mini-low"),
			Builder:     defaultBuilder,
			WholeFileBuilder: Pointer(getModelRoleConfig(ModelRoleWholeFileBuilder,
				"openai/o4-mini-medium")),
			Namer:      getModelRoleConfig(ModelRoleName, "openai/gpt-4.1-mini"),
			CommitMsg:  getModelRoleConfig(ModelRoleCommitMsg, "openai/gpt-4.1-mini"),
			ExecStatus: getModelRoleConfig(ModelRoleExecStatus, "openai/o4-mini-low"),
		},
	}

	O3PlannerSchema = ModelPackSchema{
		Name:        "o3-planner",
		Description: "Uses OpenAI o3-medium for planning, default models for other roles. Supports up to 160k input context.",
		ModelPackSchemaRoles: ModelPackSchemaRoles{
			Planner: getModelRoleConfig(ModelRolePlanner, "openai/o3-medium"),
			Coder: Pointer(getModelRoleConfig(ModelRoleCoder, "anthropic/claude-sonnet-4",
				getLargeContextFallback(ModelRoleCoder, "openai/gpt-4.1"),
			)),
			PlanSummary: getModelRoleConfig(ModelRolePlanSummary, "openai/o4-mini-low"),
			Builder:     defaultBuilder,
			WholeFileBuilder: Pointer(getModelRoleConfig(ModelRoleWholeFileBuilder,
				"openai/o4-mini-medium")),
			Namer:      getModelRoleConfig(ModelRoleName, "openai/gpt-4.1-mini"),
			CommitMsg:  getModelRoleConfig(ModelRoleCommitMsg, "openai/gpt-4.1-mini"),
			ExecStatus: getModelRoleConfig(ModelRoleExecStatus, "openai/o4-mini-low"),
		},
	}

	R1PlannerSchema = ModelPackSchema{
		Name:        "r1-planner",
		Description: "Uses DeepSeek R1 for planning, Qwen for light tasks, and default models for implementation. Supports up to 56k input context.",
		ModelPackSchemaRoles: ModelPackSchemaRoles{
			Planner:     getModelRoleConfig(ModelRolePlanner, "deepseek/r1"),
			Coder:       Pointer(getModelRoleConfig(ModelRoleCoder, "anthropic/claude-sonnet-4")),
			PlanSummary: getModelRoleConfig(ModelRolePlanSummary, "openai/o4-mini-low"),
			Builder:     getModelRoleConfig(ModelRoleBuilder, "openai/o4-mini-medium"),
			WholeFileBuilder: Pointer(getModelRoleConfig(ModelRoleWholeFileBuilder,
				"openai/o4-mini-low")),
			Namer:      getModelRoleConfig(ModelRoleName, "openai/gpt-4.1-mini"),
			CommitMsg:  getModelRoleConfig(ModelRoleCommitMsg, "openai/gpt-4.1-mini"),
			ExecStatus: getModelRoleConfig(ModelRoleExecStatus, "openai/o4-mini-medium"),
		},
	}

	PerplexityPlannerSchema = ModelPackSchema{
		Name:        "perplexity-planner",
		Description: "Uses Perplexity Sonar for planning, Qwen for light tasks, and default models for implementation. Supports up to 97k input context.",
		ModelPackSchemaRoles: ModelPackSchemaRoles{
			Planner:     getModelRoleConfig(ModelRolePlanner, "perplexity/sonar-reasoning"),
			Coder:       Pointer(getModelRoleConfig(ModelRoleCoder, "anthropic/claude-sonnet-4")),
			PlanSummary: getModelRoleConfig(ModelRolePlanSummary, "openai/o4-mini-low"),
			Builder:     getModelRoleConfig(ModelRoleBuilder, "openai/o4-mini-medium"),
			WholeFileBuilder: Pointer(getModelRoleConfig(ModelRoleWholeFileBuilder,
				"openai/o4-mini-low")),
			Namer:      getModelRoleConfig(ModelRoleName, "openai/gpt-4.1-mini"),
			CommitMsg:  getModelRoleConfig(ModelRoleCommitMsg, "openai/gpt-4.1-mini"),
			ExecStatus: getModelRoleConfig(ModelRoleExecStatus, "openai/o4-mini-medium"),
		},
	}

	DailyDriverModelPack = DailyDriverSchema.ToModelPack()
	ReasoningModelPack = ReasoningSchema.ToModelPack()
	StrongModelPack = StrongSchema.ToModelPack()
	CheapModelPack = CheapSchema.ToModelPack()
	OSSModelPack = OssSchema.ToModelPack()
	OllamaExperimentalModelPack = OllamaExperimentalSchema.ToModelPack()
	OllamaAdaptiveOssModelPack = OllamaAdaptiveOssSchema.ToModelPack()
	OllamaAdaptiveDailyModelPack = OllamaAdaptiveDailySchema.ToModelPack()
	AnthropicModelPack = AnthropicSchema.ToModelPack()
	OpenAIModelPack = OpenAISchema.ToModelPack()
	GoogleModelPack = GoogleSchema.ToModelPack()
	GeminiPlannerModelPack = GeminiPlannerSchema.ToModelPack()
	OpusPlannerModelPack = OpusPlannerSchema.ToModelPack()
	R1PlannerModelPack = R1PlannerSchema.ToModelPack()
	PerplexityPlannerModelPack = PerplexityPlannerSchema.ToModelPack()
	O3PlannerModelPack = O3PlannerSchema.ToModelPack()

	BuiltInModelPacks = []*ModelPack{
		&DailyDriverModelPack,
		&ReasoningModelPack,
		&StrongModelPack,
		&CheapModelPack,
		&OSSModelPack,
		&OllamaExperimentalModelPack,
		&OllamaAdaptiveOssModelPack,
		&OllamaAdaptiveDailyModelPack,
		&AnthropicModelPack,
		&OpenAIModelPack,
		&GoogleModelPack,
		&GeminiPlannerModelPack,
		&OpusPlannerModelPack,
		&O3PlannerModelPack,
		&R1PlannerModelPack,
		&PerplexityPlannerModelPack,
	}

	DefaultModelPack = &DailyDriverModelPack

	for _, mp := range BuiltInModelPacks {
		BuiltInModelPacksByName[mp.Name] = mp

		for _, id := range mp.ToModelPackSchema().AllModelIds() {
			if BuiltInBaseModelsById[id] == nil {
				panic("missing base model: " + id)
			}
		}
	}

}

// pointer fields need to be cloned to avoid modifying the original schema
func cloneSchema(schema ModelPackSchema) ModelPackSchema {
	res := schema

	if schema.Architect != nil {
		tmp := *schema.Architect
		res.Architect = &tmp
	}

	if schema.Coder != nil {
		tmp := *schema.Coder
		res.Coder = &tmp
	}
	if schema.WholeFileBuilder != nil {
		tmp := *schema.WholeFileBuilder
		res.WholeFileBuilder = &tmp
	}

	return res
}
