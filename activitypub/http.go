package activitypub

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/hacdias/eagle/pkg/contenttype"
)

const (
	followersPerPage = 50
)

func (ap *ActivityPub) FollowersHandler(w http.ResponseWriter, r *http.Request) (int, error) {
	if r.Method != http.MethodGet {
		return http.StatusMethodNotAllowed, errors.New("method not allowed")
	}

	count, err := ap.Store.GetFollowersCount()
	if err != nil {
		return http.StatusInternalServerError, err
	}

	pageQuery := r.URL.Query().Get("page")

	if pageQuery == "" {
		ap.serve(w, http.StatusOK, map[string]interface{}{
			"@context":   "https://www.w3.org/ns/activitystreams",
			"id":         ap.Options.FollowersURL,
			"type":       "OrderedCollection",
			"totalItems": count,
			"first":      ap.Options.FollowersURL + "?page=1",
		})
		return 0, nil
	}

	page, err := strconv.Atoi(pageQuery)
	if err != nil {
		return http.StatusBadRequest, err
	}

	if page < 1 {
		return http.StatusBadRequest, errors.New("page number is invalid, must be >= 1")
	}

	followers, err := ap.Store.GetFollowersByPage(page, followersPerPage)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	activity := map[string]interface{}{
		"@context":   "https://www.w3.org/ns/activitystreams",
		"id":         ap.Options.FollowersURL + "?page=" + pageQuery,
		"partOf":     ap.Options.FollowersURL,
		"type":       "OrderedCollectionPage",
		"totalItems": count,
	}

	items := []string{}
	for _, f := range followers {
		items = append(items, f.ID)
	}
	activity["orderedItems"] = items

	if len(followers) == followersPerPage {
		activity["next"] = ap.Options.FollowersURL + "?page=" + strconv.Itoa(page+1)
	}

	ap.serve(w, http.StatusOK, activity)
	return 0, nil
}

func (ap *ActivityPub) serve(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", contenttype.ASUTF8)
	w.WriteHeader(code)
	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		ap.Notifier.Error(fmt.Errorf("serving activity: %w", err))
	}
}

func (ap *ActivityPub) RemoteFollowHandler(w http.ResponseWriter, r *http.Request) (int, error) {
	if r.Method != http.MethodPost {
		return http.StatusMethodNotAllowed, errors.New("method not allowed")
	}

	if err := r.ParseForm(); err != nil {
		return http.StatusBadRequest, err
	}

	handle := strings.TrimPrefix(r.Form.Get("handle"), "@")
	handleParts := strings.Split(handle, "@")
	if len(handleParts) != 2 {
		return http.StatusBadRequest, errors.New("user handle must be in form of user@example.org or @user@example.org")
	}

	user := handleParts[0]
	instance := handleParts[1]
	if user == "" || instance == "" {
		return http.StatusBadRequest, errors.New("user handle must be in form of user@example.org or @user@example.org")
	}

	webFinger, err := ap.getWebFinger(r.Context(), instance, handle)
	if err != nil {
		if err == errNotFound {
			return http.StatusNotFound, nil
		}

		return http.StatusInternalServerError, err
	}

	template := ""
	for _, link := range webFinger.Links {
		if link.Rel == "http://ostatus.org/schema/1.0/subscribe" {
			template = link.Template
			break
		}
	}

	if template == "" {
		return http.StatusBadRequest, errors.New("instance does not support subscribe schema version 1.0")
	}

	redirect := strings.ReplaceAll(template, "{uri}", ap.Config.Server.BaseURL)
	http.Redirect(w, r, redirect, http.StatusSeeOther)
	return 0, nil
}
