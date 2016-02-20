package jira

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type HTTPDoer interface {
	Do(r *http.Request) (*http.Response, error)
}

type HTTPError struct {
	StatusCode int
	Status     string
	Body       string
}

func NewHTTPError(resp *http.Response) *HTTPError {
	body, _ := ioutil.ReadAll(resp.Body)

	return &HTTPError{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		Body:       string(body),
	}
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP Error: %d - %s - %s", e.StatusCode, e.Status, e.Body)
}

type JIRA struct {
	HTTPDoer
	URL      string
	Username string
	Password string
}

type User struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
}

type Issue struct {
	Key    string      `json:"key"`
	Fields IssueFields `json:"fields"`
}

type IssueFields struct {
	Summary     string      `json:"summary"`
	Description string      `json:"description"`
	Reporter    User        `json:"reporter"`
	IssueType   IssueType   `json:"issuetype"`
	FixVersions []Version   `json:"fixVersions"`
	Assignee    *User       `json:"assignee"`
	Components  []Component `json:"components"`
	Status      IssueStatus `json:"status"`
}

type IssueType struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type IssueStatus struct {
	ID       string              `json:"id"`
	Name     string              `json:"name"`
	Category IssueStatusCategory `json:"statusCategory"`
}

type IssueStatusCategory struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	ColorName string `json:"colorName"`
}

type Version struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Archived bool   `json:"archived"`
	Released bool   `json:"released"`
}

type Component struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type TransitionID struct {
	ID string `json:"id"`
}

type Transition struct {
	TransitionID
	Name string      `json:"name"`
	To   IssueStatus `json:"to"`
}

func (j *JIRA) DoWithAuth(req *http.Request) (*http.Response, error) {
	req.SetBasicAuth(j.Username, j.Password)
	return j.Do(req)
}

func (j *JIRA) Issue(key string) (*Issue, error) {
	url := fmt.Sprintf("%srest/api/latest/issue/%s", j.URL, key)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := j.DoWithAuth(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, NewHTTPError(resp)
	}

	var issue Issue
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		return nil, err
	}

	return &issue, nil
}

type Pagination struct {
	MaxResults int `json:"maxResults"`
	StartAt    int `json:"startAt"`
}

type PaginatedIssues struct {
	Pagination
	Issues []Issue `json:"issues"`
}

type GetTransitionsResponse struct {
	Transitions []Transition `json:"transitions"`
}

type TransitionUpdate struct {
	Comment []map[string]TransitionUpdateComment `json:"comment"`
}

type TransitionUpdateComment struct {
	Body string `json:"body"`
}

type DoTransitionRequest struct {
	TransitionID `json:"transition"`
	Update       *TransitionUpdate `json:"update,omitempty"`
}

func (j *JIRA) TransitionIssue(issueKey, comment, status string) error {
	url := fmt.Sprintf("%srest/api/latest/issue/%s/transitions", j.URL, issueKey)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := j.DoWithAuth(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return NewHTTPError(resp)
	}

	var getTransitionsResp GetTransitionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&getTransitionsResp); err != nil {
		return err
	}

	var transition *Transition
	for _, t := range getTransitionsResp.Transitions {
		if t.To.Name == status {
			transition = &t
			break
		}
	}

	if transition == nil {
		return fmt.Errorf("No transition for issue %s to status %s", issueKey, status)
	}

	doTransitionReq := &DoTransitionRequest{
		TransitionID: TransitionID{
			ID: transition.ID,
		},
	}

	if comment != "" {
		commentMap := make(map[string]TransitionUpdateComment)
		commentMap["add"] = TransitionUpdateComment{
			Body: comment,
		}
		doTransitionReq.Update = &TransitionUpdate{
			Comment: []map[string]TransitionUpdateComment{commentMap},
		}
	}

	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	if err := encoder.Encode(doTransitionReq); err != nil {
		return err
	}

	req, err = http.NewRequest("POST", url, &buffer)
	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/json")

	resp, err = j.DoWithAuth(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return NewHTTPError(resp)
	}

	return nil
}

func (j *JIRA) ResolveIssue(issueKey, comment string) error {
	return j.TransitionIssue(issueKey, comment, "Resolved")
}
