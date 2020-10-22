package main

import (
	"encoding/json"
	"fmt"
	"github.com/BurntSushi/toml"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"time"
)

type Config struct {
	Username        string `toml:"username"`
	Password        string `toml:"password"`
	JiraURL         string `toml:"url"`
	NumberOfPeriods string `toml:"numberOfPeriods"`
}

var re = regexp.MustCompile(`(?m)name="ajs-tempo-user-key" content="(\w+)"`)

func main() {
	var config Config
	_, err := toml.DecodeFile("config.toml", &config)
	if err != nil {
		log.Fatal(err)
	}

	jar, _ := cookiejar.New(nil)
	c := http.Client{Jar: jar}

	log.Println("Logging in")
	resp, err := c.PostForm(config.JiraURL+"/login.jsp", url.Values{
		"os_username":    {config.Username},
		"os_password":    {config.Password},
		"os_destination": {"/secure/Tempo.jspa"},
		"user_role":      {},
		"atl_token":      {},
		"login":          {"Anmelden"},
	})
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	var userID string
	for _, match := range re.FindAllStringSubmatch(string(data), -1) {
		log.Println("Found UserID:", match[1])
		userID = match[1]
		break
	}

	log.Println("Fetching Timesheets")
	resp, err = c.Get(config.JiraURL + "/rest/tempo-timesheets/4/timesheet-approval/approval-statuses?" + url.Values{"userKey": {userID}, "numberOfPeriods": {config.NumberOfPeriods}}.Encode())
	if err != nil {
		log.Fatal(err)
	}

	var tts []TempoTimeSheet
	if err := json.NewDecoder(resp.Body).Decode(&tts); err != nil {
		log.Fatal(err)
	}

	var budget int
	for _, v := range tts {
		if v.WorkedSeconds == 0 {
			log.Println("Skipping", v.SmartDateString)
			continue
		}

		budget += v.WorkedSeconds
		budget -= v.RequiredSecondsRelativeToday
	}

	duration, err := time.ParseDuration(fmt.Sprintf("%ds", budget))
	if err != nil {
		log.Fatal(err)
	}

	log.Println(duration.String())
}

type TempoTimeSheet struct {
	User                         User     `json:"user"`
	Status                       string   `json:"status"`
	WorkedSeconds                int      `json:"workedSeconds"`
	SubmittedSeconds             int      `json:"submittedSeconds"`
	RequiredSeconds              int      `json:"requiredSeconds"`
	RequiredSecondsRelativeToday int      `json:"requiredSecondsRelativeToday"`
	Period                       Period   `json:"period"`
	SmartDateString              string   `json:"smartDateString"`
	Worklogs                     Worklogs `json:"worklogs"`
	Action                       Action   `json:"action,omitempty"`
}

type User struct {
	Self        string `json:"self"`
	Name        string `json:"name"`
	Key         string `json:"key"`
	DisplayName string `json:"displayName"`
	Avatar      string `json:"avatar"`
}
type Period struct {
	PeriodView string `json:"periodView"`
	DateFrom   string `json:"dateFrom"`
	DateTo     string `json:"dateTo"`
}
type Worklogs struct {
	Href string `json:"href"`
}
type Reviewer struct {
	Self        string `json:"self"`
	Name        string `json:"name"`
	Key         string `json:"key"`
	DisplayName string `json:"displayName"`
	Avatar      string `json:"avatar"`
}
type Actor struct {
	Self        string `json:"self"`
	Name        string `json:"name"`
	Key         string `json:"key"`
	DisplayName string `json:"displayName"`
	Avatar      string `json:"avatar"`
}
type Action struct {
	Name     string   `json:"name"`
	Comment  string   `json:"comment"`
	Reviewer Reviewer `json:"reviewer"`
	Actor    Actor    `json:"actor"`
	Created  string   `json:"created"`
}
