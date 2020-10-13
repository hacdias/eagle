package services

/*
type RemoteActor struct {
	iri, inbox, sharedInbox string
	info                    map[string]interface{}
}

func NewRemoteActor(iri string) (RemoteActor, error) {
	info, err := get(iri)
	if err != nil {
		return RemoteActor{}, err
	}
	inbox := (*info)["inbox"].(string)
	var endpoints map[string]interface{}
	var sharedInbox string
	if (*info)["endpoints"] != nil {
		endpoints = (*info)["endpoints"].(map[string]interface{})
		if val, ok := endpoints["sharedInbox"]; ok {
			sharedInbox = val.(string)
		}
	}
	return RemoteActor{
		iri:         iri,
		inbox:       inbox,
		sharedInbox: sharedInbox,
	}, err
}

func get(iri string) (info *map[string]interface{}, err error) {
	buf := new(bytes.Buffer)
	req, err := http.NewRequest("GET", iri, buf)
	if err != nil {
		return
	}
	req.Header.Add("Accept", ContentTypeAs2)
	req.Header.Add("User-Agent", fmt.Sprintf("%s %s", libName, version))
	req.Header.Add("Accept-Charset", "utf-8")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	if !isSuccess(resp.StatusCode) {
		return
	}
	var e map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&e)
	if err != nil {
		return
	}
	info = &e
	return
}

func isSuccess(code int) bool {
	return code == http.StatusOK ||
		code == http.StatusCreated ||
		code == http.StatusAccepted ||
		code == http.StatusNoContent
}
*/
