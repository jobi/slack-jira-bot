package jira

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type SprintState string

const (
	SprintStateActive SprintState = "active"
	SprintStateClosed             = "closed"
)

type Sprint struct {
	ID        int         `json:"id"`
	State     SprintState `json:"state"`
	Name      string      `json:"name"`
	StartDate time.Time   `json:"startDate"`
	EndDate   time.Time   `json:"endDate"`
}

type PaginatedSprints struct {
	Pagination
	Sprints []Sprint `json:"values"`
}

func (j *JIRA) GetSprintsOfBoard(boardID string) ([]Sprint, error) {
	var sprints []Sprint
	var startAt int
	for {
		v := url.Values{}
		v.Add("startAt", fmt.Sprintf("%d", startAt))

		url := fmt.Sprintf("%srest/agile/latest/board/%s/sprint?%s", j.URL, boardID, v.Encode())

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

		var paginatedSprints PaginatedSprints
		if err := json.NewDecoder(resp.Body).Decode(&paginatedSprints); err != nil {
			return nil, err
		}

		sprints = append(sprints, paginatedSprints.Sprints...)

		if len(paginatedSprints.Sprints) == paginatedSprints.MaxResults {
			startAt = startAt + paginatedSprints.MaxResults
		} else {
			break
		}
	}

	return sprints, nil
}

func (j *JIRA) GetIssuesOfSprint(boardID string, sprintID int) ([]Issue, error) {
	var issues []Issue
	var startAt int
	for {
		v := url.Values{}
		v.Add("startAt", fmt.Sprintf("%d", startAt))

		url := fmt.Sprintf("%srest/agile/latest/board/%s/sprint/%d/issue?%s", j.URL, boardID, sprintID, v.Encode())

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

		var paginatedIssues PaginatedIssues
		if err := json.NewDecoder(resp.Body).Decode(&paginatedIssues); err != nil {
			return nil, err
		}

		issues = append(issues, paginatedIssues.Issues...)

		if len(paginatedIssues.Issues) == paginatedIssues.MaxResults {
			startAt = startAt + paginatedIssues.MaxResults
		} else {
			break
		}
	}

	return issues, nil
}
