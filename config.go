package main

import (
	"encoding/json"
	"log"
)

type Config struct {
	Version  string  `json:"version"`
	Port     string  `json:"port"`
	Jenkins  Jenkins `json:"jenkins"`
	Triggers Trigger `json:"triggers"`
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

type JenkinsEndpointRetry struct {
	Delay string `json:"delay"`
	Count string `json:"count"`
}

type JenkinsEndpointSuccess struct {
	HTTPStatus string `json:"http_status"`
}

type Trigger struct {
	Jenkins []JenkinsTrigger `json:"jenkins"`
}

type JenkinsTrigger struct {
	Endpoint string `json:"endpoint"`
	Events   Events `json:"events"`
}

type Events struct {
	Push *EndpointConditions `json:"push"`
}

type EndpointConditions struct {
	Repositories *([]EndpointConditionRepository) `json:"repositories"`
	Branches     *([]EndpointConditionBranch)     `json:"branches"`
}

type EndpointConditionRepository struct {
	Name     string   `json:"name"`
	Branches *([]string) `json:"branches"`
}

type EndpointConditionBranch struct {
	Name         string   `json:"name"`
	Repositories *([]string) `json:"repositories"`
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
