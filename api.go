package main

import (
	"github.com/gorilla/mux"
	"io/ioutil"
	"log"
	"net/http"
)

type API struct {
	router *mux.Router
	app    *App
}

func NewAPI() *API {
	api := &API{}
	return api
}

func (api *API) Init(app *App) {
	api.app = app
	api.router = mux.NewRouter()
	api.router.HandleFunc("/", api.handler).Methods("POST")
}

func (api *API) Run(app *App) {
	api.Init(app)

	config := api.app.GetConfig()
	log.Print("Starting daemon listening on " + config.Port + "...")
	log.Fatal(http.ListenAndServe(":"+config.Port, api.router))
}

func (api *API) handler(w http.ResponseWriter, r *http.Request) {
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	githubAPI := api.app.GetGitHubAPI()
	event := githubAPI.GetEvent(r)
	signature := githubAPI.GetSignature(r)
	config := api.app.GetConfig()
	if config.Secret != "" {
		if !githubAPI.VerifySignature([]byte(config.Secret), signature, &b) {
			http.Error(w, "Signature verification failed", 401)
			return
		}
	}

	if event != "ping" {
		err = api.app.ProcessGitHubPayload(&b, event)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		err = api.app.ForwardGitHubPayload(&b, r.Header)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("content-type", "application/json")
}
