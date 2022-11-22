package activitypub

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/hacdias/eagle/pkg/contenttype"
)

const (
	followersPerPage = 50
)

func (ap *ActivityPub) FollowersCollectionHandler(w http.ResponseWriter, r *http.Request) (int, error) {
	if r.Method != http.MethodGet {
		return http.StatusMethodNotAllowed, errors.New("method not allowed")
	}

	count, err := ap.Storage.GetFollowersCount()
	if err != nil {
		return http.StatusInternalServerError, err
	}

	pageQuery := r.URL.Query().Get("page")

	if pageQuery == "" {
		ap.serve(w, http.StatusOK, map[string]interface{}{
			"@context":   "https://www.w3.org/ns/activitystreams",
			"id":         ap.options.FollowersURL,
			"type":       "OrderedCollection",
			"totalItems": count,
			"first":      ap.options.FollowersURL + "?page=1",
		})
		return 0, nil
	}

	page, err := strconv.Atoi(pageQuery)
	if err != nil {
		return http.StatusBadRequest, err
	}

	followers, err := ap.Storage.GetFollowersByPage(page, followersPerPage)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	activity := map[string]interface{}{
		"@context":   "https://www.w3.org/ns/activitystreams",
		"id":         ap.options.FollowersURL + "?page=" + pageQuery,
		"partOf":     ap.options.FollowersURL,
		"type":       "OrderedCollectionPage",
		"totalItems": count,
	}

	items := []string{}
	for _, f := range followers {
		items = append(items, f.ID)
	}
	activity["orderedItems"] = items

	if len(followers) == followersPerPage {
		activity["next"] = ap.options.FollowersURL + "?page=" + strconv.Itoa(page+1)
	}

	ap.serve(w, http.StatusOK, activity)
	return 0, nil
}

func (ap *ActivityPub) serve(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", contenttype.ASUTF8)
	w.WriteHeader(code)
	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		ap.n.Error(fmt.Errorf("serving activity: %w", err))
	}
}
