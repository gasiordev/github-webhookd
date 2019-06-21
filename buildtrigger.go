package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type BuildTrigger struct {
	config Config
}

func NewBuildTrigger() *BuildTrigger {
	trig := &BuildTrigger{}
	return trig
}

func (trig *BuildTrigger) GetConfig() *Config {
	return &(trig.config)
}

func (trig *BuildTrigger) Init(p string) {
	c, err := ioutil.ReadFile(p)
	if err != nil {
		log.Fatal("Error reading config file")
	}

	var cfg Config
	cfg.SetFromJSON(c)
	trig.config = cfg
}

func (trig *BuildTrigger) Start() int {
	done := make(chan bool)
	go trig.startTriggerAPI()
	<-done
	return 0
}

func (trig *BuildTrigger) Run() {
	trigCLI := NewBuildTriggerCLI(trig)
	os.Exit(trigCLI.Run(os.Stdout, os.Stderr))
}

func (trig *BuildTrigger) startTriggerAPI() {
	router := NewTriggerAPIRouter(trig)
	log.Print("Starting daemon listening on " + trig.config.Port + "...")
	log.Fatal(http.ListenAndServe(":"+trig.config.Port, router))
}

func (trig *BuildTrigger) getJenkinsEndpointRetryCount(e *JenkinsEndpoint) (int, error) {
	rc := int(1)
	if e.Retry.Count != "" {
		i, err := strconv.Atoi(e.Retry.Count)
		if err != nil {
			return 0, errors.New("Value of Retry.Count cannot be converted to int")
		}
		rc = i
	}
	return rc, nil
}

func (trig *BuildTrigger) getJenkinsEndpointRetryDelay(e *JenkinsEndpoint) (int, error) {
	rd := int(0)
	if e.Retry.Delay != "" {
		i, err := strconv.Atoi(e.Retry.Count)
		if err != nil {
			return 0, errors.New("Value of Retry.Delay cannot be converted to int")
		}
		rd = i
	}
	return rd, nil
}

func (trig *BuildTrigger) processJenkinsEndpoint(e string, repo string, branch string) error {
	endp := trig.config.Jenkins.EndpointsMap[e]
	if endp == nil {
		return nil
	}

	rd, err := trig.getJenkinsEndpointRetryDelay(endp)
	if err != nil {
		return nil
	}
	rc, err := trig.getJenkinsEndpointRetryCount(endp)
	if err != nil {
		return nil
	}

	return trig.processJenkinsEndpointRetries(endp, repo, branch, rd, rc)
}

func (trig *BuildTrigger) printIteration(i int, rc int) {
	log.Print("Retry: (" + strconv.Itoa(i+1) + "/" + strconv.Itoa(rc) + ")")
}

func (trig *BuildTrigger) getCrumbAndSleep(u string, t string, rd int) (string, error) {
	crumb, err := trig.getCrumb(u, t)
	if err != nil {
		log.Print("Error getting crumb")
		time.Sleep(time.Second * time.Duration(rd))
		return "", errors.New("Error getting crumb")
	}
	return crumb, nil
}

func (trig *BuildTrigger) replacePathWithRepoAndBranch(p string, r string, b string) string {
	s := strings.ReplaceAll(p, "{{.repository}}", r)
	s = strings.ReplaceAll(s, "{{.branch}}", b)
	return s
}

func (trig *BuildTrigger) compareHTTPStatusAndSleep(rsCode int, es string, rd int) (bool, error) {
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

func (trig *BuildTrigger) processJenkinsEndpointRetries(endpointDef *JenkinsEndpoint, repo string, branch string, retryDelay int, retryCount int) error {
	iterations := int(0)
	if retryCount > 0 {
		for iterations < retryCount {
			trig.printIteration(iterations, retryCount)

			crumb, err := trig.getCrumbAndSleep(trig.config.Jenkins.User, trig.config.Jenkins.Token, retryDelay)
			if err != nil {
				iterations++
				continue
			}

			endpointPath := trig.replacePathWithRepoAndBranch(endpointDef.Path, repo, branch)

			req, err := http.NewRequest("POST", trig.config.Jenkins.BaseURL+"/"+endpointPath, strings.NewReader(""))
			if err != nil {
				log.Print("Error creating request to " + endpointPath)
				time.Sleep(time.Second * time.Duration(retryDelay))
				iterations++
				continue
			}

			req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
			req.SetBasicAuth(trig.config.Jenkins.User, trig.config.Jenkins.Token)
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
				cmp, err := trig.compareHTTPStatusAndSleep(resp.StatusCode, endpointDef.Success.HTTPStatus, retryDelay)
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

func (trig *BuildTrigger) ProcessGitHubPayload(b *([]byte)) error {
	j := make(map[string]interface{})
	err := json.Unmarshal(*b, &j)
	if err == nil {
		if j["pusher"] != nil && j["ref"] != nil && j["repository"] != nil && j["repository"].(map[string]interface{})["name"] != nil {
			ref := strings.Split(j["ref"].(string), "/")
			if ref[1] != "tag" {
				log.Print("Got payload from GitHub to process")

				if trig.config.EndpointsToTrigger.Jenkins != nil {
					for _, e := range trig.config.EndpointsToTrigger.Jenkins {
						err := trig.processJenkinsEndpoint(e, j["repository"].(map[string]interface{})["name"].(string), ref[2])
						if err != nil {
							log.Print("Error processing endpoint " + e + ". Breaking.")
							break
						}
					}
				}

				return nil
			}
		}
	} else {
		return errors.New("Got non-JSON payload")
	}
	return nil
}

func (trig *BuildTrigger) getCrumb(user string, token string) (string, error) {
	req, err := http.NewRequest("GET", trig.config.Jenkins.BaseURL+"/crumbIssuer/api/xml?xpath=concat(//crumbRequestField,\":\",//crumb)", strings.NewReader(""))
	if err != nil {
		return "", err
	}

	req.SetBasicAuth(trig.config.Jenkins.User, trig.config.Jenkins.Token)
	c := &http.Client{}
	resp, err := c.Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()
	b, _ := ioutil.ReadAll(resp.Body)
	return strings.Split(string(b), ":")[1], nil
}
