package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/nlopes/slack"
	"net/http"
	"os"
	"regexp"
	"sort"
	"time"
)

func main() {
	cmdGitHub, _ := regexp.Compile("^!gh\\s([a-zA-Z0-9_]+)")

	api := slack.New(
		os.Getenv("SLACK_TOKEN"),
		// slack.OptionDebug(true),
		// slack.OptionLog(log.New(os.Stdout, "slack-bot: ", log.Lshortfile|log.LstdFlags)),
	)

	rtm := api.NewRTM()
	go rtm.ManageConnection()

	for msg := range rtm.IncomingEvents {
		fmt.Print("Event Received\n")
		switch ev := msg.Data.(type) {
		case *slack.HelloEvent:
			// Ignore hello

		case *slack.ConnectedEvent:
			fmt.Printf("\tInfos: %+v\n", ev.Info)
			fmt.Printf("\tConnection counter: %d\n", ev.ConnectionCount)
			// Replace C2147483705 with your Channel ID
			// rtm.SendMessage(rtm.NewOutgoingMessage("Hello world", "C2147483705"))

		case *slack.MessageEvent:
			cmd := cmdGitHub.FindStringSubmatch(ev.Text)
			if len(cmd) != 2 {
				continue
			}
			user := cmd[1]

			rtm.SendMessage(rtm.NewOutgoingMessage(fmt.Sprintf("Fetching github repositories for %s...", user), ev.Channel))

			for _, m := range SplitSubN(getUserRepositories(user), 4000) {
				time.Sleep(time.Second)
				rtm.SendMessage(rtm.NewOutgoingMessage(m, ev.Channel))
			}

		case *slack.PresenceChangeEvent:
			fmt.Printf("\tPresence Change: %v\n", ev)

		case *slack.LatencyReport:
			fmt.Printf("\tCurrent latency: %v\n", ev.Value)

		case *slack.RTMError:
			fmt.Printf("\tError: %s\n", ev.Error())

		case *slack.InvalidAuthEvent:
			fmt.Printf("\tInvalid credentials")
			return

		default:

			// Ignore other events..
			fmt.Printf("\tUnexpected: %+v\n", msg.Data)
		}
	}
}

type UserRepository struct {
	FullName       string `json:"full_name"`
	StargazerCount int    `json:"stargazers_count"`
}

type UserRepositories []UserRepository

func (a UserRepositories) Len() int           { return len(a) }
func (a UserRepositories) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a UserRepositories) Less(i, j int) bool { return a[i].StargazerCount > a[j].StargazerCount }

func getUserRepositories(u string) string {
	res, err := http.Get("https://api.github.com/users/" + u + "/repos?per_page=100")
	if err != nil {
		return fmt.Sprintf("Error occurred: %s", err)
	}
	defer res.Body.Close()

	var repositories UserRepositories

	if err := json.NewDecoder(res.Body).Decode(&repositories); err != nil {
		return fmt.Sprintf("Error occurred: %s", err)
	}

	sort.Sort(repositories)

	var out bytes.Buffer
	for _, repository := range repositories {
		fmt.Fprintf(&out, "https://github.com/%s with stars %d\n", repository.FullName, repository.StargazerCount)
	}

	return out.String()
}

func SplitSubN(s string, n int) []string {
	sub := ""
	subs := []string{}

	runes := bytes.Runes([]byte(s))
	l := len(runes)
	for i, r := range runes {
		sub = sub + string(r)
		if (i + 1) % n == 0 {
			subs = append(subs, sub)
			sub = ""
		} else if (i + 1) == l {
			subs = append(subs, sub)
		}
	}

	return subs
}
