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

		err = trig.ProcessGitHubPayload(&b, event)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Header().Set("content-type", "application/json")
	}
	return http.HandlerFunc(fn)
}

func NewTriggerAPIRouter(trig *BuildTrigger) *(mux.Router) {
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/github_webhook/", getAPIGitHubWebhookPostHandler(trig)).Methods("POST")
	return router
}
