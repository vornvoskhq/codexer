package model

import (
	"context"
	"fmt"
	"log"
	"plandex-server/db"
	"plandex-server/hooks"
	"plandex-server/notify"
	"plandex-server/types"
	shared "plandex-shared"
	"runtime/debug"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/sashabaranov/go-openai"
)

type ModelRequestParams struct {
	Clients       map[string]ClientInfo
	AuthVars      map[string]string
	Auth          *types.ServerAuth
	Plan          *db.Plan
	ModelConfig   *shared.ModelRoleConfig
	Settings      *shared.PlanSettings
	OrgUserConfig *shared.OrgUserConfig
	Purpose       string

	Messages   []types.ExtendedChatMessage
	Prediction string
	Stop       []string
	Tools      []openai.Tool
	ToolChoice *openai.ToolChoice

	EstimatedOutputTokens int // optional

	ModelStreamId  string
	ConvoMessageId string
	BuildId        string
	ModelPackName  string
	SessionId      string

	BeforeReq func()
	AfterReq  func()

	OnStream func(string, string) bool

	WillCacheNumTokens int
}

func ModelRequest(
	ctx context.Context,
	params ModelRequestParams,
) (*types.ModelResponse, error) {
	clients := params.Clients
	authVars := params.AuthVars
	auth := params.Auth
	plan := params.Plan
	messages := params.Messages
	prediction := params.Prediction
	stop := params.Stop
	tools := params.Tools
	toolChoice := params.ToolChoice
	modelConfig := params.ModelConfig
	modelStreamId := params.ModelStreamId
	convoMessageId := params.ConvoMessageId
	buildId := params.BuildId
	modelPackName := params.ModelPackName
	purpose := params.Purpose
	sessionId := params.SessionId
	settings := params.Settings
	orgUserConfig := params.OrgUserConfig
	currentOrgId := auth.OrgId
	currentUserId := auth.User.Id

	if purpose == "" {
		return nil, fmt.Errorf("purpose is required")
	}

	baseModelConfig := modelConfig.GetBaseModelConfig(authVars, settings, orgUserConfig)

	messages = FilterEmptyMessages(messages)
	messages = CheckSingleSystemMessage(modelConfig, baseModelConfig, messages)
	inputTokensEstimate := GetMessagesTokenEstimate(messages...) + TokensPerRequest

	config := modelConfig.GetRoleForInputTokens(inputTokensEstimate, settings)
	modelConfig = &config

	if params.EstimatedOutputTokens != 0 {
		config = modelConfig.GetRoleForOutputTokens(params.EstimatedOutputTokens, settings)
		modelConfig = &config
	}

	log.Println("ModelRequest - modelConfig:")
	spew.Dump(modelConfig)
	log.Println("ModelRequest - baseModelConfig:")
	spew.Dump(baseModelConfig)

	log.Printf("Model config - role: %s, model: %s, max output tokens: %d\n", modelConfig.Role, baseModelConfig.ModelName, baseModelConfig.MaxOutputTokens)

	expectedOutputTokens := baseModelConfig.MaxOutputTokens - inputTokensEstimate
	if params.EstimatedOutputTokens != 0 {
		expectedOutputTokens = params.EstimatedOutputTokens
	}

	_, apiErr := hooks.ExecHook(hooks.WillSendModelRequest, hooks.HookParams{
		Auth: auth,
		Plan: plan,
		WillSendModelRequestParams: &hooks.WillSendModelRequestParams{
			InputTokens:  inputTokensEstimate,
			OutputTokens: expectedOutputTokens,
			ModelName:    baseModelConfig.ModelName,
			ModelId:      baseModelConfig.ModelId,
			ModelTag:     baseModelConfig.ModelTag,
		},
	})

	if apiErr != nil {
		return nil, apiErr
	}

	if params.BeforeReq != nil {
		params.BeforeReq()
	}

	reqStarted := time.Now()

	req := types.ExtendedChatCompletionRequest{
		Model:    baseModelConfig.ModelName,
		Messages: messages,
	}

	if !baseModelConfig.RoleParamsDisabled {
		req.Temperature = modelConfig.Temperature
		req.TopP = modelConfig.TopP
	}

	if len(tools) > 0 {
		req.Tools = tools
	}

	if toolChoice != nil {
		req.ToolChoice = toolChoice
	}

	onStream := params.OnStream
	if baseModelConfig.StopDisabled {
		if len(stop) > 0 {
			onStream = func(chunk string, buffer string) (shouldStop bool) {
				for _, stopSequence := range stop {
					if strings.Contains(buffer, stopSequence) {
						return true
					}
				}
				if params.OnStream != nil {
					return params.OnStream(chunk, buffer)
				}
				return false
			}
		}
	} else {
		req.Stop = stop
	}

	if prediction != "" {
		req.Prediction = &types.OpenAIPrediction{
			Type:    "content",
			Content: prediction,
		}
	}

	res, err := CreateChatCompletionWithInternalStream(clients, authVars, modelConfig, settings, orgUserConfig, currentOrgId, currentUserId, ctx, req, onStream, reqStarted)

	if err != nil {
		return nil, err
	}

	if baseModelConfig.StopDisabled && len(stop) > 0 {
		earliest := len(res.Content)
		found := false
		for _, s := range stop {
			if i := strings.Index(res.Content, s); i != -1 && i < earliest {
				earliest = i
				found = true
			}
		}
		if found {
			res.Content = res.Content[:earliest]
		}
	}

	if params.AfterReq != nil {
		params.AfterReq()
	}

	// log.Printf("\n\n**\n\nModel response: %s\n\n**\n\n", res.Content)

	var inputTokens int
	var outputTokens int
	var cachedTokens int

	if res.Usage != nil {
		if res.Usage.PromptTokensDetails != nil {
			cachedTokens = res.Usage.PromptTokensDetails.CachedTokens
		}
		inputTokens = res.Usage.PromptTokens
		outputTokens = res.Usage.CompletionTokens
	} else {
		inputTokens = inputTokensEstimate
		outputTokens = shared.GetNumTokensEstimate(res.Content)

		if params.WillCacheNumTokens > 0 {
			cachedTokens = params.WillCacheNumTokens
		}
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("panic in DidSendModelRequest hook: %v\n%s", r, debug.Stack())
				go notify.NotifyErr(notify.SeverityError, fmt.Errorf("panic in DidSendModelRequest hook: %v\n%s", r, debug.Stack()))
			}
		}()

		_, apiErr := hooks.ExecHook(hooks.DidSendModelRequest, hooks.HookParams{
			Auth: auth,
			Plan: plan,
			DidSendModelRequestParams: &hooks.DidSendModelRequestParams{
				InputTokens:    inputTokens,
				OutputTokens:   outputTokens,
				CachedTokens:   cachedTokens,
				ModelId:        baseModelConfig.ModelId,
				ModelTag:       baseModelConfig.ModelTag,
				ModelName:      baseModelConfig.ModelName,
				ModelProvider:  baseModelConfig.Provider,
				ModelPackName:  modelPackName,
				ModelRole:      modelConfig.Role,
				Purpose:        purpose,
				GenerationId:   res.GenerationId,
				PlanId:         plan.Id,
				ModelStreamId:  modelStreamId,
				ConvoMessageId: convoMessageId,
				BuildId:        buildId,

				RequestStartedAt: reqStarted,
				Streaming:        true,
				Req:              &req,
				StreamResult:     res.Content,
				ModelConfig:      modelConfig,
				FirstTokenAt:     res.FirstTokenAt,
				SessionId:        sessionId,
			},
		})

		if apiErr != nil {
			log.Printf("buildWholeFile - error executing DidSendModelRequest hook: %v", apiErr)
			go notify.NotifyErr(notify.SeverityError, fmt.Errorf("error executing DidSendModelRequest hook: %v", apiErr))
		}
	}()

	return res, nil
}

func FilterEmptyMessages(messages []types.ExtendedChatMessage) []types.ExtendedChatMessage {
	filteredMessages := []types.ExtendedChatMessage{}
	for _, message := range messages {
		var content []types.ExtendedChatMessagePart
		for _, part := range message.Content {
			if part.Type != openai.ChatMessagePartTypeText || part.Text != "" {
				content = append(content, part)
			}
		}
		if len(content) > 0 {
			filteredMessages = append(filteredMessages, types.ExtendedChatMessage{
				Role:    message.Role,
				Content: content,
			})
		}
	}
	return filteredMessages
}

func CheckSingleSystemMessage(modelConfig *shared.ModelRoleConfig, baseModelConfig *shared.BaseModelConfig, messages []types.ExtendedChatMessage) []types.ExtendedChatMessage {
	if len(messages) == 1 && baseModelConfig.SingleMessageNoSystemPrompt {
		if messages[0].Role == openai.ChatMessageRoleSystem {
			msg := messages[0]
			msg.Role = openai.ChatMessageRoleUser
			return []types.ExtendedChatMessage{msg}
		}
	}

	return messages
}
