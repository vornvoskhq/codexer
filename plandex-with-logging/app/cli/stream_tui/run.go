package streamtui

import (
	"fmt"
	"log"
	"os"
	"plandex-cli/term"
	"sync"

	shared "plandex-shared"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fatih/color"
)

var ui *tea.Program
var mu sync.Mutex
var wg sync.WaitGroup

var prestartReply string
var prestartErr *shared.ApiError
var prestartAbort bool

func StartStreamUI(prompt string, buildOnly, canSendToBg bool) error {
	if prestartErr != nil {
		log.Println("stream UI - prestart error: ", prestartErr)
		term.HandleApiError(prestartErr)
	}

	if prestartAbort {
		fmt.Println("🛑 Stopped early")
		os.Exit(0)
	}

	log.Println("Starting stream UI")

	initial := initialModel(prestartReply, prompt, buildOnly, canSendToBg)

	mu.Lock()
	ui = tea.NewProgram(initial, tea.WithAltScreen())
	mu.Unlock()

	log.Println("Running bubbletea program")
	wg.Add(1)
	m, err := ui.Run()
	log.Println("Bubbletea program finished")
	wg.Done()

	log.Println("Stream UI finished")

	if err != nil {
		return fmt.Errorf("error running stream UI: %v", err)
	}

	var mod *streamUIModel
	c, ok := m.(*streamUIModel)
	if ok {
		mod = c
	} else {
		c := m.(streamUIModel)
		mod = &c
	}

	fmt.Println()

	if !mod.buildOnly {
		fmt.Println(mod.mainDisplay)
	}

	if len(mod.finishedByPath) > 0 || len(mod.tokensByPath) > 0 {
		fmt.Println(mod.renderStaticBuild())
	}

	if mod.err != nil {
		log.Println("stream UI - error: ", mod.err)

		fmt.Println()
		term.OutputErrorAndExit(mod.err.Error())
	}

	if mod.apiErr != nil {
		log.Println("stream UI - api error: ", mod.apiErr)

		fmt.Println()
		term.HandleApiError(mod.apiErr)
	}

	if mod.stopped {
		fmt.Println()
		color.New(color.BgBlack, color.Bold, color.FgHiRed).Println(" 🛑 Stopped early ")
		fmt.Println()
		term.PrintCmds("", "log", "rewind", "tell")
		os.Exit(0)
	} else if mod.background {
		fmt.Println()
		color.New(color.BgBlack, color.Bold, color.FgHiGreen).Println(" ✅ Plan is active in the background ")
		fmt.Println()
		term.PrintCmds("", "ps", "connect", "stop")
		os.Exit(0)
	}

	if os.Getenv("PLANDEX_REPL") != "" && os.Getenv("PLANDEX_REPL_OUTPUT_FILE") != "" {
		// write output to file
		err := os.WriteFile(os.Getenv("PLANDEX_REPL_OUTPUT_FILE"), []byte(mod.reply), 0644)
		if err != nil {
			log.Println("stream UI - error writing output to repl temp file: ", err)
		}
	}

	return nil
}

func Quit() {
	if ui == nil {
		log.Println("stream UI is nil, can't quit")
		return
	}
	mu.Lock()
	if ui != nil {
		ui.Quit()
	}
	mu.Unlock()

	wg.Wait() // Wait for the UI to fully terminate
}

func Send(msg shared.StreamMessage) {
	if ui == nil {
		log.Println("stream ui is nil")

		if msg.Type == shared.StreamMessageError {
			prestartErr = msg.Error
		} else if msg.Type == shared.StreamMessageAborted {

		} else if msg.Type == shared.StreamMessageReply {
			prestartReply += msg.ReplyChunk
		}
		return
	}
	mu.Lock()
	defer mu.Unlock()
	ui.Send(msg)
}

func ToggleVisibility(hide bool) {
	if ui == nil {
		return
	}
	mu.Lock()
	defer mu.Unlock()

	if hide {
		ui.Send(tea.ExitAltScreen())
	} else {
		ui.Send(tea.EnterAltScreen())
	}
}
