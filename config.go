package main

import (
	"encoding/json"
	"log"
)

type Config struct {
	Version            string             `json:"version"`
	Port               string             `json:"port"`
	Jenkins            Jenkins            `json:"jenkins"`
	EndpointsToTrigger EndpointsToTrigger `json:"endpoints_to_trigger"`
}

type Jenkins struct {
	User         string            `json:"user"`
	Token        string            `json:"token"`
	BaseURL      string            `json:"base_url"`
	Endpoints    []JenkinsEndpoint `json:"endpoints"`
	EndpointsMap map[string]*JenkinsEndpoint
}

type JenkinsEndpoint struct {
	Id      string                 `json:"id"`
	Path    string                 `json:"path"`
	Retry   JenkinsEndpointRetry   `json:"retry"`
	Success JenkinsEndpointSuccess `json:"success"`
}

type JenkinsEndpointRetry struct {
	Delay string `json:"delay"`
	Count string `json:"count"`
}

type JenkinsEndpointSuccess struct {
	HTTPStatus string `json:"http_status"`
}

type EndpointsToTrigger struct {
	Jenkins []string `json:"jenkins"`
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
