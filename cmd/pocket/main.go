package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"text/template"

	"github.com/docopt/docopt-go"
	"github.com/motemen/go-pocket/api"
	"github.com/motemen/go-pocket/auth"
)

var version = "0.1"

var consumerKey string

var defaultItemTemplate = template.Must(template.New("item").Parse(
	`[{{.ItemId | printf "%9d"}}] {{.ResolvedTitle}} <{{.ResolvedURL}}>`,
))

func main() {
	usage := `A Pocket <getpocket.com> client.

Usage:
  pocket list [--format=<Go template>] [--domain=<domain>] [--search=<query>]
`

	arguments, err := docopt.Parse(usage, nil, true, version, false)
	if err != nil {
		panic(err)
	}

	accessToken, err := restoreAccessToken()
	if err != nil {
		panic(err)
	}

	client := api.NewClient(consumerKey, accessToken.AccessToken)

	if doList, ok := arguments["list"].(bool); ok && doList {
		commandList(arguments, client)
	}
}

func commandList(arguments map[string]interface{}, client *api.Client) {
	options := &api.RetrieveAPIOption{}

	if domain, ok := arguments["--domain"].(string); ok {
		options.Domain = domain
	}

	if search, ok := arguments["--search"].(string); ok {
		options.Search = search
	}

	res, err := client.Retrieve(options)
	if err != nil {
		panic(err)
	}

	var itemTemplate *template.Template
	if format, ok := arguments["--format"].(string); ok {
		itemTemplate = template.Must(template.New("item").Parse(format))
	} else {
		itemTemplate = defaultItemTemplate
	}
	for _, item := range res.List {
		_ = itemTemplate.Execute(os.Stdout, item)
		fmt.Println("")
	}
}

func restoreAccessToken() (*auth.OAuthAuthorizeAPIResponse, error) {
	accessToken := &auth.OAuthAuthorizeAPIResponse{}

	err := loadJSONFromFile(".auth_cache.json", accessToken)

	if err != nil {
		log.Println(err)

		accessToken, err = obtainAccessToken()
		if err != nil {
			return nil, err
		}

		err = saveJSONToFile(".auth_cache.json", accessToken)
		if err != nil {
			return nil, err
		}
	}

	return accessToken, nil
}

func obtainAccessToken() (*auth.OAuthAuthorizeAPIResponse, error) {
	ch := make(chan struct{})
	ts := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if req.URL.Path == "/favicon.ico" {
				http.Error(w, "Not Found", 404)
				return
			}

			w.Header().Set("Content-Type", "text/plain")
			fmt.Fprintln(w, "Authorized.")
			ch <- struct{}{}
		}))
	defer ts.Close()

	redirectURL := ts.URL

	requestToken, err := auth.ObtainRequestToken(consumerKey, redirectURL)
	if err != nil {
		return nil, err
	}

	url := auth.GenerateAuthorizationURL(requestToken, redirectURL)
	fmt.Println(url)

	<-ch

	return auth.ObtainAccessToken(consumerKey, requestToken)
}

func saveJSONToFile(path string, v interface{}) error {
	w, err := os.Create(path)
	if err != nil {
		return err
	}

	defer w.Close()

	return json.NewEncoder(w).Encode(v)
}

func loadJSONFromFile(path string, v interface{}) error {
	r, err := os.Open(path)
	if err != nil {
		return err
	}

	defer r.Close()

	return json.NewDecoder(r).Decode(v)
}
