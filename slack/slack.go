package slack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

const apiURL = "https://slack.com/api"

type Slack struct {
	Token string
}

type User struct {
	ID   string `json:"id"`
	Name string `json:"name"`
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

type Response struct {
	OK bool `json:"ok"`
}

type APIError struct {
	ErrorCode string `json:"error"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("Slack API error: %s", e.ErrorCode)
}

type Attachment struct {
	Title      string            `json:"title"`
	TitleLink  string            `json:"title_link"`
	Text       string            `json:"text"`
	Fallback   string            `json:"fallback"`
	Color      string            `json:"color"`
	Pretext    string            `json:"pretext"`
	AuthorName string            `json:"author_name"`
	AuthorLink string            `json:"author_link"`
	Fields     []AttachmentField `json:"fields"`
}

type AttachmentField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

func (s *Slack) CallAPI(method string, arguments map[string]string) (*http.Response, error) {
	v := &url.Values{}
	v.Add("token", s.Token)

	for key, value := range arguments {
		v.Add(key, value)
	}

	url := fmt.Sprintf("%s/%s", apiURL, method)
	startReq, err := http.NewRequest("POST", url, bytes.NewReader([]byte(v.Encode())))
	if err != nil {
		return nil, err
	}

	startReq.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(startReq)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, NewHTTPError(resp)
	}

	return resp, nil
}

func (s *Slack) PostMessage(channel, username, text string, attachments []Attachment) error {
	args := make(map[string]string)

	args["channel"] = channel
	args["username"] = username
	args["text"] = text

	if attachments != nil {
		attachmentsJSON, err := json.Marshal(attachments)
		if err != nil {
			return err
		}
		args["attachments"] = string(attachmentsJSON)
	}

	resp, err := s.CallAPI("chat.postMessage", args)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	return nil
}
