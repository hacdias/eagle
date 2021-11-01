package server

import (
	"errors"
	"net/http"

	"github.com/hacdias/eagle/v2/pkg/micropub"
)

func (s *Server) getMicropubHandler(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Query().Get("q") {
	case "source":
		s.micropubSource(w, r)
	case "config", "syndicate-to":
		// syndications := []map[string]string{}
		// for id, service := range s.Syndicator {
		// 	syndications = append(syndications, map[string]string{
		// 		"uid":  id,
		// 		"name": service.Name(),
		// 	})
		// }

		config := map[string]interface{}{
			// "syndicate-to": syndications,
		}

		s.serveJSON(w, http.StatusOK, config)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (s *Server) micropubSource(w http.ResponseWriter, r *http.Request) {
	s.serveError(w, http.StatusNotImplemented, errors.New("not implemented"))

	// s.Debug("micropub: source request received")
	// id, err := s.micropubParseURL(r.URL.Query().Get("url"))
	// if err != nil {
	// 	s.Errorf("micropub: cannot parse url: %s", err)
	// 	s.serveError(w, http.StatusBadRequest, err)
	// 	return
	// }

	// post, err := s.Hugo.GetEntry(id)
	// if err != nil {
	// 	if os.IsNotExist(err) {
	// 		s.Errorf("micropub: post not found: %s", err)
	// 		s.serveError(w, http.StatusNotFound, fmt.Errorf("post not found: %s", id))
	// 	} else {
	// 		s.Errorf("micropub: cannot get hugo entry: %s", err)
	// 		s.serveError(w, http.StatusBadRequest, err)
	// 	}
	// 	return
	// }

	// entry := map[string]interface{}{
	// 	"type": []string{"h-entry"},
	// }

	// props := post.Metadata["properties"].(map[string][]interface{})

	// if title, ok := post.Metadata.StringIf("title"); ok {
	// 	props["name"] = []interface{}{title}
	// }

	// if tags, ok := post.Metadata.StringsIf("tags"); ok {
	// 	props["category"] = []interface{}{}
	// 	for _, tag := range tags {
	// 		props["category"] = append(props["category"], tag)
	// 	}
	// }

	// if date, ok := post.Metadata.StringIf("lastmod"); ok {
	// 	props["published"] = []interface{}{date}
	// } else if date, ok := post.Metadata.StringIf("date"); ok {
	// 	props["published"] = []interface{}{date}
	// }

	// if post.Content != "" {
	// 	props["content"] = []interface{}{post.Content}
	// }

	// entry["properties"] = props
	// s.serveJSON(w, http.StatusOK, entry)
	// s.Debug("micropub: source request ok")
}

func (s *Server) postMicropubHandler(w http.ResponseWriter, r *http.Request) {
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
		s.log.Errorf("micropub: error on post: %s", err)
		s.serveError(w, code, err)
	}

	switch mr.Action {
	case micropub.ActionCreate:
	case micropub.ActionUpdate:
		return
	}

	// err = s.Hugo.Build(mr.Action == micropub.ActionDelete)
	// if err != nil {
	// 	s.Errorf("micropub: error hugo build: %s", err)
	// 	s.Notify.Error(err)
	// }
}

func (s *Server) micropubCreate(w http.ResponseWriter, r *http.Request, mr *micropub.Request) (int, error) {
	// s.Debug("micropub: create request received")
	// entry, synd, err := s.Hugo.FromMicropub(mr)
	// if err != nil {
	// 	return http.StatusBadRequest, err
	// }
	// s.Debugw("micropub: create request", "entry", entry, "synd", synd)

	// err = s.Hugo.SaveEntry(entry)
	// if err != nil {
	// 	return http.StatusInternalServerError, err
	// }

	// for _, rel := range synd.Related {
	// 	err = s.XRay.RequestAndSave(rel)
	// 	if err != nil {
	// 		s.Warnf("could not xray %s: %s", rel, err)
	// 		s.Notify.Error(err)
	// 	}
	// }

	// err = s.Store.Persist("add " + entry.ID)
	// if err != nil {
	// 	s.Errorf("micropub: error git commit: %s", err)
	// 	s.Notify.Error(err)
	// }

	// err = s.Hugo.Build(false)
	// if err != nil {
	// 	s.Errorf("micropub: error hugo build: %s", err)
	// 	s.Notify.Error(err)
	// }

	// url := s.c.Domain + entry.ID
	// http.Redirect(w, r, url, http.StatusAccepted)

	// go func() {
	// 	s.sendWebmentions(entry, synd.Related...)
	// 	s.syndicate(entry, synd)
	// 	s.activity(entry)
	// 	err := s.MeiliSearch.Add(entry)
	// 	if err != nil {
	// 		s.Warnf("could not add to meilisearch: %s", err)
	// 		s.Notify.Error(err)
	// 	}
	// }()

	// s.Debug("micropub: create request ok")

	http.Redirect(w, r, "https://example.com", http.StatusAccepted)
	return 0, nil
}

func (s *Server) micropubUpdate(w http.ResponseWriter, r *http.Request, mr *micropub.Request) (int, error) {
	// s.Debug("micropub: update request received")
	// id := strings.Replace(mr.URL, s.c.Domain, "", 1)
	// entry, err := s.Hugo.GetEntry(id)
	// if err != nil {
	// 	s.Errorf("micropub: cannot get entry: %s", err)
	// 	return http.StatusBadRequest, err
	// }

	// err = entry.Update(mr)
	// if err != nil {
	// 	s.Errorf("micropub: cannot update entry: %s", err)
	// 	return http.StatusBadRequest, err
	// }

	// err = s.Hugo.SaveEntry(entry)
	// if err != nil {
	// 	s.Errorf("micropub: cannot save entry: %s", err)
	// 	return http.StatusInternalServerError, err
	// }

	// err = s.Store.Persist("update " + entry.ID)
	// if err != nil {
	// 	s.Errorf("micropub: cannot git commit: %s", err)
	// 	return http.StatusInternalServerError, err
	// }

	// err = s.Hugo.Build(false)
	// if err != nil {
	// 	s.Errorf("micropub: error hugo build: %s", err)
	// 	s.Notify.Error(err)
	// }

	// http.Redirect(w, r, mr.URL, http.StatusOK)
	// s.Debug("micropub: update request ok")

	// go func() {
	// 	s.sendWebmentions(entry)
	// 	s.activity(entry)
	// 	err := s.MeiliSearch.Add(entry)
	// 	if err != nil {
	// 		s.Warnf("could not update meilisearch: %s", err)
	// 		s.Notify.Error(err)
	// 	}
	// }()

	return 0, nil
}

func (s *Server) micropubUnremove(w http.ResponseWriter, r *http.Request, mr *micropub.Request) (int, error) {
	// s.Debug("micropub: unremove request received")
	// id, err := s.micropubParseURL(mr.URL)
	// if err != nil {
	// 	return http.StatusBadRequest, err
	// }

	// entry, err := s.Hugo.GetEntry(id)
	// if err != nil {
	// 	return http.StatusBadRequest, err
	// }

	// delete(entry.Metadata, "expiryDate")

	// err = s.Hugo.SaveEntry(entry)
	// if err != nil {
	// 	return http.StatusInternalServerError, err
	// }

	// err = s.Store.Persist("undelete " + id)
	// if err != nil {
	// 	return http.StatusInternalServerError, err
	// }

	// go func() {
	// 	err := s.MeiliSearch.Add(entry)
	// 	if err != nil {
	// 		s.Warnf("could not add to meilisearch: %s", err)
	// 		s.Notify.Error(err)
	// 	}
	// }()

	// s.Debug("micropub: unremove request ok")
	return http.StatusOK, nil
}

func (s *Server) micropubRemove(w http.ResponseWriter, r *http.Request, mr *micropub.Request) (int, error) {
	// s.Debug("micropub: remove request received")
	// id, err := s.micropubParseURL(mr.URL)
	// if err != nil {
	// 	return http.StatusBadRequest, err
	// }

	// entry, err := s.Hugo.GetEntry(id)
	// if err != nil {
	// 	return http.StatusInternalServerError, err
	// }

	// entry.Metadata["expiryDate"] = time.Now().Format(time.RFC3339)

	// err = s.Hugo.SaveEntry(entry)
	// if err != nil {
	// 	return http.StatusInternalServerError, err
	// }

	// err = s.Store.Persist("delete " + id)
	// if err != nil {
	// 	return http.StatusInternalServerError, err
	// }

	// go func() {
	// 	err := s.MeiliSearch.Delete(entry)
	// 	if err != nil {
	// 		s.Warnf("could not remove from meilisearch: %s", err)
	// 		s.Notify.Error(err)
	// 	}
	// }()

	// s.Debug("micropub: remove request ok")
	return http.StatusOK, nil
}
