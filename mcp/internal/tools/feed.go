package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-park-mail-ru/2026_1_ARIS/mcp/internal/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func RegisterFeedTools(s *server.MCPServer, c *client.Client) {
	s.AddTool(
		mcp.NewTool("get_public_feed",
			mcp.WithDescription("Get public feed posts (no auth required). Returns paginated list of posts with author info, likes, comments count."),
			mcp.WithNumber("limit",
				mcp.Description("Number of posts to return (default 20)"),
			),
			mcp.WithString("cursor",
				mcp.Description("Pagination cursor from previous response"),
			),
		),
		getPublicFeedHandler(c),
	)

	s.AddTool(
		mcp.NewTool("get_feed",
			mcp.WithDescription("Get personalized feed for authenticated user. Returns paginated list of posts."),
			mcp.WithString("session_cookie",
				mcp.Description("Session cookie value for authentication"),
				mcp.Required(),
			),
			mcp.WithNumber("limit",
				mcp.Description("Number of posts to return (default 20)"),
			),
			mcp.WithString("cursor",
				mcp.Description("Pagination cursor from previous response"),
			),
		),
		getFeedHandler(c),
	)

	s.AddTool(
		mcp.NewTool("get_popular_posts",
			mcp.WithDescription("Get list of popular post topics (no auth required)."),
		),
		getPopularPostsHandler(c),
	)
}

func getPublicFeedHandler(c *client.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		path := "/api/public/feed"
		params := ""
		if limit := req.GetInt("limit", 0); limit > 0 {
			params += fmt.Sprintf("?limit=%d", limit)
		}
		if cursor := req.GetString("cursor", ""); cursor != "" {
			if params == "" {
				params = "?cursor=" + cursor
			} else {
				params += "&cursor=" + cursor
			}
		}

		var result any
		if err := c.Get(path+params, "", &result); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		data, _ := json.MarshalIndent(result, "", "  ")
		return mcp.NewToolResultText(string(data)), nil
	}
}

func getFeedHandler(c *client.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		session := req.GetString("session_cookie", "")
		path := "/api/feed"
		params := ""
		if limit := req.GetInt("limit", 0); limit > 0 {
			params += fmt.Sprintf("?limit=%d", limit)
		}
		if cursor := req.GetString("cursor", ""); cursor != "" {
			if params == "" {
				params = "?cursor=" + cursor
			} else {
				params += "&cursor=" + cursor
			}
		}

		var result any
		if err := c.Get(path+params, session, &result); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		data, _ := json.MarshalIndent(result, "", "  ")
		return mcp.NewToolResultText(string(data)), nil
	}
}

func getPopularPostsHandler(c *client.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var result any
		if err := c.Get("/api/public/popular-posts", "", &result); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		data, _ := json.MarshalIndent(result, "", "  ")
		return mcp.NewToolResultText(string(data)), nil
	}
}
