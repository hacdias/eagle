package server

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/modelcontextprotocol/go-sdk/oauthex"
	"github.com/samber/lo"
	"go.hacdias.com/eagle/core"
)

const (
	mcpPath  = "/mcp"
	mcpScope = "mcp"

	wellKnownOAuthProtectedResource = "/.well-known/oauth-protected-resource" + mcpPath
)

// mcpAuth requires a valid access token with the MCP scope.
func (s *Server) mcpAuth(next http.Handler) http.Handler {
	return auth.RequireBearerToken(s.mcpVerifyToken, &auth.RequireBearerTokenOptions{
		ResourceMetadataURL: s.c.AbsoluteURL(wellKnownOAuthProtectedResource),
		Scopes:              []string{mcpScope},
	})(next)
}

func (s *Server) mcpVerifyToken(ctx context.Context, tokenID string, _ *http.Request) (*auth.TokenInfo, error) {
	token, err := s.core.DB().GetToken(ctx, tokenID, core.TokenTypeAccess)
	if err != nil {
		return nil, auth.ErrInvalidToken
	}

	if time.Now().After(token.Expiry) {
		_ = s.core.DB().DeleteToken(ctx, tokenID)
		return nil, auth.ErrInvalidToken
	}

	return &auth.TokenInfo{
		Scopes:     strings.Fields(token.Scope),
		Expiration: token.Expiry,
		UserID:     s.c.ID(),
	}, nil
}

func (s *Server) mcpProtectedResourceGet() http.Handler {
	return auth.ProtectedResourceMetadataHandler(&oauthex.ProtectedResourceMetadata{
		Resource:             s.c.AbsoluteURL(mcpPath),
		AuthorizationServers: []string{s.c.ID()},
		ScopesSupported:      []string{mcpScope},
	})
}

func (s *Server) initMcp() error {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "Eagle",
		Version: "0.0.2",
	}, nil)

	if s.meilisearch != nil {
		mcp.AddTool(server, &mcp.Tool{
			Name:        "searchEntries",
			Description: "Search entries with pagination and query",
			Annotations: &mcp.ToolAnnotations{
				ReadOnlyHint:  true,
				OpenWorldHint: new(false),
			},
		}, s.mcpSearchEntries)
	}

	for _, plugin := range s.plugins {
		mcpPlugin, ok := plugin.(MCPPlugin)
		if !ok {
			continue
		}

		mcpPlugin.RegisterMCPTools(server)
	}

	s.mcp = server
	return nil
}

type SearchEntriesParams struct {
	Page  int64  `json:"page" jsonschema:"Page number for search results, 0 indexed"`
	Limit int64  `json:"limit" jsonschema:"Number of results per page"`
	Query string `json:"query" jsonschema:"Search query"`
}

func (s *Server) mcpSearchEntries(ctx context.Context, req *mcp.CallToolRequest, params *SearchEntriesParams) (*mcp.CallToolResult, any, error) {
	results, err := s.meilisearch.Search(params.Page, params.Limit, params.Query)
	if err != nil {
		return nil, nil, err
	}

	content := lo.Map(results, func(e *core.Entry, _ int) mcp.Content {
		return &mcp.TextContent{Text: e.Content}
	})

	return &mcp.CallToolResult{
		Content: content,
	}, nil, nil
}
