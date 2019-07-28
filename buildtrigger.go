package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
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

func (trig *BuildTrigger) getRepository(j map[string]interface{}, event string) string {
	if event == "push" || event == "create" || event == "delete" {
		if j["repository"] != nil {
			if j["repository"].(map[string]interface{})["name"] != nil {
				return j["repository"].(map[string]interface{})["name"].(string)
			}
		}
	} else if event == "pull_request" {
		if j["pull_request"] != nil {
			if j["pull_request"].(map[string]interface{})["head"] != nil {
				if j["pull_request"].(map[string]interface{})["head"].(map[string]interface{})["repo"] != nil {
					if j["pull_request"].(map[string]interface{})["head"].(map[string]interface{})["repo"].(map[string]interface{})["name"] != nil {
						return j["pull_request"].(map[string]interface{})["head"].(map[string]interface{})["repo"].(map[string]interface{})["name"].(string)
					}
				}
			}
		}
	}
	return ""
}
func (trig *BuildTrigger) getRef(j map[string]interface{}, event string) string {
	if j["ref"] != nil {
		return j["ref"].(string)
	} else {
		return ""
	}
}
func (trig *BuildTrigger) getRefType(j map[string]interface{}, event string) string {
	if j["ref_type"] != nil {
		return j["ref_type"].(string)
	} else {
		return ""
	}
}
func (trig *BuildTrigger) getBranch(j map[string]interface{}, event string) string {
	if event == "push" {
		ref := strings.Split(j["ref"].(string), "/")
		if ref[1] == "tag" {
			return ""
		}
		branch := ref[2]
		return branch
	}
	if event == "create" || event == "delete" {
		ref := j["ref"].(string)
		refType := j["ref_type"].(string)
		if refType != "branch" {
			return ""
		} else {
			return ref
		}
	}
	return ""
}
func (trig *BuildTrigger) getAction(j map[string]interface{}, event string) string {
	if event == "pull_request" {
		if j["action"] != nil {
			return j["action"].(string)
		}
	}
	return ""
}

func (trig *BuildTrigger) checkEventRepositories(repos *([]EndpointConditionRepository), repo string, branch string) bool {
	for _, r := range *repos {
		if r.Name == repo || r.Name == "*" {
			if r.Branches == nil || len(*(r.Branches)) == 0 {
				log.Print("Found " + r.Name + " repo")
				return true
			} else {
				for _, b := range *(r.Branches) {
					if b == branch {
						log.Print("Found " + b + " branch in " + r.Name + " repo")
						return true
					}
				}
			}
		}
	}
	return false
}
func (trig *BuildTrigger) checkEventBranches(branches *([]EndpointConditionBranch), branch string, repo string) bool {
	for _, b := range *branches {
		if b.Name == branch || b.Name == "*" {
			if b.Repositories == nil || len(*(b.Repositories)) == 0 {
				log.Print("Found " + b.Name + " branch")
				return true
			} else {
				for _, r := range *(b.Repositories) {
					if r == repo {
						log.Print("Found " + r + " repository in " + b.Name + " branch")
						return true
					}
				}
			}
		}
	}
	return false
}
func (trig *BuildTrigger) checkEventActions(actions *([]string), action string) bool {
	for _, a := range *actions {
		if a == action || a == "*" {
			return true
		}
	}
	return false
}

func (trig *BuildTrigger) checkEndpointEvent(t *JenkinsTrigger, j map[string]interface{}, event string) error {
	repo := trig.getRepository(j, event)
	branch := trig.getBranch(j, event)

	action := ""
	if t.Events.PullRequest != nil && event == "pull_request" {
		action = trig.getAction(j, event)
		if action == "" {
			return errors.New("action is empty")
		}
		inActions := trig.checkEventActions(t.Events.PullRequest.Actions, action)
		if !inActions {
			return errors.New("Event " + event + "not supported")
		}
	}

	var c *EndpointConditions
	if event == "push" && t.Events.Push != nil {
		c = t.Events.Push
	} else if event == "pull_request" && t.Events.PullRequest != nil {
		c = t.Events.PullRequest
	} else if event == "create" && t.Events.Create != nil {
		c = t.Events.Create
	} else if event == "delete" && t.Events.Delete != nil {
		c = t.Events.Delete
	} else {
		return errors.New("Event " + event + "not supported")
	}

	inRepos := false
	if c.Repositories != nil {
		inRepos = trig.checkEventRepositories(c.Repositories, repo, branch)
	}
	inBranches := false
	if c.Branches != nil && event == "push" {
		inBranches = trig.checkEventBranches(c.Branches, branch, repo)
	}
	inExcludeRepos := false
	if c.ExcludeRepositories != nil {
		inExcludeRepos = trig.checkEventRepositories(c.ExcludeRepositories, repo, branch)
	}
	inExcludeBranches := false
	if c.ExcludeBranches != nil && event == "push" {
		inExcludeBranches = trig.checkEventBranches(c.ExcludeBranches, branch, repo)
	}
	if (inRepos || inBranches) && !inExcludeRepos && !inExcludeBranches {
		return nil
	}

	return errors.New("Event " + event + "not supported")
}

func (trig *BuildTrigger) processJenkinsEndpoint(t *JenkinsTrigger, j map[string]interface{}, event string) error {
	repo := trig.getRepository(j, event)
	ref := trig.getRef(j, event)
	if repo == "" {
		return nil
	}

	if event == "push" {
		if ref == "" {
			return nil
		}
	}

	branch := trig.getBranch(j, event)
	if event == "push" {
		if branch == "" {
			return nil
		}
	}

	endp := trig.config.Jenkins.EndpointsMap[t.Endpoint]
	if endp == nil {
		return nil
	}

	err := trig.checkEndpointEvent(t, j, event)
	if err != nil {
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

func (trig *BuildTrigger) ProcessGitHubPayload(b *([]byte), event string) error {
	j := make(map[string]interface{})
	err := json.Unmarshal(*b, &j)
	if err != nil {
		return errors.New("Got non-JSON payload")
	}

	if trig.config.Triggers.Jenkins != nil {
		for _, t := range trig.config.Triggers.Jenkins {
			err := trig.processJenkinsEndpoint(&t, j, event)
			if err != nil {
				log.Print("Error processing endpoint " + t.Endpoint + ". Breaking.")
				break
			}
		}
	}
	return nil
}

func (trig *BuildTrigger) ForwardGitHubPayload(b *([]byte), h http.Header) error {
	githubHeaders := []string{"X-GitHub-Event", "X-Hub-Signature", "X-GitHub-Delivery", "content-type"}
	if trig.config.Forward != nil {
		for _, f := range *(trig.config.Forward) {
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
			}
		}
	}
	return nil
}

func (trig *BuildTrigger) signBody(secret []byte, body []byte) []byte {
	computed := hmac.New(sha1.New, secret)
	computed.Write(body)
	return []byte(computed.Sum(nil))
}

func (trig *BuildTrigger) VerifySignature(secret []byte, signature string, body *([]byte)) bool {
	actual := make([]byte, 20)
	hex.Decode(actual, []byte(signature[5:]))
	return hmac.Equal(trig.signBody(secret, *body), actual)
}
