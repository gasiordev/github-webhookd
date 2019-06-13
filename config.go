package main

import (
	"encoding/json"
	"log"
)

type Config struct {
	Version      string `json:"version"`
	Port         string `json:"port"`
	JenkinsURL   string `json:"jenkins_url"`
	JenkinsUser  string `json:"jenkins_user"`
	JenkinsToken string `json:"jenkins_token"`
}

func (c *Config) SetFromJSON(b []byte) {
	err := json.Unmarshal(b, c)
	if err != nil {
		log.Fatal("Error setting config from JSON:", err.Error())
	}
}
