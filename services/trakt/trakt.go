package trakt

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/hacdias/eagle/eagle"
	"github.com/hacdias/eagle/fs"
	"github.com/hacdias/eagle/util"
	"golang.org/x/oauth2"
)

const (
	entryID         = "/watches/summary"
	historyFileName = ".history.json"
	summaryStartTag = "<!--WATCHES-->"
	summaryEndTag   = "<!--/WATCHES-->"
)

type Trakt struct {
	config *eagle.Trakt
	oauth2 *oauth2.Config
	token  *oauth2.Token
	fs     *fs.FS
}

func NewTrakt(c *eagle.Trakt, fs *fs.FS) (*Trakt, error) {
	t := &Trakt{
		config: c,
		fs:     fs,
		oauth2: &oauth2.Config{
			ClientID:     c.Client,
			ClientSecret: c.Secret,
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://trakt.tv/oauth/authorize",
				TokenURL: "https://trakt.tv/oauth/token",
			},
		},
	}

	raw, err := os.ReadFile(c.Token)
	if err == nil {
		err = json.Unmarshal(raw, &t.token)
		if err != nil {
			return nil, err
		}
	}

	return t, nil
}

func (t *Trakt) InteractiveLogin(port int) error {
	state, err := randomString(10)
	if err != nil {
		return nil
	}

	t.oauth2.RedirectURL = fmt.Sprintf("http://localhost:%d/callback", port)

	url := t.oauth2.AuthCodeURL(state)
	fmt.Printf("Please open the following URL, authenticate, and close the tab:\n%s\n", url)

	request := make(chan *http.Request, 1)

	server := &http.Server{Addr: ":" + strconv.Itoa(port), Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		request <- r
	})}

	go func() {
		_ = server.ListenAndServe()
	}()
	defer func() {
		_ = server.Shutdown(context.Background())
	}()

	r := <-request

	if s := r.URL.Query().Get("state"); s != state {
		return errors.New("state does not match")
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		return errors.New("code was empty")
	}

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()
	token, err := t.oauth2.Exchange(ctx, code)
	if err != nil {
		return nil
	}

	t.token = token
	err = t.updateToken()
	if err != nil {
		return nil
	}

	fmt.Println("Token updated.")
	return nil
}

func (t *Trakt) Fetch(ctx context.Context, page int, start time.Time, end time.Time) (traktHistory, bool, error) {
	limit := 100
	u, err := url.Parse("https://api.trakt.tv/sync/history")
	if err != nil {
		return nil, false, err
	}

	q := u.Query()
	q.Set("extended", "full")
	q.Set("limit", strconv.Itoa(limit))
	q.Set("page", strconv.Itoa(page))

	if !start.IsZero() {
		q.Set("start_at", start.Format(time.RFC3339Nano))
	}

	if !end.IsZero() {
		q.Set("end_at", end.Format(time.RFC3339Nano))
	}

	u.RawQuery = q.Encode()

	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, false, err
	}

	t.oauth2.Client(ctx, t.token)

	httpClient := t.oauth2.Client(ctx, t.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("trakt-api-key", t.config.Client)
	req.Header.Set("trakt-api-version", "2")

	res, err := httpClient.Do(req)
	if err != nil {
		return nil, false, err
	}
	defer res.Body.Close()

	currentPage, err := strconv.Atoi(res.Header.Get("X-Pagination-Page"))
	if err != nil {
		return nil, false, err
	}

	totalPages, err := strconv.Atoi(res.Header.Get("X-Pagination-Page-Count"))
	if err != nil {
		return nil, false, err
	}

	var history traktHistory

	err = json.NewDecoder(res.Body).Decode(&history)
	if err != nil {
		return nil, false, err
	}

	return history, currentPage < totalPages, nil
}

func (t *Trakt) FetchAll(ctx context.Context) error {
	var (
		err         error
		page        = 1
		history     traktHistory
		historyPage traktHistory
		hasMore     = true
	)

	for hasMore {
		historyPage, hasMore, err = t.Fetch(ctx, page, time.Time{}, time.Time{})
		if err != nil {
			return err
		}

		history = append(history, historyPage...)
		time.Sleep(time.Millisecond * 500) // Beware of those rate limitings.
		page++
	}

	err = t.saveHistory(history)
	if err != nil {
		return err
	}

	return t.updateToken()
}

func (t *Trakt) GenerateSummary() (*summary, error) {
	var (
		err           error
		moviesCount   = 0
		episodesCount = 0
		history       traktHistory
		moviesMap     = map[int]traktHistoryItem{}
		showsMap      = map[int]traktHistoryItem{}
	)

	history, err = t.loadHistory()
	if err != nil {
		return nil, err
	}

	for _, h := range history {
		if h.Type == "movie" {
			moviesCount++

			if h.Movie.IDs.Trakt == 0 {
				return nil, fmt.Errorf("movie id is invalid: %s", h.Movie.Title)
			}

			if v, ok := moviesMap[h.Movie.IDs.Trakt]; ok {
				if v.WatchedAt.Before(h.WatchedAt) {
					moviesMap[h.Movie.IDs.Trakt] = h
				}
			} else {
				moviesMap[h.Movie.IDs.Trakt] = h
			}
		} else if h.Type == "episode" {
			episodesCount++

			if h.Show.IDs.Trakt == 0 {
				return nil, fmt.Errorf("show id is invalid: %s", h.Show.Title)
			}

			if v, ok := showsMap[h.Show.IDs.Trakt]; ok {
				if v.WatchedAt.Before(h.WatchedAt) {
					showsMap[h.Show.IDs.Trakt] = h
				}
			} else {
				showsMap[h.Show.IDs.Trakt] = h
			}
		} else {
			return nil, fmt.Errorf("unknown type: %s", h.Type)
		}
	}

	var (
		movies = traktHistory{}
		shows  = traktHistory{}
	)

	for _, movie := range moviesMap {
		movies = append(movies, movie)
	}

	for _, show := range showsMap {
		shows = append(shows, show)
	}

	movies.sort()
	shows.sort()

	return &summary{
		UniqueMovies:  len(moviesMap),
		UniqueShows:   len(showsMap),
		TotalMovies:   moviesCount,
		TotalEpisodes: episodesCount,
		TotalWatches:  moviesCount + episodesCount,
		Movies:        movies,
		Shows:         shows,
	}, nil
}

func (t *Trakt) UpdateWatches() error {
	s, err := t.GenerateSummary()
	if err != nil {
		return err
	}

	_, err = t.fs.TransformEntry(entryID, func(e *eagle.Entry) (*eagle.Entry, error) {
		var err error
		e.Updated = time.Now()
		e.Content, err = util.ReplaceInBetween(e.Content, summaryStartTag, summaryEndTag, s.toMarkdown())
		return e, err
	})
	return err
}

func (t *Trakt) FetchAndUpdateWatches() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*5)
	defer cancel()

	err := t.FetchAll(ctx)
	if err != nil {
		return err
	}

	return t.UpdateWatches()
}

func (t *Trakt) updateToken() error {
	raw, err := json.MarshalIndent(t.token, "", "  ")
	if err != nil {
		return nil
	}

	return os.WriteFile(t.config.Token, raw, 0644)
}

func (t *Trakt) saveHistory(history traktHistory) error {
	raw, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return nil
	}

	return t.fs.WriteFile(filepath.Join(fs.ContentDirectory, entryID, historyFileName), raw, "update trakt history")
}

func (t *Trakt) loadHistory() (traktHistory, error) {
	raw, err := t.fs.ReadFile(filepath.Join(fs.ContentDirectory, entryID, historyFileName))
	if err != nil {
		return nil, err
	}

	var h traktHistory
	return h, json.Unmarshal(raw, &h)
}
