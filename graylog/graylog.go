package graylog

import (
	"encoding/json"
	"fmt"
	"net/http"
	// "net/url"
	"io/ioutil"
	// "time"
	log "github.com/sirupsen/logrus"
)

// Client credentials struct
type Client struct {
	BaseURL  string
	Username string
	Password string
}

// NewBasicAuthClient returns new Client credential struct
func NewBasicAuthClient(baseurl, username, password string) *Client {
	return &Client{
		BaseURL:  baseurl,
		Username: username,
		Password: password,
	}
}

// Streams JSON struct
type Streams struct {
	Data []map[string]interface{} `json:"streams"`
}

func (s *Client) doRequest(req *http.Request) ([]byte, error) {
	log.Infof("%v\n", s)
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

// ListStreams get from graylog server list of streams
func (s *Client) ListStreams() (Streams, error) {
	var streams Streams
	log.Infof("%v\n", s)
	url := fmt.Sprintf(s.BaseURL + "/streams")
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

// Messages JSON struct
type Messages struct {
	Data []map[string]interface{} `json:"messages"`
}

// SearchLogs function
func (s *Client) SearchLogs(queryString, streamID string) (Messages, error) {
	var messages Messages

	url := fmt.Sprintf(s.BaseURL + "/search/universal/absolute")

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
