package server

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/hacdias/eagle/middleware/micropub"
)

func (s *Server) postMicropubHandler(w http.ResponseWriter, r *http.Request) {
	s.Lock()
	defer s.Unlock()

	mr, err := micropub.ParseRequest(r)
	if err != nil {
		s.serveError(w, http.StatusBadRequest, err)
		return
	}

	var code int

	switch mr.Action {
	case micropub.ActionCreate:
		code, err = s.micropubCreate(w, r, mr)
	case micropub.ActionUpdate:
		code, err = s.micropubUpdate(w, r, mr)
	case micropub.ActionDelete:
		code, err = s.micropubRemove(w, r, mr)
	case micropub.ActionUndelete:
		code, err = s.micropubUnremove(w, r, mr)
	default:
		code, err = http.StatusBadRequest, errors.New("invalid action")
	}

	if code >= 200 && code < 400 {
		w.WriteHeader(code)
	} else if code >= 400 {
		s.Errorf("micropub: error on post: %s", err)
		s.serveError(w, code, err)
	}

	if mr.Action == micropub.ActionCreate {
		return
	}

	err = s.Hugo.Build(mr.Action == micropub.ActionDelete)
	if err != nil {
		s.Errorf("micropub: error hugo build: %s", err)
		s.Notify.Error(err)
	}
}

func (s *Server) micropubCreate(w http.ResponseWriter, r *http.Request, mr *micropub.Request) (int, error) {
	s.Debug("micropub: create request received")
	entry, synd, err := s.Hugo.FromMicropub(mr)
	if err != nil {
		return http.StatusBadRequest, err
	}

	err = s.Hugo.SaveEntry(entry)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	url := s.c.Domain + entry.ID
	http.Redirect(w, r, url, http.StatusAccepted)

	err = s.Store.Persist("add " + entry.ID)
	if err != nil {
		s.Errorf("micropub: error git commit: %s", err)
		s.Notify.Error(err)
	}

	err = s.Hugo.Build(false)
	if err != nil {
		s.Errorf("micropub: error hugo build: %s", err)
		s.Notify.Error(err)
	}

	for _, rel := range synd.Related {
		err = s.XRay.RequestAndSave(rel)
		if err != nil {
			s.Warnf("could not xray %s: %s", rel, err)
			s.Notify.Error(err)
		}
	}

	go s.Gossip(entry, synd)
	s.Debug("micropub: create request ok")
	return 0, nil
}

func (s *Server) micropubUpdate(w http.ResponseWriter, r *http.Request, mr *micropub.Request) (int, error) {
	s.Debug("micropub: update request received")
	id := strings.Replace(mr.URL, s.c.Domain, "", 1)
	entry, err := s.Hugo.GetEntry(id)
	if err != nil {
		s.Errorf("micropub: cannot get entry: %s", err)
		return http.StatusBadRequest, err
	}

	err = entry.Update(mr)
	if err != nil {
		s.Errorf("micropub: cannot update entry: %s", err)
		return http.StatusBadRequest, err
	}

	err = s.Hugo.SaveEntry(entry)
	if err != nil {
		s.Errorf("micropub: cannot save entry: %s", err)
		return http.StatusInternalServerError, err
	}

	err = s.Store.Persist("update " + entry.ID)
	if err != nil {
		s.Errorf("micropub: cannot git commit: %s", err)
		return http.StatusInternalServerError, err
	}

	http.Redirect(w, r, mr.URL, http.StatusOK)
	s.Debug("micropub: update request ok")
	return 0, nil
}

func (s *Server) micropubUnremove(w http.ResponseWriter, r *http.Request, mr *micropub.Request) (int, error) {
	s.Debug("micropub: unremove request received")
	id, err := s.micropubParseURL(mr.URL)
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

	err = s.Store.Persist("undelete " + id)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	s.Debug("micropub: unremove request ok")
	return http.StatusOK, nil
}

func (s *Server) micropubRemove(w http.ResponseWriter, r *http.Request, mr *micropub.Request) (int, error) {
	s.Debug("micropub: remove request received")
	id, err := s.micropubParseURL(mr.URL)
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

	err = s.Store.Persist("delete " + id)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	s.Debug("micropub: remove request ok")
	return http.StatusOK, nil
}
