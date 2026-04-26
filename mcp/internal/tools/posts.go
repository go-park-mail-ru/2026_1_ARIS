package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-park-mail-ru/2026_1_ARIS/mcp/internal/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func RegisterPostTools(s *server.MCPServer, c *client.Client) {
	s.AddTool(
		mcp.NewTool("get_post",
			mcp.WithDescription("Get a single post by ID. Returns post content, author info, and attached media."),
			mcp.WithString("session_cookie",
				mcp.Description("Session cookie value for authentication"),
				mcp.Required(),
			),
			mcp.WithNumber("post_id",
				mcp.Description("Numeric post ID"),
				mcp.Required(),
			),
		),
		getPostHandler(c),
	)

	s.AddTool(
		mcp.NewTool("get_my_posts",
			mcp.WithDescription("Get all posts created by the authenticated user."),
			mcp.WithString("session_cookie",
				mcp.Description("Session cookie value for authentication"),
				mcp.Required(),
			),
		),
		getMyPostsHandler(c),
	)

	s.AddTool(
		mcp.NewTool("get_profile_posts",
			mcp.WithDescription("Get all posts by a specific user profile ID."),
			mcp.WithString("session_cookie",
				mcp.Description("Session cookie value for authentication"),
				mcp.Required(),
			),
			mcp.WithNumber("profile_id",
				mcp.Description("Numeric profile ID of the target user"),
				mcp.Required(),
			),
		),
		getProfilePostsHandler(c),
	)
}

func getPostHandler(c *client.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		session := req.GetString("session_cookie", "")
		postID := req.GetInt("post_id", 0)
		if postID == 0 {
			return mcp.NewToolResultError("post_id is required"), nil
		}

		var result any
		if err := c.Get(fmt.Sprintf("/api/post/%d", postID), session, &result); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		data, _ := json.MarshalIndent(result, "", "  ")
		return mcp.NewToolResultText(string(data)), nil
	}
}

func getMyPostsHandler(c *client.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		session := req.GetString("session_cookie", "")

		var result any
		if err := c.Get("/api/post/me", session, &result); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		data, _ := json.MarshalIndent(result, "", "  ")
		return mcp.NewToolResultText(string(data)), nil
	}
}

func getProfilePostsHandler(c *client.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		session := req.GetString("session_cookie", "")
		profileID := req.GetInt("profile_id", 0)
		if profileID == 0 {
			return mcp.NewToolResultError("profile_id is required"), nil
		}

		var result any
		if err := c.Get(fmt.Sprintf("/api/post/profile/%d", profileID), session, &result); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		data, _ := json.MarshalIndent(result, "", "  ")
		return mcp.NewToolResultText(string(data)), nil
	}
}
