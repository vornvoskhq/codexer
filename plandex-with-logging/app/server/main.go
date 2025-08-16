package main

import (
	"fmt"
	"log"
	"os"
	"plandex-server/model"
	"plandex-server/routes"
	"plandex-server/setup"

	"github.com/gorilla/mux"

	// LLM logging package
	"plandex-server/pkg/llmlog"
	shared "plandex-shared"
)

var llmLogger *llmlog.Logger // LLM logging instance (global for now)

func main() {
	// Configure the default logger to include milliseconds in timestamps
	log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.Lshortfile)

	// --- LLM Logging Integration ---
	
	// Initialize logger with default config (can be overridden by environment variables)
	llmLogCfg := &llmlog.Config{
		Enabled:  true,  // Enable by default
		FilePath: "./llm-logs/llm-requests.log",
	}

	// Initialize the logger
	var err error
	llmLogger, err = llmlog.NewLogger(llmLogCfg)
	if err != nil {
		log.Printf("Warning: Failed to initialize LLM logger: %v. Continuing without LLM logging.", err)
	} else {
		log.Println("LLM request logging enabled")
	}

	// Register shutdown hook to close the logger
	setup.RegisterShutdownHook(func() {
		if llmLogger != nil {
			if err := llmLogger.Close(); err != nil {
				log.Printf("Error closing LLM logger: %v", err)
			}
		}
		model.ShutdownLiteLLMServer()
	})

	// Register the route handler
	routes.RegisterHandlePlandex(func(router *mux.Router, path string, isStreaming bool, handler routes.PlandexHandler) *mux.Route {
		return router.HandleFunc(path, handler)
	})

	// Initialize and start the LiteLLM proxy
	err = model.EnsureLiteLLM(2)
	if err != nil {
		panic(fmt.Sprintf("Failed to start LiteLLM proxy: %v", err))
	}

	// Initialize clients with empty configs (will be configured per-request)
	authVars := make(map[string]string)
	settings := &shared.PlanSettings{}
	orgUserConfig := &shared.OrgUserConfig{}

	// Initialize the clients
	clients := model.InitClients(authVars, settings, orgUserConfig)

	// Wrap the clients with logging if the logger is available
	if llmLogger != nil {
		for modelName, clientInfo := range clients {
			wrappedClient := llmlog.WrapClient(clientInfo, llmLogger)
			clients[modelName] = wrappedClient
		}
	}

	// Set up the HTTP server
	r := mux.NewRouter()
	routes.AddHealthRoutes(r)
	routes.AddApiRoutes(r)
	routes.AddProxyableApiRoutes(r)

	// Initialize the database and start the server
	setup.MustLoadIp()
	setup.MustInitDb()
	setup.StartServer(r, nil, nil)

	os.Exit(0)
}
