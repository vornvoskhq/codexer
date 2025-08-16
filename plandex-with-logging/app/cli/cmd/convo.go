package cmd

import (
	"fmt"
	"plandex-cli/api"
	"plandex-cli/auth"
	"plandex-cli/lib"
	"plandex-cli/term"
	"regexp"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var plainTextOutput bool
var convoRaw bool

// convoCmd represents the convo command
var convoCmd = &cobra.Command{
	Use:   "convo [msg-range]",
	Short: "Display complete conversation history",
	Long:  `Display complete conversation history. Optionally specify a message number or range of messages (e.g. '1' or '5' or '1-5' or '5-')`,
	Run:   convo,
}

func init() {
	RootCmd.AddCommand(convoCmd)

	convoCmd.Flags().BoolVarP(&plainTextOutput, "plain", "p", false, "Output conversation in plain text with no ANSI codes")

	// for debugging output
	convoCmd.Flags().BoolVar(&convoRaw, "raw", false, "Output conversation in raw format")
	convoCmd.Flags().MarkHidden("raw")
}

const stoppedEarlyMsg = "You stopped the reply early"

func convo(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()
	lib.MustResolveProject()

	term.StartSpinner("")
	conversation, apiErr := api.Client.ListConvo(lib.CurrentPlanId, lib.CurrentBranch)
	term.StopSpinner()

	if apiErr != nil {
		term.OutputErrorAndExit("Error loading conversation: %v", apiErr.Msg)
	}

	if len(conversation) == 0 {
		fmt.Println("🤷‍♂️ No conversation history")
		return
	}

	var msgRange string
	var msgRangeStart, msgRangeEnd int
	if len(args) > 0 {
		msgRange = args[0]
	}
	if msgRange != "" {
		// validate either a number or a range of numbers
		if strings.Contains(msgRange, "-") {
			_, err := fmt.Sscanf(msgRange, "%d-%d", &msgRangeStart, &msgRangeEnd)
			if err != nil {
				_, err := fmt.Sscanf(msgRange, "%d-", &msgRangeStart)

				if err != nil {
					term.OutputErrorAndExit("Invalid message range: %s", msgRange)
				}

				msgRangeEnd = len(conversation)
			}
		} else {
			_, err := fmt.Sscanf(msgRange, "%d", &msgRangeStart)
			if err != nil {
				term.OutputErrorAndExit("Invalid message number: %s", msgRange)
			}
			msgRangeEnd = msgRangeStart
		}
	}

	var convo string
	var totalTokens int
	var didCut bool
	for i, msg := range conversation {
		if msgRangeStart > 0 && msg.Num < msgRangeStart {
			didCut = true
			continue
		}
		if msgRangeEnd > 0 && msg.Num > msgRangeEnd {
			didCut = true
			break
		}

		var author string
		if msg.Role == "assistant" {
			author = "🤖 Plandex"
		} else if msg.Role == "user" {
			author = "💬 You"
		} else {
			author = msg.Role
		}

		replyTags := msg.Flags.GetReplyTags()

		// format as above but start with day of week
		formattedTs := msg.CreatedAt.Local().Format("Mon Jan 2, 2006 | 3:04pm MST")

		// if it's today then use 'Today' instead of the date
		if msg.CreatedAt.Day() == time.Now().Day() {
			formattedTs = msg.CreatedAt.Local().Format("Today | 3:04pm MST")
		}

		// if it's yesterday then use 'Yesterday' instead of the date
		if msg.CreatedAt.Day() == time.Now().AddDate(0, 0, -1).Day() {
			formattedTs = msg.CreatedAt.Local().Format("Yesterday | 3:04pm MST")
		}

		var header string
		if len(replyTags) > 0 {
			header = fmt.Sprintf("#### %d | %s | %s | %s | %d 🪙 ", i+1,
				author, strings.Join(replyTags, " | "), formattedTs, msg.Tokens)
		} else {
			header = fmt.Sprintf("#### %d | %s | %s | %d 🪙 ", i+1,
				author, formattedTs, msg.Tokens)
		}

		txt := msg.Message
		if !convoRaw {
			txt = convertCodeBlocks(msg.Message)
		}

		if plainTextOutput {
			convo += header + "\n" + txt + "\n\n"
		} else {
			md, err := term.GetMarkdown(header + "\n" + txt + "\n\n")
			if err != nil {
				term.OutputErrorAndExit("Error creating markdown representation: %v", err)
			}
			convo += md
		}

		if !didCut && msg.Stopped {
			if plainTextOutput {
				convo += fmt.Sprintf(" 🛑 %s\n\n", stoppedEarlyMsg)
			} else {
				convo += fmt.Sprintf(" 🛑 %s\n\n", color.New(color.Bold).Sprint(stoppedEarlyMsg))
			}
		}

		totalTokens += msg.Tokens
	}

	if !plainTextOutput {
		convo = strings.ReplaceAll(convo, stoppedEarlyMsg, color.New(term.ColorHiRed).Sprint(stoppedEarlyMsg))
	}

	output :=
		fmt.Sprintf("\n%s", convo)

	if !plainTextOutput && !didCut {
		output += term.GetDivisionLine() +
			color.New(color.Bold, term.ColorHiCyan).Sprint("  Conversation size →") + fmt.Sprintf(" %d 🪙", totalTokens) + "\n\n"
	}

	if plainTextOutput {
		fmt.Println(output)
	} else {
		term.PageOutput(output)

		fmt.Println()
		term.PrintCmds("", "convo 1", "convo 2-5", "convo --plain", "log")
	}
}

var codeBlockPattern = regexp.MustCompile(`<PlandexBlock\s+lang="(.+?)".*?>([\s\S]+?)</PlandexBlock>`)

func convertCodeBlocks(msg string) string {
	return codeBlockPattern.ReplaceAllStringFunc(msg, func(match string) string {
		// Extract language and content from the match
		submatches := codeBlockPattern.FindStringSubmatch(match)
		lang := submatches[1]
		content := submatches[2]

		// Escape any backticks in the content
		content = strings.ReplaceAll(content, "```", "\\`\\`\\`")

		// Return markdown code block format
		return fmt.Sprintf("```%s%s```", lang, content)
	})
}
