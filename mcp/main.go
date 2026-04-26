package main

import (
	"log"
	"os"

	"github.com/go-park-mail-ru/2026_1_ARIS/mcp/internal/client"
	"github.com/go-park-mail-ru/2026_1_ARIS/mcp/internal/tools"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	baseURL := os.Getenv("ARIS_API_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	apiClient := client.New(baseURL)

	s := server.NewMCPServer(
		"ARIS MCP Server",
		"1.0.0",
		server.WithToolCapabilities(false),
	)

	tools.RegisterFeedTools(s, apiClient)
	tools.RegisterPostTools(s, apiClient)
	tools.RegisterUserTools(s, apiClient)

	if err := server.ServeStdio(s); err != nil {
		log.Fatal(err)
	}
}
