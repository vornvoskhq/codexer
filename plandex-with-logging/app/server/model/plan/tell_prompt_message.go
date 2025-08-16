package plan

import (
	"log"
	"net/http"
	"plandex-server/model/prompts"
	"plandex-server/types"
	shared "plandex-shared"

	"github.com/sashabaranov/go-openai"
)

func (state *activeTellStreamState) resolvePromptMessage(
	unfinishedSubtaskReasoning string,
) (*types.ExtendedChatMessage, bool) {
	req := state.req
	active := state.activePlan
	iteration := state.iteration

	var promptMessage *types.ExtendedChatMessage

	state.skipConvoMessages = map[string]bool{}

	lastMessage := state.lastSuccessfulConvoMessage()

	if req.IsUserContinue {
		if lastMessage == nil {
			active.StreamDoneCh <- &shared.ApiError{
				Type:   shared.ApiErrorTypeContinueNoMessages,
				Status: http.StatusBadRequest,
				Msg:    "No messages yet. Can't continue plan.",
			}
			return nil, false
		}
		log.Println("User is continuing plan. Last message role:", lastMessage.Role)
	}

	if req.IsChatOnly {
		var prompt string
		if req.IsUserContinue {
			if lastMessage.Role == openai.ChatMessageRoleUser {
				log.Println("User is continuing plan in chat only mode. Last message was user message. Using last user message as prompt")
				content := lastMessage.Message
				prompt = content
				state.userPrompt = content
				state.skipConvoMessages[lastMessage.Id] = true
			} else {
				log.Println("User is continuing plan in chat only mode. Last message was assistant message. Using user continue prompt")
				prompt = prompts.UserContinuePrompt
			}
		} else {
			prompt = req.Prompt
		}

		wrapped := prompts.GetWrappedChatOnlyPrompt(prompts.ChatUserPromptParams{
			CreatePromptParams: prompts.CreatePromptParams{
				AutoContext: req.AutoContext,
				ExecMode:    req.ExecEnabled,
				IsGitRepo:   req.IsGitRepo,
				// no need to pass in IsUserDebug or IsApplyDebug here because it's a chat message
			},
			Prompt:    prompt,
			OsDetails: req.OsDetails,
			// no current task for chat only mode
		})

		promptMessage = &types.ExtendedChatMessage{
			Role: openai.ChatMessageRoleUser,
			Content: []types.ExtendedChatMessagePart{
				{
					Type: openai.ChatMessagePartTypeText,
					Text: wrapped,
				},
			},
		}
	} else if req.IsUserContinue {
		// log.Println("User is continuing plan. Last message:\n\n", lastMessage.Content)
		if lastMessage.Role == openai.ChatMessageRoleUser {
			// if last message was a user message, we want to remove it from the messages array and then use that last message as the prompt so we can continue from where the user left off

			log.Println("User is continuing plan in tell mode. Last message was user message. Using last user message as prompt")
			content := lastMessage.Message

			params := prompts.UserPromptParams{
				CreatePromptParams: prompts.CreatePromptParams{
					ExecMode:          req.ExecEnabled,
					AutoContext:       req.AutoContext,
					IsGitRepo:         req.IsGitRepo,
					ContextTokenLimit: state.settings.GetPlannerEffectiveMaxTokens(),
					// no need to pass in IsUserDebug or IsApplyDebug here because we're continuing
				},
				Prompt:                     content,
				OsDetails:                  req.OsDetails,
				CurrentStage:               state.currentStage,
				UnfinishedSubtaskReasoning: unfinishedSubtaskReasoning,
			}

			promptMessage = &types.ExtendedChatMessage{
				Role: openai.ChatMessageRoleUser,
				Content: []types.ExtendedChatMessagePart{
					{
						Type: openai.ChatMessagePartTypeText,
						Text: prompts.GetWrappedPrompt(params),
					},
				},
			}

			state.userPrompt = content
		} else {

			// if the last message was an assistant message, we'll use the user continue prompt
			log.Println("User is continuing plan in tell mode. Last message was assistant message. Using user continue prompt")

			params := prompts.UserPromptParams{
				CreatePromptParams: prompts.CreatePromptParams{
					ExecMode:          req.ExecEnabled,
					AutoContext:       req.AutoContext,
					IsGitRepo:         req.IsGitRepo,
					ContextTokenLimit: state.settings.GetPlannerEffectiveMaxTokens(),
					// no need to pass in IsUserDebug or IsApplyDebug here because we're continuing
				},
				Prompt:                     prompts.UserContinuePrompt,
				OsDetails:                  req.OsDetails,
				CurrentStage:               state.currentStage,
				UnfinishedSubtaskReasoning: unfinishedSubtaskReasoning,
			}

			promptMessage = &types.ExtendedChatMessage{
				Role: openai.ChatMessageRoleUser,
				Content: []types.ExtendedChatMessagePart{
					{
						Type: openai.ChatMessagePartTypeText,
						Text: prompts.GetWrappedPrompt(params),
					},
				},
			}
		}
	} else {
		var prompt string
		if iteration == 0 {
			prompt = req.Prompt
		} else if state.currentStage.TellStage == shared.TellStageImplementation {
			prompt = prompts.AutoContinueImplementationPrompt
		} else {
			prompt = prompts.AutoContinuePlanningPrompt
		}

		state.userPrompt = prompt

		params := prompts.UserPromptParams{
			CreatePromptParams: prompts.CreatePromptParams{
				ExecMode:          req.ExecEnabled,
				AutoContext:       req.AutoContext,
				IsUserDebug:       req.IsUserDebug,
				IsApplyDebug:      req.IsApplyDebug,
				IsGitRepo:         req.IsGitRepo,
				ContextTokenLimit: state.settings.GetPlannerEffectiveMaxTokens(),
			},
			Prompt:                     prompt,
			OsDetails:                  req.OsDetails,
			CurrentStage:               state.currentStage,
			UnfinishedSubtaskReasoning: unfinishedSubtaskReasoning,
		}

		finalPrompt := prompts.GetWrappedPrompt(params)

		// log.Println("Final prompt:", finalPrompt)

		promptMessage = &types.ExtendedChatMessage{
			Role: openai.ChatMessageRoleUser,
			Content: []types.ExtendedChatMessagePart{
				{
					Type: openai.ChatMessagePartTypeText,
					Text: finalPrompt,
				},
			},
		}
	}

	// log.Println("Prompt message:", promptMessage.Content)

	return promptMessage, true
}
