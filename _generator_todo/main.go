package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"runtime"

	"willnorris.com/go/microformats"
)

type dataInfo struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

func generateJf2(data *microformats.Data, url, base string) error {
	/*if len(data.Items) != 1 {
		return errors.New("only one thing allowed")
	}

	var j interface{}
	jf2.ConvertItem(&j, data.Items[0])

	resJSON, err := json.Marshal(j)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filepath.Join(base, "index.jf2"), resJSON, 0644) */
	return ioutil.WriteFile(filepath.Join(base, "index.jf2"), []byte("{}"), 0644)
}

func process(path string, info os.FileInfo) error {
	base := filepath.Dir(path)

	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	var data *dataInfo
	err = json.Unmarshal(bytes, &data)
	if err != nil {
		return err
	}

	url, err := url.Parse(data.URL)
	if err != nil {
		return err
	}

	fd, err := os.Open(filepath.Join(base, "index.html"))
	if err != nil {
		return err
	}
	defer fd.Close()

	res := microformats.Parse(fd, url)

	resJSON, err := json.Marshal(res)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filepath.Join(base, "index.mf2"), resJSON, 0644)
	if err != nil {
		return err
	}

	if data.Type == "list" {
		return os.RemoveAll(path)
	} else {

		err = generateAs2(res, data.URL, base)
		if err != nil {
			return err
		}

		return generateJf2(res, data.URL, base)
	}
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	filepath.Walk("../../hacdias.com/public", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Println(err)
		}
		name := info.Name()
		switch name {
		case "mentions.json":
			return os.RemoveAll(path)
		case "index.as2":
			return process(path, info)
		}
		return nil
	})
}
