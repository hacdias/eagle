package logs

import (
	"context"
	"strings"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

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

func (l *Logs) RegisterMCPTools(srv *mcp.Server) {
	// TODO: consider adding "after" / "before" query parameters for filtering logs by date.
	addLogTools[MovieLogEntry, *LogMovieParams, *EditMovieLogParams](srv, l, "Movie", moviesFile,
		"Edit a logged movie, identified by its original name and date. All fields are replaced",
		mcpLogInputSchema[LogMovieParams](), mcpLogInputSchema[EditMovieLogParams]())

	addLogTools[PerformanceLogEntry, *LogPerformanceParams, *EditPerformanceLogParams](srv, l, "Performance", performancesFile,
		"Edit a logged performance, identified by its original date and its original name, author, or both. All fields are replaced",
		mcpPerformanceInputSchema[LogPerformanceParams](), mcpPerformanceInputSchema[EditPerformanceLogParams]())

	addLogTools[ReadingLogEntry, *LogReadingParams, *EditReadingLogParams](srv, l, "Reading", readingsFile,
		"Edit a logged reading, identified by its original name and date. All fields are replaced",
		mcpLogInputSchema[LogReadingParams](), mcpLogInputSchema[EditReadingLogParams]())

	addLogTools[ShowLogEntry, *LogShowParams, *EditShowLogParams](srv, l, "Show", showsFile,
		"Edit a logged show, identified by its original name, date and season. All fields are replaced",
		mcpLogInputSchema[LogShowParams](), mcpLogInputSchema[EditShowLogParams]())
}

type mcpLogs[T any] struct {
	Logs []T `json:"logs"`
}

// addLogTools registers the tools to get, log and edit the entries of a log.
// The tool names and descriptions are derived from noun.
func addLogTools[T logEntry, P interface{ entry() T }, E interface {
	entry() T
	matches(T) bool
}](srv *mcp.Server, l *Logs, noun, file, editDescription string, logSchema, editSchema *jsonschema.Schema) {
	lowerNoun := strings.ToLower(noun)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "get" + noun + "Logs",
		Description: "Get all " + lowerNoun + " logs",
		Annotations: mcpReadAnnotations,
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, *mcpLogs[T], error) {
		logs, err := l.get[T](file)
		if err != nil {
			return nil, nil, err
		}

		return nil, &mcpLogs[T]{Logs: logs}, nil
	})

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "log" + noun,
		Description: "Log a " + lowerNoun,
		Annotations: mcpLogAnnotations,
		InputSchema: logSchema,
	}, func(ctx context.Context, req *mcp.CallToolRequest, params P) (*mcp.CallToolResult, any, error) {
		entry := params.entry()
		err := l.add(file, entry)
		if err != nil {
			return nil, nil, err
		}

		return nil, entry, nil
	})

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "edit" + noun + "Log",
		Description: editDescription,
		Annotations: mcpEditAnnotations,
		InputSchema: editSchema,
	}, func(ctx context.Context, req *mcp.CallToolRequest, params E) (*mcp.CallToolResult, any, error) {
		entry := params.entry()
		err := l.edit(file, params.matches, entry)
		if err != nil {
			return nil, nil, err
		}

		return nil, entry, nil
	})
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

func (p *LogMovieParams) entry() MovieLogEntry {
	return MovieLogEntry{
		Name:     p.Name,
		Date:     p.Date,
		Rating:   p.Rating,
		Location: p.Location,
		Room:     p.Room,
		Row:      p.Row,
		Seat:     p.Seat,
	}
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

func (p *LogPerformanceParams) entry() PerformanceLogEntry {
	return PerformanceLogEntry{
		Name:          p.Name,
		Author:        p.Author,
		Rating:        p.Rating,
		Date:          p.Date,
		Location:      p.Location,
		Category:      p.Category,
		SupportingAct: p.SupportingAct,
	}
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

func (p *LogReadingParams) entry() ReadingLogEntry {
	return ReadingLogEntry{
		Name:      p.Name,
		Author:    p.Author,
		Rating:    p.Rating,
		Publisher: p.Publisher,
		Pages:     p.Pages,
		Date:      p.Date,
		UID:       p.UID,
	}
}

type LogShowParams struct {
	Name   string    `json:"name" jsonschema:"Title of the show"`
	Season int       `json:"season,omitempty" jsonschema:"Season number. Omit for shows without seasons, such as miniseries"`
	Date   time.Time `json:"date" jsonschema:"When the show or season was watched. A full RFC 3339 date-time with time and offset, e.g. 2026-10-16T21:30:00+01:00; a bare date is rejected"`
	Rating int       `json:"rating" jsonschema:"Rating of the show or season, from 1 to 10"`
}

func (p *LogShowParams) entry() ShowLogEntry {
	return ShowLogEntry{
		Name:   p.Name,
		Season: p.Season,
		Date:   p.Date,
		Rating: p.Rating,
	}
}

type EditMovieLogParams struct {
	OriginalName string    `json:"originalName" jsonschema:"Current name of the movie entry to edit"`
	OriginalDate time.Time `json:"originalDate" jsonschema:"Current date of the movie entry to edit, exactly as returned by the get tool for this log"`
	LogMovieParams
}

func (p *EditMovieLogParams) matches(m MovieLogEntry) bool {
	return m.Date.Equal(p.OriginalDate) && m.Name == p.OriginalName
}

type EditPerformanceLogParams struct {
	OriginalName   string    `json:"originalName,omitempty" jsonschema:"Current name of the performance entry to edit. At least one of originalName and originalAuthor is required; only the ones given are matched"`
	OriginalAuthor string    `json:"originalAuthor,omitempty" jsonschema:"Current author of the performance entry to edit. At least one of originalName and originalAuthor is required; only the ones given are matched"`
	OriginalDate   time.Time `json:"originalDate" jsonschema:"Current date of the performance entry to edit, exactly as returned by the get tool for this log"`
	LogPerformanceParams
}

// matches compares the original name, author, or both. Empty values are not
// compared, but nothing matches if both are empty.
func (p *EditPerformanceLogParams) matches(e PerformanceLogEntry) bool {
	if p.OriginalName == "" && p.OriginalAuthor == "" {
		return false
	}

	return e.Date.Equal(p.OriginalDate) &&
		(p.OriginalName == "" || e.Name == p.OriginalName) &&
		(p.OriginalAuthor == "" || e.Author == p.OriginalAuthor)
}

type EditReadingLogParams struct {
	OriginalName string    `json:"originalName" jsonschema:"Current name of the reading entry to edit"`
	OriginalDate time.Time `json:"originalDate" jsonschema:"Current date of the reading entry to edit, exactly as returned by the get tool for this log"`
	LogReadingParams
}

func (p *EditReadingLogParams) matches(r ReadingLogEntry) bool {
	return r.Date.Equal(p.OriginalDate) && r.Name == p.OriginalName
}

type EditShowLogParams struct {
	OriginalName   string    `json:"originalName" jsonschema:"Current name of the show entry to edit"`
	OriginalDate   time.Time `json:"originalDate" jsonschema:"Current date of the show entry to edit, exactly as returned by the get tool for this log"`
	OriginalSeason int       `json:"originalSeason,omitempty" jsonschema:"Current season of the show entry to edit, omit if it has none"`
	LogShowParams
}

func (p *EditShowLogParams) matches(s ShowLogEntry) bool {
	return s.Date.Equal(p.OriginalDate) && s.Name == p.OriginalName && s.Season == p.OriginalSeason
}
