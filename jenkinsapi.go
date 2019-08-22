package main

import (
	"io/ioutil"
	"net/http"
	"strings"
)

type JenkinsAPI struct {
}

func NewJenkinsAPI() *JenkinsAPI {
	jenkinsapi := &JenkinsAPI{}
	return jenkinsapi
}

func (jenkinsapi *JenkinsAPI) GetCrumb(baseURL string, user string, token string) (string, error) {
	req, err := http.NewRequest("GET", baseURL+"/crumbIssuer/api/xml?xpath=concat(//crumbRequestField,\":\",//crumb)", strings.NewReader(""))
	if err != nil {
		return "", err
	}

	req.SetBasicAuth(user, token)
	c := &http.Client{}
	resp, err := c.Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()
	b, _ := ioutil.ReadAll(resp.Body)
	return strings.Split(string(b), ":")[1], nil
}
