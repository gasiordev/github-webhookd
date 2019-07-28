package main

import (
	"github.com/gorilla/mux"
	"io/ioutil"
	"net/http"
)

func getAPIGitHubWebhookPostHandler(trig *BuildTrigger) http.HandlerFunc {
	fn := func(w http.ResponseWriter, r *http.Request) {
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		event := r.Header.Get("X-GitHub-Event")

		signature := r.Header.Get("X-Hub-Signature")
		config := trig.GetConfig()
		if config.Secret != "" {
			if !trig.VerifySignature([]byte(config.Secret), signature, &b) {
				http.Error(w, "Signature verification failed", 401)
			}
		}

		if event != "ping" {
			err = trig.ProcessGitHubPayload(&b, event)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}

			err = trig.ForwardGitHubPayload(&b, r.Header)
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

func NewTriggerAPIRouter(trig *BuildTrigger) *(mux.Router) {
	router := mux.NewRouter()
	router.HandleFunc("/", getAPIGitHubWebhookPostHandler(trig)).Methods("POST")
	return router
}
