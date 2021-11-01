package server

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/hacdias/eagle/v2/eagle"
	"github.com/hacdias/eagle/v2/pkg/micropub"
	"github.com/karlseguin/typed"
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
	id, err := s.micropubParseURL(r.URL.Query().Get("url"))
	if err != nil {
		s.serveError(w, http.StatusBadRequest, err)
		return
	}

	entry, err := s.GetEntry(id)
	if err != nil {
		if os.IsNotExist(err) {
			s.serveError(w, http.StatusNotFound, fmt.Errorf("post not found: %s", id))
		} else {
			s.serveError(w, http.StatusInternalServerError, err)
		}
		return
	}

	mf2 := map[string]interface{}{
		"type":       []string{"h-entry"},
		"properties": s.ToMicroformats(entry),
	}

	s.serveJSON(w, http.StatusOK, mf2)
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
}

func (s *Server) micropubCreate(w http.ResponseWriter, r *http.Request, mr *micropub.Request) (int, error) {
	year := time.Now().Year()
	month := time.Now().Month()
	day := time.Now().Day()
	id := fmt.Sprintf("/%04d/%02d/%02d", year, month, day)

	cmds := typed.New(mr.Commands)

	if slugSlice, ok := cmds.StringsIf("mp-slug"); ok && len(slugSlice) == 1 {
		slug := strings.TrimSpace(strings.Join(slugSlice, "\n"))
		id += "/" + slug
	} else {
		// TODO: generate something
		return http.StatusBadRequest, errors.New("slug is required")
	}

	entry, err := s.FromMicroformats(id, mr.Properties)
	if err != nil {
		return http.StatusBadRequest, err
	}

	err = s.SaveEntry(entry)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	// if targets, ok := post.Commands.StringsIf("mp-syndicate-to"); ok {
	// 	synd.Targets = targets
	// }

	// TODO: xray related, syndicate, webmentions.

	http.Redirect(w, r, s.Config.BaseURL+entry.ID, http.StatusAccepted)
	return 0, nil
}

func (s *Server) micropubUpdate(w http.ResponseWriter, r *http.Request, mr *micropub.Request) (int, error) {
	id, err := s.micropubParseURL(mr.URL)
	if err != nil {
		return http.StatusBadRequest, err
	}

	entry, err := s.TransformEntry(id, func(entry *eagle.Entry) (*eagle.Entry, error) {
		mf := s.ToMicroformats(entry)

		newMf, err := micropub.Update(mf, mr)
		if err != nil {
			return nil, err
		}

		err = s.UpdateEntry(entry, newMf)
		if err != nil {
			return nil, err
		}

		return entry, nil
	})
	if err != nil {
		return http.StatusBadRequest, err
	}

	http.Redirect(w, r, entry.Permalink, http.StatusOK)
	// TODO: cache, xray related, syndicate, webmentions.
	return 0, nil
}

func (s *Server) micropubUnremove(w http.ResponseWriter, r *http.Request, mr *micropub.Request) (int, error) {
	id, err := s.micropubParseURL(mr.URL)
	if err != nil {
		return http.StatusBadRequest, err
	}

	_, err = s.TransformEntry(id, func(entry *eagle.Entry) (*eagle.Entry, error) {
		entry.Deleted = false
		return entry, nil
	})
	if err != nil {
		return http.StatusInternalServerError, err
	}

	// TODO: syndicate, webmentions.

	return http.StatusOK, nil
}

func (s *Server) micropubRemove(w http.ResponseWriter, r *http.Request, mr *micropub.Request) (int, error) {
	id, err := s.micropubParseURL(mr.URL)
	if err != nil {
		return http.StatusBadRequest, err
	}

	_, err = s.TransformEntry(id, func(entry *eagle.Entry) (*eagle.Entry, error) {
		entry.Deleted = true
		return entry, nil
	})
	if err != nil {
		return http.StatusInternalServerError, err
	}

	// TODO: syndicate, webmentions.

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
