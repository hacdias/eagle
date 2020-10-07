package server

import (
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/hacdias/eagle/config"
	"github.com/hacdias/eagle/micropub"
	"github.com/hacdias/eagle/services"
)

func postMicropubHandler(s *services.Services, c *config.Config) http.HandlerFunc {
	create := micropubCreate(s, c)
	update := micropubUpdate(s, c)
	remove := micropubRemove(s, c)
	unremove := micropubUnremove(s, c)

	return func(w http.ResponseWriter, r *http.Request) {
		mr, err := micropub.ParseRequest(r)
		if err != nil {
			serveError(w, http.StatusBadRequest, err)
			return
		}

		var code int

		switch mr.Action {
		case micropub.ActionCreate:
			code, err = create(w, r, mr)
		case micropub.ActionUpdate:
			code, err = update(w, r, mr)
		case micropub.ActionDelete:
			code, err = remove(w, r, mr)
		case micropub.ActionUndelete:
			code, err = unremove(w, r, mr)
		default:
			code, err = http.StatusBadRequest, errors.New("invalid action")
		}

		if code >= 200 && code < 400 {
			w.WriteHeader(code)
		} else if code >= 400 {
			log.Printf("micropub: error on post: %s", err)
			serveError(w, code, err)
		}

		err = s.Hugo.Build(mr.Action == micropub.ActionDelete)
		if err != nil {
			log.Printf("micropub: error hugo build: %s", err)
			s.Notify.Error(err)
		}
	}
}

func micropubCreate(s *services.Services, c *config.Config) micropubHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, mr *micropub.Request) (int, error) {
		entry, synd, err := s.Hugo.FromMicropub(mr)
		if err != nil {
			return http.StatusBadRequest, err
		}

		err = s.Hugo.SaveEntry(entry)
		if err != nil {
			return http.StatusInternalServerError, err
		}

		url := c.Domain + entry.ID
		http.Redirect(w, r, url, http.StatusTemporaryRedirect)

		err = s.Git.Commit("add " + entry.ID)
		if err != nil {
			log.Printf("micropub: error git commit: %s", err)
			s.Notify.Error(err)
		}

		go s.Gossip(entry, synd)
		return 0, nil
	}
}

func micropubUpdate(s *services.Services, c *config.Config) micropubHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, mr *micropub.Request) (int, error) {
		// TODO(micropub): update handler
		return 0, nil
	}
}

func micropubUnremove(s *services.Services, c *config.Config) micropubHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, mr *micropub.Request) (int, error) {
		id, err := parseURL(c, mr.URL)
		if err != nil {
			return http.StatusBadRequest, err
		}

		entry, err := s.Hugo.GetEntry(id)
		if err != nil {
			return http.StatusBadRequest, err
		}

		delete(entry.Metadata, "expiryDate")

		err = s.Hugo.SaveEntry(entry)
		if err != nil {
			return http.StatusInternalServerError, err
		}

		err = s.Git.Commit("undelete " + id)
		if err != nil {
			return http.StatusInternalServerError, err
		}

		return http.StatusOK, nil
	}
}

func micropubRemove(s *services.Services, c *config.Config) micropubHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, mr *micropub.Request) (int, error) {
		id, err := parseURL(c, mr.URL)
		if err != nil {
			return http.StatusBadRequest, err
		}

		entry, err := s.Hugo.GetEntry(id)
		if err != nil {
			return http.StatusInternalServerError, err
		}

		entry.Metadata["expiryDate"] = time.Now().String()

		err = s.Hugo.SaveEntry(entry)
		if err != nil {
			return http.StatusInternalServerError, err
		}

		err = s.Git.Commit("delete " + id)
		if err != nil {
			return http.StatusInternalServerError, err
		}

		return http.StatusOK, nil
	}
}
