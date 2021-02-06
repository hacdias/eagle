package server

import (
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

func (s *Server) StartHTTP() error {
	r := mux.NewRouter()
	// r.Use(s.recoverer)

	// r.Post("/webhook", s.webhookHandler)
	// r.Post("/webmention", s.webmentionHandler)
	// r.Post("/activitypub/inbox", s.activityPubPostInboxHandler)
	// r.Get("/search.json", s.searchHandler)

	// Make sure we have a built version!
	//should, err := s.Hugo.ShouldBuild()
	//if err != nil {
	//	return err
	//}
	//if should {
	//	err = s.Hugo.Build(false)
	//	if err != nil {
	//		return err
	//	}
	//}

	//s//tatic := s.staticHandler()
	//
	//r.NotFound(static)
	//.MethodNotAllowed(static)

	// NOTE:
	//	- Should I handle /now dynamicall?
	//	- Should I handle all redirects dynamically?

	/* r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		filePath := filepath.Join(s.c.Source, "content", urlPath+".md")
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		e, _ := eagle.NewEagle(s.c)
		entry, _ := e.GetEntry(urlPath)

		fmt.Println(entry.Metadata)

		err := s.HTML(entry, w)
		if err != nil {
			s.Error(err)
		}
	}) */

	r.Use(s.cleanPath)

	r.PathPrefix("/").HandlerFunc(s.serveAll)

	s.Infof("Listening on http://localhost:%d", s.c.Port)
	return http.ListenAndServe(":"+strconv.Itoa(s.c.Port), r)
}
