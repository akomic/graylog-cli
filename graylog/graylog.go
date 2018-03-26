package graylog

import (
	"encoding/json"
	"fmt"
	"net/http"
	// "net/url"
	"io/ioutil"
	// "time"
)

const baseURL string = "http://127.0.0.1/api"

type Client struct {
	Username string
	Password string
}

func NewBasicAuthClient(username, password string) *Client {
	return &Client{
		Username: username,
		Password: password,
	}
}

type Streams struct {
	Data []map[string]interface{} `json:"streams"`
}

func (s *Client) doRequest(req *http.Request) ([]byte, error) {
	req.SetBasicAuth(s.Username, s.Password)
	req.Header.Set("Accept", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if 200 != resp.StatusCode {
		return nil, fmt.Errorf("%s", body)
	}
	return body, nil
}

func (s *Client) ListStreams() (Streams, error) {
	var streams Streams

	url := fmt.Sprintf(baseURL + "/streams")
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return streams, err
	}
	bytes, err := s.doRequest(req)
	if err != nil {
		return streams, err
	}
	err = json.Unmarshal(bytes, &streams)
	if err != nil {
		return streams, err
	}
	return streams, nil
}

type Messages struct {
	Data []map[string]interface{} `json:"messages"`
}

func (s *Client) SearchLogs(queryString, streamID string) (Messages, error) {
	var messages Messages

	url := fmt.Sprintf(baseURL + "/search/universal/absolute")

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return messages, err
	}
	q := req.URL.Query()
	q.Add("query", queryString)
	// q.Add("fields", "timestamp,source,message")
	q.Add("filter", fmt.Sprintf("streams:%s", streamID))
	q.Add("fields", "*")
	q.Add("limit", "50")
	q.Add("sort", "timestamp:asc")

	// now := time.Now()
	// then := now.Add(-12 * time.Hour)

	// msgDetails := fmt.Sprintf("%s\n", then.UTC().Format(time.RFC3339))

	// q.Add("from", fmt.Sprintf("%s", then.UTC().Format(time.RFC3339)))
	// q.Add("to", fmt.Sprintf("%s", now.UTC().Format(time.RFC3339)))

	// q.Add("from", "2018-03-01T09:54:20.000Z")
	// q.Add("to", "2018-03-09T09:54:20.000Z")

	q.Add("from", "2018-03-01T09:54:20.000Z")
	q.Add("to", "2018-03-09T09:54:20.000Z")

	req.URL.RawQuery = q.Encode()

	bytes, err := s.doRequest(req)
	if err != nil {
		return messages, err
	}
	err = json.Unmarshal(bytes, &messages)
	if err != nil {
		return messages, err
	}
	return messages, nil
}
