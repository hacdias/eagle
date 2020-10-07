package server

import (
	"net/http"
	"time"

	"github.com/hacdias/eagle/config"
	"github.com/hacdias/eagle/micropub"
	"github.com/hacdias/eagle/services"
)

type micropubHandlerFunc func(w http.ResponseWriter, r *http.Request, mr *micropub.Request) (int, error)

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
			s.Notify.Error(err)
		}

		go s.Gossip(entry, synd)

		return 0, nil
	}
}

func micropubUpdate(s *services.Services, c *config.Config) micropubHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, mr *micropub.Request) (int, error) {

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
