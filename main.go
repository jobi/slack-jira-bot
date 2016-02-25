package main

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/jobi/slack-jira-bot/jira"
	"github.com/jobi/slack-jira-bot/slack"
	"github.com/jobi/slack-jira-bot/slack/rtm"
	"github.com/joeshaw/envdecode"
)

type Config struct {
	JIRAURL           string `env:"SLACK_JIRA_BOT_JIRA_URL,required"`
	JIRAUsername      string `env:"SLACK_JIRA_BOT_JIRA_USERNAME,required"`
	JIRAPassword      string `env:"SLACK_JIRA_BOT_JIRA_PASSWORD,required"`
	JIRAAgileBoard    string `env:"SLACK_JIRA_BOT_JIRA_AGILE_BOARD,required"`
	JIRAProjectPrefix string `env:"SLACK_JIRA_BOT_JIRA_PROJECT_PREFIX,required"`
	SlackToken        string `env:"SLACK_JIRA_BOT_SLACK_TOKEN,required"`
}

type SlackJIRABot struct {
	slack.Slack
	jira.JIRA
	*rtm.Session
	JIRAAgileBoard    string
	JIRAProjectPrefix string
}

func (s *SlackJIRABot) PostCurrentSprint(channel string, filterComponents []string) error {
	componentIsInFilter := func(component string) bool {
		if len(filterComponents) == 0 {
			return true
		}

		for _, c := range filterComponents {
			if strings.ToLower(c) == strings.ToLower(component) {
				return true
			}
		}

		return false
	}

	sprints, err := s.GetSprintsOfBoard(s.JIRAAgileBoard)
	if err != nil {
		return err
	}

	for _, sprint := range sprints {
		if sprint.State == jira.SprintStateActive {
			issues, err := s.GetIssuesOfSprint(s.JIRAAgileBoard, sprint.ID)
			if err != nil {
				return err
			}

			issuesByComponent := make(map[string][]jira.Issue)

			for _, i := range issues {
				if i.Fields.Status.Category.Name == "Done" {
					continue
				}

				for _, c := range i.Fields.Components {
					if componentIsInFilter(c.Name) {
						issuesByComponent[c.Name] = append(issuesByComponent[c.Name], i)
					}
				}
			}

			for component, issues := range issuesByComponent {
				var attachments []slack.Attachment

				for _, i := range issues {
					attachments = append(attachments, slack.Attachment{
						Title:     fmt.Sprintf("%s: %s", i.Key, i.Fields.Summary),
						TitleLink: fmt.Sprintf("%sbrowse/%s", s.URL, i.Key),
					})
				}

				if err := s.PostMessage(channel, s.Self.Name, component, attachments); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (s *SlackJIRABot) ResolveIssue(issueKey, comment string) error {
	if err := s.JIRA.ResolveIssue(issueKey, comment); err != nil {
		return err
	}

	return nil
}

func (s *SlackJIRABot) PostIssue(channel, issueKey string) error {
	issue, err := s.Issue(issueKey)
	if err != nil {
		return err
	}

	var fields []slack.AttachmentField

	fields = append(fields, slack.AttachmentField{
		Title: "Reporter",
		Value: issue.Fields.Reporter.DisplayName,
		Short: true,
	})

	fields = append(fields, slack.AttachmentField{
		Title: "Type",
		Value: issue.Fields.IssueType.Name,
		Short: true,
	})

	if len(issue.Fields.FixVersions) > 0 {
		fields = append(fields, slack.AttachmentField{
			Title: "Fix Version",
			Value: issue.Fields.FixVersions[0].Name,
			Short: true,
		})
	}

	if issue.Fields.Assignee != nil {
		fields = append(fields, slack.AttachmentField{
			Title: "Assignee",
			Value: issue.Fields.Assignee.DisplayName,
			Short: true,
		})
	}

	var componentsName []string
	for _, c := range issue.Fields.Components {
		componentsName = append(componentsName, c.Name)
	}

	if len(componentsName) > 0 {
		fields = append(fields, slack.AttachmentField{
			Title: "Components",
			Value: strings.Join(componentsName, ","),
			Short: true,
		})
	}

	if err := s.PostMessage(channel, s.Self.Name, "", []slack.Attachment{
		slack.Attachment{
			Title:     issue.Fields.Summary,
			TitleLink: fmt.Sprintf("%sbrowse/%s", s.URL, issueKey),
			Text:      issue.Fields.Description,
			Fields:    fields,
		},
	}); err != nil {
		return err
	}

	return nil
}

func (s *SlackJIRABot) StartSession() error {
	session, err := rtm.StartSession(&s.Slack)
	if err != nil {
		return err
	}

	s.Session = session
	return nil
}

func (s *SlackJIRABot) HandleMessage(m *rtm.MessageMessage) error {
	t := strings.TrimSpace(m.Text)

	mentionRegexp := regexp.MustCompile(fmt.Sprintf("^@?%s:?(.*)", s.Self.Name))
	issueRegexp := regexp.MustCompile(fmt.Sprintf("((?i)%s)-[0-9]+", s.JIRAProjectPrefix))
	resolveRegexp := regexp.MustCompile(fmt.Sprintf("^resolve (((?i)%s-)?[0-9]+)(.*)$", s.JIRAProjectPrefix))
	sprintRexep := regexp.MustCompile("^sprint(.*)$")

	if submatches := mentionRegexp.FindStringSubmatch(t); len(submatches) > 1 {
		c := strings.TrimSpace(submatches[1])

		if commandSubmatches := resolveRegexp.FindStringSubmatch(c); len(commandSubmatches) == 4 {
			key := strings.ToUpper(strings.TrimSpace(commandSubmatches[1]))
			comment := strings.TrimSpace(commandSubmatches[3])

			if commandSubmatches[2] == "" {
				key = fmt.Sprintf("%s-%s", s.JIRAProjectPrefix, key)
			}

			if err := s.ResolveIssue(key, comment); err != nil {
				return err
			}

			return nil
		}

		if commandSubmatches := sprintRexep.FindStringSubmatch(c); len(commandSubmatches) > 0 {
			arguments := strings.TrimSpace(commandSubmatches[1])

			var filterComponents []string

			if len(arguments) > 0 {
				filterComponents = strings.Split(arguments, ",")
			}

			for i, f := range filterComponents {
				filterComponents[i] = strings.TrimSpace(f)
			}

			if err := s.PostCurrentSprint(m.Channel, filterComponents); err != nil {
				log.Printf("Error posting current sprint: %s", err)
				return err
			}

			return nil
		}
	}

	if issueKey := issueRegexp.FindString(t); issueKey != "" {
		issueKey = strings.ToUpper(issueKey)

		if err := s.PostIssue(m.Channel, issueKey); err != nil {
			return err
		}
	}

	return nil
}

func main() {
	var cfg Config
	if err := envdecode.Decode(&cfg); err != nil {
		log.Printf("Failed to read configuration: %s", err)
		return
	}

	s := &SlackJIRABot{
		Slack: slack.Slack{
			Token: cfg.SlackToken,
		},
		JIRA: jira.JIRA{
			Username: cfg.JIRAUsername,
			Password: cfg.JIRAPassword,
			URL:      cfg.JIRAURL,
			HTTPDoer: http.DefaultClient,
		},
		JIRAProjectPrefix: cfg.JIRAProjectPrefix,
		JIRAAgileBoard:    cfg.JIRAAgileBoard,
	}

	if err := s.StartSession(); err != nil {
		fmt.Printf("Error starting RTM: %s\n", err)
		return
	}

	for m := range s.Incoming {
		switch m.Type {
		case rtm.MessageTypeMessage:
			if err := s.HandleMessage(&m.Message); err != nil {
				log.Printf("Error processing message '%s': %s", m.Message.Text, err)
			}
		}
	}
}
