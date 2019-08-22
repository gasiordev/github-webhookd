package main

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"net/http"
	"strings"
)

type GitHubAPI struct {
}

func NewGitHubAPI() *GitHubAPI {
	githubAPI := &GitHubAPI{}
	return githubAPI
}

func (githubAPI *GitHubAPI) GetEvent(r *http.Request) string {
	return r.Header.Get("X-GitHubAPI-Event")
}

func (githubAPI *GitHubAPI) GetSignature(r *http.Request) string {
	return r.Header.Get("X-Hub-Signature")
}

func (githubAPI *GitHubAPI) signBody(secret []byte, body []byte) []byte {
	computed := hmac.New(sha1.New, secret)
	computed.Write(body)
	return []byte(computed.Sum(nil))
}

func (githubAPI *GitHubAPI) VerifySignature(secret []byte, signature string, body *([]byte)) bool {
	actual := make([]byte, 20)
	hex.Decode(actual, []byte(signature[5:]))
	return hmac.Equal(githubAPI.signBody(secret, *body), actual)
}

func (githubAPI *GitHubAPI) GetRef(j map[string]interface{}, event string) string {
	if j["ref"] != nil {
		return j["ref"].(string)
	} else {
		return ""
	}
}
func (githubAPI *GitHubAPI) GetRefType(j map[string]interface{}, event string) string {
	if j["ref_type"] != nil {
		return j["ref_type"].(string)
	} else {
		return ""
	}
}
func (githubAPI *GitHubAPI) GetBranch(j map[string]interface{}, event string) string {
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
func (githubAPI *GitHubAPI) GetAction(j map[string]interface{}, event string) string {
	if event == "pull_request" {
		if j["action"] != nil {
			return j["action"].(string)
		}
	}
	return ""
}
func (githubAPI *GitHubAPI) GetRepository(j map[string]interface{}, event string) string {
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
