package rtm

import (
	"encoding/json"

	"golang.org/x/net/websocket"

	"github.com/jobi/slack-jira-bot/slack"
)

type MessageType string

const (
	MessageTypeHello      MessageType = "hello"
	MessageTypeMessage                = "message"
	MessageTypeUserTyping             = "user_typing"
)

type MessageBase struct {
	Type MessageType `json:"type"`
}

type MessageHello struct {
}

type MessageMessage struct {
	Channel   string `json:"channel"`
	User      string `json:"user"`
	Text      string `json:"text"`
	Timestamp string `json:"ts"`
}

type MessageUserTyping struct {
	Channel string `json:"channel"`
	User    string `json:"user"`
}

type Message struct {
	MessageBase
	Hello      MessageHello
	Message    MessageMessage
	UserTyping MessageUserTyping
}

func (m *Message) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, &m.MessageBase); err != nil {
		return err
	}

	switch m.Type {
	case MessageTypeHello:
	case MessageTypeMessage:
		if err := json.Unmarshal(data, &m.Message); err != nil {
			return err
		}
	case MessageTypeUserTyping:
		if err := json.Unmarshal(data, &m.UserTyping); err != nil {
			return err
		}
	}

	return nil
}

type State struct {
	Self  slack.User   `json:"self"`
	Users []slack.User `json:"users"`
}

func (s *State) FindUser(ID string) *slack.User {
	for _, u := range s.Users {
		if u.ID == ID {
			return &u
		}
	}

	return nil
}

type Session struct {
	Conn *websocket.Conn
	State
	Incoming   <-chan Message
	OutgoingID int
}

type startRTMResponse struct {
	slack.Response
	slack.APIError
	URL string `json:"url"`
	State
}

func startRTMRequest(slack *slack.Slack) (*startRTMResponse, error) {
	resp, err := slack.CallAPI("rtm.start", nil)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	var startRTMResp startRTMResponse
	if err := json.NewDecoder(resp.Body).Decode(&startRTMResp); err != nil {
		return nil, err
	}

	if startRTMResp.OK != true {
		return nil, &startRTMResp.APIError
	}

	return &startRTMResp, nil
}

type outgoingMessage struct {
	ID      int    `json:"id"`
	Type    string `json:"type"`
	Channel string `json:"channel"`
	Text    string `json:"text"`
}

func (s *Session) SendMessage(channel, text string) error {
	m := &outgoingMessage{
		ID:      s.OutgoingID,
		Type:    "message",
		Channel: channel,
		Text:    text,
	}

	if err := websocket.JSON.Send(s.Conn, m); err != nil {
		return err
	}

	s.OutgoingID++
	return nil
}

func StartSession(slack *slack.Slack) (*Session, error) {
	r, err := startRTMRequest(slack)
	if err != nil {
		return nil, err
	}

	ws, err := websocket.Dial(r.URL, "", "http://localhost")
	if err != nil {
		return nil, err
	}

	incoming := make(chan Message)
	go func(ws *websocket.Conn, incoming chan<- Message) {
		for {
			var m Message
			if err := websocket.JSON.Receive(ws, &m); err != nil {
				close(incoming)
				return
			}

			incoming <- m
		}
	}(ws, incoming)

	session := &Session{
		State:      r.State,
		Conn:       ws,
		Incoming:   incoming,
		OutgoingID: 1,
	}

	return session, nil
}
