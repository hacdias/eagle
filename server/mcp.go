package server

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
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

var (
	mcpReadAnnotations = &mcp.ToolAnnotations{
		ReadOnlyHint:  true,
		OpenWorldHint: new(false),
	}

	// Logging appends a new entry, so repeated calls create duplicates.
	mcpLogAnnotations = &mcp.ToolAnnotations{
		DestructiveHint: new(false),
		OpenWorldHint:   new(false),
	}

	// Editing replaces all fields of an existing entry.
	mcpEditAnnotations = &mcp.ToolAnnotations{
		DestructiveHint: new(true),
		IdempotentHint:  true,
		OpenWorldHint:   new(false),
	}
)

func (s *Server) initMcp() error {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "Eagle",
		Version: "0.0.2",
	}, nil)

	if s.meilisearch != nil {
		mcp.AddTool(server, &mcp.Tool{
			Name:        "searchEntries",
			Description: "Search entries with pagination and query",
			Annotations: mcpReadAnnotations,
		}, s.mcpSearchEntries)
	}

	// Get logs
	// TODO: consider adding "after" / "before" query parameters for filtering logs by date.
	mcp.AddTool(server, &mcp.Tool{
		Name:        "getReadingLogs",
		Description: "Get all reading logs",
		Annotations: mcpReadAnnotations,
	}, s.mcpGetReadingLogs)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "getMovieLogs",
		Description: "Get all movie logs",
		Annotations: mcpReadAnnotations,
	}, s.mcpGetMovieLogs)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "getShowLogs",
		Description: "Get all show logs",
		Annotations: mcpReadAnnotations,
	}, s.mcpGetShowLogs)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "getPerformanceLogs",
		Description: "Get all performance logs",
		Annotations: mcpReadAnnotations,
	}, s.mcpGetPerformanceLogs)

	// Append to logs
	mcp.AddTool(server, &mcp.Tool{
		Name:        "logMovie",
		Description: "Log a movie",
		Annotations: mcpLogAnnotations,
		InputSchema: mcpLogInputSchema[LogMovieParams](),
	}, s.mcpLogMovie)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "logPerformance",
		Description: "Log a performance",
		Annotations: mcpLogAnnotations,
		InputSchema: mcpPerformanceInputSchema[LogPerformanceParams](),
	}, s.mcpLogPerformance)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "logReading",
		Description: "Log a reading",
		Annotations: mcpLogAnnotations,
		InputSchema: mcpLogInputSchema[LogReadingParams](),
	}, s.mcpLogReading)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "logShow",
		Description: "Log a show",
		Annotations: mcpLogAnnotations,
		InputSchema: mcpLogInputSchema[LogShowParams](),
	}, s.mcpLogShow)

	// Edit logs
	mcp.AddTool(server, &mcp.Tool{
		Name:        "editMovieLog",
		Description: "Edit a logged movie, identified by its original name and date. All fields are replaced",
		Annotations: mcpEditAnnotations,
		InputSchema: mcpLogInputSchema[EditMovieLogParams](),
	}, s.mcpEditMovieLog)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "editPerformanceLog",
		Description: "Edit a logged performance, identified by its original date and its original name, author, or both. All fields are replaced",
		Annotations: mcpEditAnnotations,
		InputSchema: mcpPerformanceInputSchema[EditPerformanceLogParams](),
	}, s.mcpEditPerformanceLog)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "editReadingLog",
		Description: "Edit a logged reading, identified by its original name and date. All fields are replaced",
		Annotations: mcpEditAnnotations,
		InputSchema: mcpLogInputSchema[EditReadingLogParams](),
	}, s.mcpEditReadingLog)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "editShowLog",
		Description: "Edit a logged show, identified by its original name, date and season. All fields are replaced",
		Annotations: mcpEditAnnotations,
		InputSchema: mcpLogInputSchema[EditShowLogParams](),
	}, s.mcpEditShowLog)

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

type mcpLogs[T any] struct {
	Logs []T `json:"logs"`
}

func (s *Server) mcpGetReadingLogs(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, *mcpLogs[core.ReadingLogEntry], error) {
	readings, err := s.core.GetReadingLogs()
	if err != nil {
		return nil, nil, err
	}

	return nil, &mcpLogs[core.ReadingLogEntry]{Logs: readings}, nil
}

func (s *Server) mcpGetMovieLogs(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, *mcpLogs[core.MovieLogEntry], error) {
	movies, err := s.core.GetMovieLogs()
	if err != nil {
		return nil, nil, err
	}

	return nil, &mcpLogs[core.MovieLogEntry]{Logs: movies}, nil
}

func (s *Server) mcpGetShowLogs(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, *mcpLogs[core.ShowLogEntry], error) {
	shows, err := s.core.GetShowLogs()
	if err != nil {
		return nil, nil, err
	}

	return nil, &mcpLogs[core.ShowLogEntry]{Logs: shows}, nil
}

func (s *Server) mcpGetPerformanceLogs(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, *mcpLogs[core.PerformanceLogEntry], error) {
	performances, err := s.core.GetPerformanceLogs()
	if err != nil {
		return nil, nil, err
	}

	return nil, &mcpLogs[core.PerformanceLogEntry]{Logs: performances}, nil
}

func mcpLogInputSchema[T any]() *jsonschema.Schema {
	schema, err := jsonschema.For[T](nil)
	if err != nil {
		panic(err)
	}

	// Required strings must not be empty.
	for _, name := range schema.Required {
		if prop := schema.Properties[name]; prop.Type == "string" {
			prop.MinLength = jsonschema.Ptr(1)
		}
	}

	for name, prop := range schema.Properties {
		switch name {
		case "date", "originalDate":
			prop.Format = "date-time"
		case "rating":
			prop.Minimum = jsonschema.Ptr(1.0)
			prop.Maximum = jsonschema.Ptr(10.0)
		case "pages", "season", "originalSeason":
			prop.Minimum = jsonschema.Ptr(1.0)
		case "category":
			prop.Enum = []any{"musical", "concert", "comedy show", "classical", "theater"}
		}
	}

	return schema
}

// mcpPerformanceInputSchema requires a performance to have a name, an author,
// or both. When editing, the same applies to the original name and author.
func mcpPerformanceInputSchema[T any]() *jsonschema.Schema {
	schema := mcpLogInputSchema[T]()
	schema.AllOf = append(schema.AllOf, mcpRequireAny(schema, "name", "author"))
	if _, ok := schema.Properties["originalName"]; ok {
		schema.AllOf = append(schema.AllOf, mcpRequireAny(schema, "originalName", "originalAuthor"))
	}
	return schema
}

// mcpRequireAny returns a schema that requires at least one of the given
// properties, which must not be empty.
func mcpRequireAny(schema *jsonschema.Schema, names ...string) *jsonschema.Schema {
	anyOf := &jsonschema.Schema{}
	for _, name := range names {
		schema.Properties[name].MinLength = jsonschema.Ptr(1)
		anyOf.AnyOf = append(anyOf.AnyOf, &jsonschema.Schema{Required: []string{name}})
	}
	return anyOf
}

type LogMovieParams struct {
	Name     string    `json:"name" jsonschema:"Title of the movie"`
	Date     time.Time `json:"date" jsonschema:"When the movie was watched. A full RFC 3339 date-time with time and offset, e.g. 2026-10-16T21:30:00+01:00; a bare date is rejected"`
	Rating   int       `json:"rating" jsonschema:"Rating of the movie, from 1 to 10"`
	Location string    `json:"location" jsonschema:"Where the movie was watched, e.g. the cinema name"`
	Room     string    `json:"room,omitempty" jsonschema:"Cinema room or screen. Omit if not watched at a cinema"`
	Row      string    `json:"row,omitempty" jsonschema:"Cinema seat row. Omit if not watched at a cinema"`
	Seat     string    `json:"seat,omitempty" jsonschema:"Cinema seat. Omit if not watched at a cinema"`
}

func (p *LogMovieParams) entry() core.MovieLogEntry {
	return core.MovieLogEntry{
		Name:     p.Name,
		Date:     p.Date,
		Rating:   p.Rating,
		Location: p.Location,
		Room:     p.Room,
		Row:      p.Row,
		Seat:     p.Seat,
	}
}

func (s *Server) mcpLogMovie(ctx context.Context, req *mcp.CallToolRequest, params *LogMovieParams) (*mcp.CallToolResult, any, error) {
	entry := params.entry()
	err := s.core.AddMovieLog(entry)
	if err != nil {
		return nil, nil, err
	}

	return nil, entry, nil
}

type LogPerformanceParams struct {
	Name          string    `json:"name,omitempty" jsonschema:"Title of the performance, e.g. the musical, play or tour. At least one of name and author is required"`
	Author        string    `json:"author,omitempty" jsonschema:"Main performer, artist, band or company. At least one of name and author is required"`
	Rating        int       `json:"rating" jsonschema:"Rating of the performance, from 1 to 10"`
	Date          time.Time `json:"date" jsonschema:"When the performance took place. A full RFC 3339 date-time with time and offset, e.g. 2026-10-16T21:30:00+01:00; a bare date is rejected"`
	Location      string    `json:"location" jsonschema:"Venue where the performance took place, formatted as 'Venue, City, CC' with CC being the ISO 3166-1 alpha-2 country code, e.g. 'Ziggo Dome, Amsterdam, NL'"`
	Category      string    `json:"category" jsonschema:"Kind of performance"`
	SupportingAct []string  `json:"supportingAct,omitempty" jsonschema:"Opening or supporting acts. Omit if there were none"`
}

func (p *LogPerformanceParams) entry() core.PerformanceLogEntry {
	return core.PerformanceLogEntry{
		Name:          p.Name,
		Author:        p.Author,
		Rating:        p.Rating,
		Date:          p.Date,
		Location:      p.Location,
		Category:      p.Category,
		SupportingAct: p.SupportingAct,
	}
}

func (s *Server) mcpLogPerformance(ctx context.Context, req *mcp.CallToolRequest, params *LogPerformanceParams) (*mcp.CallToolResult, any, error) {
	entry := params.entry()
	err := s.core.AddPerformanceLog(entry)
	if err != nil {
		return nil, nil, err
	}

	return nil, entry, nil
}

type LogReadingParams struct {
	Name      string    `json:"name" jsonschema:"Title of the book"`
	Author    string    `json:"author" jsonschema:"Author of the book"`
	Rating    int       `json:"rating" jsonschema:"Rating of the book, from 1 to 10"`
	Publisher string    `json:"publisher" jsonschema:"Publisher of the edition read"`
	Pages     int       `json:"pages" jsonschema:"Number of pages of the edition read"`
	Date      time.Time `json:"date" jsonschema:"When the book was finished. A full RFC 3339 date-time with time and offset, e.g. 2026-10-16T21:30:00+01:00; a bare date is rejected"`
	UID       string    `json:"uid" jsonschema:"Unique identifier of the edition read, in the format isbn:<ISBN-13>, e.g. isbn:9789895557738"`
}

func (p *LogReadingParams) entry() core.ReadingLogEntry {
	return core.ReadingLogEntry{
		Name:      p.Name,
		Author:    p.Author,
		Rating:    p.Rating,
		Publisher: p.Publisher,
		Pages:     p.Pages,
		Date:      p.Date,
		UID:       p.UID,
	}
}

func (s *Server) mcpLogReading(ctx context.Context, req *mcp.CallToolRequest, params *LogReadingParams) (*mcp.CallToolResult, any, error) {
	entry := params.entry()
	err := s.core.AddReadingLog(entry)
	if err != nil {
		return nil, nil, err
	}

	return nil, entry, nil
}

type LogShowParams struct {
	Name   string    `json:"name" jsonschema:"Title of the show"`
	Season int       `json:"season,omitempty" jsonschema:"Season number. Omit for shows without seasons, such as miniseries"`
	Date   time.Time `json:"date" jsonschema:"When the show or season was watched. A full RFC 3339 date-time with time and offset, e.g. 2026-10-16T21:30:00+01:00; a bare date is rejected"`
	Rating int       `json:"rating" jsonschema:"Rating of the show or season, from 1 to 10"`
}

func (p *LogShowParams) entry() core.ShowLogEntry {
	return core.ShowLogEntry{
		Name:   p.Name,
		Season: p.Season,
		Date:   p.Date,
		Rating: p.Rating,
	}
}

func (s *Server) mcpLogShow(ctx context.Context, req *mcp.CallToolRequest, params *LogShowParams) (*mcp.CallToolResult, any, error) {
	entry := params.entry()
	err := s.core.AddShowLog(entry)
	if err != nil {
		return nil, nil, err
	}

	return nil, entry, nil
}

type EditMovieLogParams struct {
	OriginalName string    `json:"originalName" jsonschema:"Current name of the movie entry to edit"`
	OriginalDate time.Time `json:"originalDate" jsonschema:"Current date of the movie entry to edit, exactly as returned by the get tool for this log"`
	LogMovieParams
}

func (s *Server) mcpEditMovieLog(ctx context.Context, req *mcp.CallToolRequest, params *EditMovieLogParams) (*mcp.CallToolResult, any, error) {
	entry := params.entry()
	err := s.core.EditMovieLog(params.OriginalDate, params.OriginalName, entry)
	if err != nil {
		return nil, nil, err
	}

	return nil, entry, nil
}

type EditPerformanceLogParams struct {
	OriginalName   string    `json:"originalName,omitempty" jsonschema:"Current name of the performance entry to edit. At least one of originalName and originalAuthor is required; only the ones given are matched"`
	OriginalAuthor string    `json:"originalAuthor,omitempty" jsonschema:"Current author of the performance entry to edit. At least one of originalName and originalAuthor is required; only the ones given are matched"`
	OriginalDate   time.Time `json:"originalDate" jsonschema:"Current date of the performance entry to edit, exactly as returned by the get tool for this log"`
	LogPerformanceParams
}

func (s *Server) mcpEditPerformanceLog(ctx context.Context, req *mcp.CallToolRequest, params *EditPerformanceLogParams) (*mcp.CallToolResult, any, error) {
	entry := params.entry()
	err := s.core.EditPerformanceLog(params.OriginalDate, params.OriginalName, params.OriginalAuthor, entry)
	if err != nil {
		return nil, nil, err
	}

	return nil, entry, nil
}

type EditReadingLogParams struct {
	OriginalName string    `json:"originalName" jsonschema:"Current name of the reading entry to edit"`
	OriginalDate time.Time `json:"originalDate" jsonschema:"Current date of the reading entry to edit, exactly as returned by the get tool for this log"`
	LogReadingParams
}

func (s *Server) mcpEditReadingLog(ctx context.Context, req *mcp.CallToolRequest, params *EditReadingLogParams) (*mcp.CallToolResult, any, error) {
	entry := params.entry()
	err := s.core.EditReadingLog(params.OriginalDate, params.OriginalName, entry)
	if err != nil {
		return nil, nil, err
	}

	return nil, entry, nil
}

type EditShowLogParams struct {
	OriginalName   string    `json:"originalName" jsonschema:"Current name of the show entry to edit"`
	OriginalDate   time.Time `json:"originalDate" jsonschema:"Current date of the show entry to edit, exactly as returned by the get tool for this log"`
	OriginalSeason int       `json:"originalSeason,omitempty" jsonschema:"Current season of the show entry to edit, omit if it has none"`
	LogShowParams
}

func (s *Server) mcpEditShowLog(ctx context.Context, req *mcp.CallToolRequest, params *EditShowLogParams) (*mcp.CallToolResult, any, error) {
	entry := params.entry()
	err := s.core.EditShowLog(params.OriginalDate, params.OriginalName, params.OriginalSeason, entry)
	if err != nil {
		return nil, nil, err
	}

	return nil, entry, nil
}
