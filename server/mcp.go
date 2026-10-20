package server

import (
	"context"
	"net/http"
	"strings"

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
	token, err := s.getToken(ctx, tokenID, core.TokenTypeAccess)
	if err != nil {
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
	}, &mcp.ServerOptions{
		Instructions: "Tools that write data only save the changes: they are not visible on the website until buildWebsite is called. " +
			"Call buildWebsite after a write or a batch of writes. When planning more than one write, make them all first and call buildWebsite once at the end.",
	})

	if s.meilisearch != nil {
		readAnnotations := &mcp.ToolAnnotations{
			ReadOnlyHint:  true,
			OpenWorldHint: new(false),
		}

		mcp.AddTool(server, &mcp.Tool{
			Name:        "searchEntries",
			Description: "Search entries with pagination and query",
			Annotations: readAnnotations,
		}, s.mcpSearchEntries)

		mcp.AddTool(server, &mcp.Tool{
			Name:        "getTags",
			Description: "Get all tags of the searchable entries, with the number of entries per tag",
			Annotations: readAnnotations,
		}, s.mcpGetTags)
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "buildWebsite",
		Description: "Build the website so that changes are published. Changes made by write tools are not visible until this is called. Call it once after a write or a batch of writes, not after each write",
		Annotations: &mcp.ToolAnnotations{
			DestructiveHint: new(false),
			IdempotentHint:  true,
			OpenWorldHint:   new(false),
		},
	}, s.mcpBuildWebsite)

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

type BuildWebsiteParams struct {
	Clean bool `json:"clean,omitempty" jsonschema:"Build into a new directory, dropping files from removed pages"`
}

func (s *Server) mcpBuildWebsite(ctx context.Context, req *mcp.CallToolRequest, params *BuildWebsiteParams) (*mcp.CallToolResult, any, error) {
	return nil, nil, s.core.Build(params.Clean)
}

type mcpTags struct {
	Tags map[string]int64 `json:"tags" jsonschema:"Number of entries per tag"`
}

func (s *Server) mcpGetTags(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, *mcpTags, error) {
	tags, err := s.meilisearch.Tags()
	if err != nil {
		return nil, nil, err
	}

	return nil, &mcpTags{Tags: tags}, nil
}
