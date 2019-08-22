package main

import (
	"encoding/json"
	"errors"
	"log"
	"strconv"
)

type Config struct {
	Version  string       `json:"version"`
	Port     string       `json:"port"`
	Jenkins  Jenkins      `json:"jenkins"`
	Triggers Trigger      `json:"triggers"`
	Forward  *([]Forward) `json:"forward"`
	Secret   string       `json:"secret";omitempty`
}

func (c *Config) SetFromJSON(b []byte) {
	err := json.Unmarshal(b, c)
	if err != nil {
		log.Fatal("Error setting config from JSON:", err.Error())
	}
	c.Jenkins.EndpointsMap = make(map[string]*JenkinsEndpoint)
	for i, e := range c.Jenkins.Endpoints {
		c.Jenkins.EndpointsMap[e.Id] = &(c.Jenkins.Endpoints[i])
	}
}

type Jenkins struct {
	User         string            `json:"user"`
	Token        string            `json:"token"`
	BaseURL      string            `json:"base_url"`
	Endpoints    []JenkinsEndpoint `json:"endpoints"`
	EndpointsMap map[string]*JenkinsEndpoint
}

type JenkinsEndpoint struct {
	Id        string                 `json:"id"`
	Path      string                 `json:"path"`
	Retry     JenkinsEndpointRetry   `json:"retry"`
	Success   JenkinsEndpointSuccess `json:"success"`
	Condition string                 `json:"condition"`
}

func (endpoint *JenkinsEndpoint) GetRetryCount() (int, error) {
	rc := int(1)
	if endpoint.Retry.Count != "" {
		i, err := strconv.Atoi(endpoint.Retry.Count)
		if err != nil {
			return 0, errors.New("Value of Retry.Count cannot be converted to int")
		}
		rc = i
	}
	return rc, nil
}

func (endpoint *JenkinsEndpoint) GetRetryDelay() (int, error) {
	rd := int(0)
	if endpoint.Retry.Delay != "" {
		i, err := strconv.Atoi(endpoint.Retry.Count)
		if err != nil {
			return 0, errors.New("Value of Retry.Delay cannot be converted to int")
		}
		rd = i
	}
	return rd, nil
}

func (endpoint *JenkinsEndpoint) CheckHTTPStatus(statusCode int) bool {
	expected, err := strconv.Atoi(endpoint.Success.HTTPStatus)
	if err != nil {
		return false
	}
	if statusCode != expected {
		return false
	}
	return true
}

type JenkinsEndpointRetry struct {
	Delay string `json:"delay"`
	Count string `json:"count"`
}

type JenkinsEndpointSuccess struct {
	HTTPStatus string `json:"http_status"`
}

type Forward struct {
	URL     string `json:"url"`
	Headers bool   `json:"headers";omitempty`
}

type Trigger struct {
	Jenkins []JenkinsTrigger `json:"jenkins"`
}

type JenkinsTrigger struct {
	Endpoint string `json:"endpoint"`
	Events   Events `json:"events"`
}

func (jenkinstrigger *JenkinsTrigger) CheckEvent(repo string, branch string, action string, event string) error {
	if jenkinstrigger.Events.PullRequest != nil && event == "pull_request" {
		if action == "" {
			return errors.New("action is empty")
		}
		inActions := jenkinstrigger.Events.PullRequest.CheckActions(action)
		if !inActions {
			return errors.New("Event " + event + "not supported")
		}
	}

	var c *EndpointConditions
	if event == "push" && jenkinstrigger.Events.Push != nil {
		c = jenkinstrigger.Events.Push
	} else if event == "pull_request" && jenkinstrigger.Events.PullRequest != nil {
		c = jenkinstrigger.Events.PullRequest
	} else if event == "create" && jenkinstrigger.Events.Create != nil {
		c = jenkinstrigger.Events.Create
	} else if event == "delete" && jenkinstrigger.Events.Delete != nil {
		c = jenkinstrigger.Events.Delete
	} else {
		return errors.New("Event " + event + "not supported")
	}

	inRepos := false
	if c.Repositories != nil {
		inRepos = c.CheckRepositories(repo, branch, false)
	}
	inBranches := false
	if c.Branches != nil && event == "push" {
		inBranches = c.CheckBranches(branch, repo, false)
	}
	inExcludeRepos := false
	if c.ExcludeRepositories != nil {
		inExcludeRepos = c.CheckRepositories(repo, branch, true)
	}
	inExcludeBranches := false
	if c.ExcludeBranches != nil && event == "push" {
		inExcludeBranches = c.CheckBranches(branch, repo, true)
	}
	if (inRepos || inBranches) && !inExcludeRepos && !inExcludeBranches {
		return nil
	}

	return errors.New("Event " + event + "not supported")
}

type Events struct {
	Push        *EndpointConditions `json:"push";omitempty`
	PullRequest *EndpointConditions `json:"pull_request";omitempty`
	Create      *EndpointConditions `json:"create";omitempty`
	Delete      *EndpointConditions `json:"delete";omitempty`
}

type EndpointConditions struct {
	Repositories        *([]EndpointConditionRepository) `json:"repositories";omitempty`
	Branches            *([]EndpointConditionBranch)     `json:"branches";omitempty`
	ExcludeRepositories *([]EndpointConditionRepository) `json:"exclude_repositories";omitempty`
	ExcludeBranches     *([]EndpointConditionBranch)     `json:"exclude_branches";omitempty`
	Actions             *([]string)                      `json:"actions";omitempty`
}

func (cond *EndpointConditions) CheckActions(action string) bool {
	for _, a := range *cond.Actions {
		if a == action || a == "*" {
			return true
		}
	}
	return false
}

func (cond *EndpointConditions) CheckRepositories(repo string, branch string, exclude bool) bool {
	repos := cond.Repositories
	if exclude {
		repos = cond.ExcludeRepositories
	}
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
func (cond *EndpointConditions) CheckBranches(branch string, repo string, exclude bool) bool {
	branches := cond.Branches
	if exclude {
		branches = cond.ExcludeBranches
	}
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

type EndpointConditionRepository struct {
	Name     string      `json:"name"`
	Branches *([]string) `json:"branches";omitempty`
}

type EndpointConditionBranch struct {
	Name         string      `json:"name"`
	Repositories *([]string) `json:"repositories";omitempty`
}
