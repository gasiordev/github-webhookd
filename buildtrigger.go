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

func (trig *BuildTrigger) Init(cfgFile string) {
	cfgJSON, err := ioutil.ReadFile(cfgFile)
	if err != nil {
		log.Fatal("Error reading config file")
	}

	var cfg Config
	cfg.SetFromJSON(cfgJSON)
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

func (trig *BuildTrigger) ProcessGitHubPayload(b *([]byte)) error {
	j := make(map[string]interface{})
	err := json.Unmarshal(*b, &j)
	if err == nil {
		log.Print("Got payload from GitHub")
		if j["pusher"] != nil && j["ref"] != nil && j["repository"] != nil && j["repository"].(map[string]interface{})["name"] != nil {
			ref := strings.Split(j["ref"].(string), "/")
			if ref[1] != "tag" {
				log.Print("Got payload to process")
				return trig.triggerMultibranchJob(j["repository"].(map[string]interface{})["name"].(string), ref[2])
			}
		}
	} else {
		return errors.New("Got non-JSON payload")
	}
	return nil
}

func (trig *BuildTrigger) triggerMultibranchJob(repo string, branch string) error {
	err := trig.triggerMultibranchJobScan(repo)
	if err != nil {
		return errors.New("Error sending POST to multibranch job scan of " + repo)
	}

	crumb, _ := trig.getCrumb(trig.config.JenkinsUser, trig.config.JenkinsToken)

	iterations := int(0)
	for iterations < 20 {
		req, err := http.NewRequest("POST", trig.config.JenkinsURL+"/job/"+repo+"_multibranch/job/"+branch+"/build", strings.NewReader(""))
		if err != nil {
			log.Print("Error creating request for building multibranch job")
			time.Sleep(time.Second * time.Duration(5))
			iterations++
			continue
		}

		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		req.SetBasicAuth(trig.config.JenkinsUser, trig.config.JenkinsToken)
		req.Header.Add("Jenkins-Crumb", crumb)
		c := &http.Client{}
		resp, err := c.Do(req)
		defer resp.Body.Close()

		if err == nil && resp.StatusCode == 201 {
			log.Print("Found and triggered branch " + branch + " in " + repo)
			return nil
		}

		log.Print("Couldn't find branch " + branch + " in " + repo + " (" + strconv.Itoa(iterations+1) + "/20)")
		time.Sleep(time.Second * time.Duration(5))

		iterations++
	}

	return errors.New("Timeout waiting for branch existance")
}

func (trig *BuildTrigger) getCrumb(user string, token string) (string, error) {
	req, err := http.NewRequest("GET", trig.config.JenkinsURL+"/crumbIssuer/api/xml?xpath=concat(//crumbRequestField,\":\",//crumb)", strings.NewReader(""))
	if err != nil {
		return "", err
	}

	req.SetBasicAuth(trig.config.JenkinsUser, trig.config.JenkinsToken)
	c := &http.Client{}
	resp, err := c.Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()
	b, _ := ioutil.ReadAll(resp.Body)
	return strings.Split(string(b), ":")[1], nil
}

func (trig *BuildTrigger) triggerMultibranchJobScan(repo string) error {
	crumb, _ := trig.getCrumb(trig.config.JenkinsUser, trig.config.JenkinsToken)
	req, err := http.NewRequest("POST", trig.config.JenkinsURL+"/job/"+repo+"_multibranch/build", strings.NewReader(""))
	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(trig.config.JenkinsUser, trig.config.JenkinsToken)
	req.Header.Add("Jenkins-Crumb", crumb)
	c := &http.Client{}
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()

	log.Print("Triggered multibranch job scan for " + repo)

	return nil
}
