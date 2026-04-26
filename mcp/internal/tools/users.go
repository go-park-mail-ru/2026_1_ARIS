package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-park-mail-ru/2026_1_ARIS/mcp/internal/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func RegisterUserTools(s *server.MCPServer, c *client.Client) {
	s.AddTool(
		mcp.NewTool("get_my_profile",
			mcp.WithDescription("Get the authenticated user's own profile (name, avatar, bio, stats)."),
			mcp.WithString("session_cookie",
				mcp.Description("Session cookie value for authentication"),
				mcp.Required(),
			),
		),
		getMyProfileHandler(c),
	)

	s.AddTool(
		mcp.NewTool("get_profile",
			mcp.WithDescription("Get a user profile by profile ID."),
			mcp.WithString("session_cookie",
				mcp.Description("Session cookie value for authentication"),
				mcp.Required(),
			),
			mcp.WithNumber("profile_id",
				mcp.Description("Numeric profile ID of the target user"),
				mcp.Required(),
			),
		),
		getProfileHandler(c),
	)

	s.AddTool(
		mcp.NewTool("get_suggested_users",
			mcp.WithDescription("Get a list of suggested users to follow for the authenticated user."),
			mcp.WithString("session_cookie",
				mcp.Description("Session cookie value for authentication"),
				mcp.Required(),
			),
		),
		getSuggestedUsersHandler(c),
	)

	s.AddTool(
		mcp.NewTool("get_popular_users",
			mcp.WithDescription("Get a list of popular users on the platform (no auth required)."),
		),
		getPopularUsersHandler(c),
	)
}

func getMyProfileHandler(c *client.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		session := req.GetString("session_cookie", "")

		var result any
		if err := c.Get("/api/profile/me", session, &result); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		data, _ := json.MarshalIndent(result, "", "  ")
		return mcp.NewToolResultText(string(data)), nil
	}
}

func getProfileHandler(c *client.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		session := req.GetString("session_cookie", "")
		profileID := req.GetInt("profile_id", 0)
		if profileID == 0 {
			return mcp.NewToolResultError("profile_id is required"), nil
		}

		var result any
		if err := c.Get(fmt.Sprintf("/api/profile/%d", profileID), session, &result); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		data, _ := json.MarshalIndent(result, "", "  ")
		return mcp.NewToolResultText(string(data)), nil
	}
}

func getSuggestedUsersHandler(c *client.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		session := req.GetString("session_cookie", "")

		var result any
		if err := c.Get("/api/users/suggested", session, &result); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		data, _ := json.MarshalIndent(result, "", "  ")
		return mcp.NewToolResultText(string(data)), nil
	}
}

func getPopularUsersHandler(c *client.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var result any
		if err := c.Get("/api/public/popular-users", "", &result); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		data, _ := json.MarshalIndent(result, "", "  ")
		return mcp.NewToolResultText(string(data)), nil
	}
}
