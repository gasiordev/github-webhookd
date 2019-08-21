package main

import (
	"github.com/gorilla/mux"
	"io/ioutil"
	"net/http"
)

func getAPIGitHubWebhookPostHandler(app *App) http.HandlerFunc {
	fn := func(w http.ResponseWriter, r *http.Request) {
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		event := r.Header.Get("X-GitHub-Event")

		signature := r.Header.Get("X-Hub-Signature")
		config := app.GetConfig()
		if config.Secret != "" {
			if !app.VerifySignature([]byte(config.Secret), signature, &b) {
				http.Error(w, "Signature verification failed", 401)
				return
			}
		}

		if event != "ping" {
			err = app.ProcessGitHubPayload(&b, event)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}

			err = app.ForwardGitHubPayload(&b, r.Header)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
		}

		w.WriteHeader(http.StatusOK)
		w.Header().Set("content-type", "application/json")
	}
	return http.HandlerFunc(fn)
}

func NewTriggerAPIRouter(app *App) *(mux.Router) {
	router := mux.NewRouter()
	router.HandleFunc("/", getAPIGitHubWebhookPostHandler(app)).Methods("POST")
	return router
}
