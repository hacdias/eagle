package server

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/hacdias/eagle/v2/eagle"
	"github.com/hacdias/eagle/v2/pkg/mf2"
	"github.com/hacdias/eagle/v2/pkg/micropub"
	"github.com/karlseguin/typed"
)

func (s *Server) micropubGet(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Query().Get("q") {
	case "source":
		s.micropubSource(w, r)
	case "config", "syndicate-to":
		s.micropubConfig(w, r)
	default:
		s.serveErrorHTML(w, r, http.StatusNotFound, nil)
	}
}

func (s *Server) micropubSource(w http.ResponseWriter, r *http.Request) {
	id, err := s.micropubParseURL(r.URL.Query().Get("url"))
	if err != nil {
		s.serveErrorJSON(w, http.StatusBadRequest, err)
		return
	}

	entry, err := s.GetEntry(id)
	if err != nil {
		if os.IsNotExist(err) {
			s.serveErrorJSON(w, http.StatusNotFound, fmt.Errorf("post not found: %s", id))
		} else {
			s.serveErrorJSON(w, http.StatusInternalServerError, err)
		}
		return
	}

	s.serveJSON(w, http.StatusOK, entry.ToMF2())
}

func (s *Server) micropubConfig(w http.ResponseWriter, r *http.Request) {
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
}

func (s *Server) micropubPost(w http.ResponseWriter, r *http.Request) {
	mr, err := micropub.ParseRequest(r)
	if err != nil {
		s.serveErrorJSON(w, http.StatusBadRequest, err)
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
		s.serveErrorJSON(w, code, err)
	} else if err != nil {
		s.NotifyError(fmt.Errorf("micropub: %w", err))
	}
}

func (s *Server) micropubCreate(w http.ResponseWriter, r *http.Request, mr *micropub.Request) (int, error) {
	cmds := typed.New(mf2.Flatten(mr.Commands))
	slug := ""
	if s, ok := cmds.StringIf("mp-slug"); ok {
		slug = s
	}

	entry, err := s.EntryFromMF2(mr.Properties, slug)
	if err != nil {
		return http.StatusBadRequest, err
	}

	// TODO: parse this to add twitter
	// if targets, ok := post.Commands.StringsIf("mp-syndicate-to"); ok {
	// 	synd.Targets = targets
	// }

	err = s.savePost(entry, &saveOptions{})
	if err != nil {
		return http.StatusInternalServerError, err
	}

	http.Redirect(w, r, s.Config.BaseURL+entry.ID, http.StatusAccepted)
	return 0, nil
}

func (s *Server) micropubUpdate(w http.ResponseWriter, r *http.Request, mr *micropub.Request) (int, error) {
	id, err := s.micropubParseURL(mr.URL)
	if err != nil {
		return http.StatusBadRequest, err
	}

	entry, err := s.TransformEntry(id, func(entry *eagle.Entry) (*eagle.Entry, error) {
		mf := entry.ToMF2()
		props := mf["properties"].(map[string][]interface{})
		newMf, err := micropub.Update(props, mr)
		if err != nil {
			return nil, err
		}

		err = s.UpdateEntryWithMF2(entry, newMf)
		if err != nil {
			return nil, err
		}

		return entry, nil
	})
	if err != nil {
		return http.StatusBadRequest, err
	}

	http.Redirect(w, r, entry.Permalink, http.StatusOK)

	return 0, s.savePost(entry, &saveOptions{
		skipSave: true,
	})
}

func (s *Server) micropubUnremove(w http.ResponseWriter, r *http.Request, mr *micropub.Request) (int, error) {
	id, err := s.micropubParseURL(mr.URL)
	if err != nil {
		return http.StatusBadRequest, err
	}

	entry, err := s.TransformEntry(id, func(entry *eagle.Entry) (*eagle.Entry, error) {
		entry.Deleted = false
		return entry, nil
	})
	if err != nil {
		return http.StatusInternalServerError, err
	}

	err = s.savePost(entry, &saveOptions{
		skipSave: true,
	})
	if err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

func (s *Server) micropubRemove(w http.ResponseWriter, r *http.Request, mr *micropub.Request) (int, error) {
	id, err := s.micropubParseURL(mr.URL)
	if err != nil {
		return http.StatusBadRequest, err
	}

	entry, err := s.TransformEntry(id, func(entry *eagle.Entry) (*eagle.Entry, error) {
		entry.Deleted = true
		return entry, nil
	})
	if err != nil {
		return http.StatusInternalServerError, err
	}

	err = s.savePost(entry, &saveOptions{
		skipSave: true,
	})
	if err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

func (s *Server) micropubParseURL(url string) (string, error) {
	if url == "" {
		return "", errors.New("url must be set")
	}

	if !strings.HasPrefix(url, s.Config.BaseURL) {
		return "", errors.New("invalid domain in url")
	}

	return strings.Replace(url, s.Config.BaseURL, "", 1), nil
}
