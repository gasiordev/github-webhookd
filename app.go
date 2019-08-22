package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type App struct {
	config        Config
	githubPayload *GitHubPayload
	jenkinsAPI    *JenkinsAPI
}

func NewApp() *App {
	app := &App{}
	return app
}

func (app *App) GetConfig() *Config {
	return &(app.config)
}

func (app *App) GetGitHubPayload() *GitHubPayload {
	return app.githubPayload
}

func (app *App) GetJenkinsAPI() *JenkinsAPI {
	return app.jenkinsAPI
}

func (app *App) Init(p string) {
	c, err := ioutil.ReadFile(p)
	if err != nil {
		log.Fatal("Error reading config file")
	}

	var cfg Config
	cfg.SetFromJSON(c)
	app.config = cfg

	app.githubPayload = NewGitHubPayload()
	app.jenkinsAPI = NewJenkinsAPI()
}

func (app *App) Start() int {
	done := make(chan bool)
	go app.startAPI()
	<-done
	return 0
}

func (app *App) Run() {
	cli := NewCLI()
	cli.Run(app)
}

func (app *App) startAPI() {
	api := NewAPI()
	api.Run(app)
}

func (app *App) printIteration(i int, rc int) {
	log.Print("Retry: (" + strconv.Itoa(i+1) + "/" + strconv.Itoa(rc) + ")")
}

func (app *App) getCrumbAndSleep(u string, t string, rd int) (string, error) {
	crumb, err := app.jenkinsAPI.GetCrumb(app.config.Jenkins.BaseURL, u, t)
	if err != nil {
		log.Print("Error getting crumb")
		time.Sleep(time.Second * time.Duration(rd))
		return "", errors.New("Error getting crumb")
	}
	return crumb, nil
}

func (app *App) replacePathWithRepoAndBranch(p string, r string, b string) string {
	s := strings.ReplaceAll(p, "{{.repository}}", r)
	s = strings.ReplaceAll(s, "{{.branch}}", b)
	return s
}

func (app *App) compareHTTPStatusAndSleep(rsCode int, es string, rd int) (bool, error) {
	esCode, err := strconv.Atoi(es)
	if err != nil {
		return false, errors.New("Error converting Success.HTTPStatus to int")
	}
	if rsCode != esCode {
		rs := strconv.Itoa(rsCode)
		log.Print("HTTP Status " + rs + " different than expected " + es)
		time.Sleep(time.Second * time.Duration(rd))
		return false, nil
	}
	return true, nil
}

func (app *App) processJenkinsEndpointRetries(endpointDef *JenkinsEndpoint, repo string, branch string, retryDelay int, retryCount int) error {
	iterations := int(0)
	if retryCount > 0 {
		for iterations < retryCount {
			app.printIteration(iterations, retryCount)

			crumb, err := app.getCrumbAndSleep(app.config.Jenkins.User, app.config.Jenkins.Token, retryDelay)
			if err != nil {
				iterations++
				continue
			}

			endpointPath := app.replacePathWithRepoAndBranch(endpointDef.Path, repo, branch)

			req, err := http.NewRequest("POST", app.config.Jenkins.BaseURL+"/"+endpointPath, strings.NewReader(""))
			if err != nil {
				log.Print("Error creating request to " + endpointPath)
				time.Sleep(time.Second * time.Duration(retryDelay))
				iterations++
				continue
			}

			req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
			req.SetBasicAuth(app.config.Jenkins.User, app.config.Jenkins.Token)
			req.Header.Add("Jenkins-Crumb", crumb)
			c := &http.Client{}
			resp, err := c.Do(req)
			if err != nil {
				log.Print("Error from request to " + endpointPath)
				time.Sleep(time.Second * time.Duration(retryDelay))
				iterations++
				continue
			}
			resp.Body.Close()

			log.Print("Posted to endpoint " + endpointPath)

			if endpointDef.Success.HTTPStatus != "" {
				cmp, err := app.compareHTTPStatusAndSleep(resp.StatusCode, endpointDef.Success.HTTPStatus, retryDelay)
				if err != nil {
					return err
				}
				if !cmp {
					iterations++
					continue
				}
			}

			return nil
		}
	}
	return errors.New("Unable to post to endpoint " + endpointDef.Path)
}

func (app *App) processPayloadOnJenkinsTrigger(jenkinstrigger *JenkinsTrigger, j map[string]interface{}, event string) error {
	githubPayload := app.GetGitHubPayload()
	config := app.GetConfig()

	repo := githubPayload.GetRepository(j, event)
	ref := githubPayload.GetRef(j, event)
	branch := githubPayload.GetBranch(j, event)
	action := ""
	if jenkinstrigger.Events.PullRequest != nil && event == "pull_request" {
		action = githubPayload.GetAction(j, event)
	}
	if repo == "" {
		return nil
	}

	if event == "push" {
		if ref == "" {
			return nil
		}
	}

	if event == "push" {
		if branch == "" {
			return nil
		}
	}

	endp := config.Jenkins.EndpointsMap[jenkinstrigger.Endpoint]
	if endp == nil {
		return nil
	}

	err := jenkinstrigger.CheckEvent(repo, branch, action, event)
	if err != nil {
		return nil
	}

	rd, err := endp.GetRetryDelay()
	if err != nil {
		return nil
	}
	rc, err := endp.GetRetryCount()
	if err != nil {
		return nil
	}

	return app.processJenkinsEndpointRetries(endp, repo, branch, rd, rc)
}

func (app *App) ProcessGitHubPayload(b *([]byte), event string) error {
	j := make(map[string]interface{})
	err := json.Unmarshal(*b, &j)
	if err != nil {
		return errors.New("Got non-JSON payload")
	}

	if app.config.Triggers.Jenkins != nil {
		for _, t := range app.config.Triggers.Jenkins {
			err = app.processPayloadOnJenkinsTrigger(&t, j, event)
			if err != nil {
				log.Print("Error processing endpoint " + t.Endpoint + ". Breaking.")
				break
			}
		}
	}
	return nil
}

func (app *App) ForwardGitHubPayload(b *([]byte), h http.Header) error {
	githubHeaders := []string{"X-GitHubPayload-Event", "X-Hub-Signature", "X-GitHubPayload-Delivery", "content-type"}
	if app.config.Forward != nil {
		for _, f := range *(app.config.Forward) {
			if f.URL != "" {
				req, err := http.NewRequest("POST", f.URL, bytes.NewReader(*b))
				if f.Headers {
					for _, k := range githubHeaders {
						if h.Get(k) != "" {
							req.Header.Add(k, h.Get(k))
						}
					}
				}
				if err != nil {
					return err
				}
				c := &http.Client{}
				_, err = c.Do(req)
				if err != nil {
					return err
				}

				log.Print("Forwarded to endpoint " + f.URL)
			}
		}
	}
	return nil
}
