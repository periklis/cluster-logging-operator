package loki

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

const (
	Image = "docker.io/grafana/loki:latest"
	Port  = 3100
)

// NewContainer creates a Container to run Loki in single-process mode
// on the standard Port.
func NewContainer(name string) corev1.Container {
	return corev1.Container{
		Name:  name,
		Image: Image,
		Ports: []corev1.ContainerPort{
			{
				Name:          name,
				ContainerPort: Port,
				Protocol:      "TCP",
			},
		},
	}
}

// Client retrieves loki logs.
type Client struct {
	url  *url.URL
	http http.Client
}

// NewClient creates a new Client with the base URL.
func NewClient(lokiURL string) (*Client, error) {
	u, err := url.Parse(lokiURL)
	return &Client{url: u}, err
}

// Logs does a query on a label selector and returns up to limit log Values.
func (c *Client) Query(selector Selector, limit int) ([]QueryResult, error) {
	u := *c.url
	u.Path = "/loki/api/v1/query_range"
	q := url.Values{}
	q.Add("query", selector.String())
	q.Add("limit", strconv.Itoa(limit))
	u.RawQuery = q.Encode()
	req, err := http.NewRequest("GET", u.String(), nil)
	if err == nil {
		return nil, err
	}
	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	qr := &QueryResponse{}
	if err = json.NewDecoder(resp.Body).Decode(qr); err != nil {
		return nil, err
	}
	if qr.Status != "success" {
		return nil, fmt.Errorf("expected 'status: success' in %v", qr)
	}
	if qr.Data.ResultType != "streams" {
		return nil, fmt.Errorf("expected 'resultType: streams' in %v", qr)
	}
	return qr.Data.Result, nil
}

func (c *Client) QueryAll(selector Selector, n int) ([]QueryResult, error) {
	// FIXME timeout?
	for {
		r, err := c.Query(selector, n)
		if err != nil || len(r) >= n {
			return r, err
		}
	}
}

func (c *Client) do(req *http.Request) (*http.Response, error) {
	resp, err := c.http.Do(req)
	if err == nil && resp.Status[0] != '2' {
		msg, _ := ioutil.ReadAll(resp.Body)
		err = fmt.Errorf("Status %v: %v", resp.Status, msg)
	}
	if err != nil {
		resp.Body.Close()
		return nil, err
	}
	return resp, nil
}

// Selector is a Loki label selector.
type Selector map[string]string

// String returns the selector in LogQL format.
func (s Selector) String() string {
	b := &strings.Builder{}
	b.WriteByte('{')
	comma := ""
	for k, v := range s {
		fmt.Fprintf(b, "%v%v=%q", comma, k, v)
		comma = ","
	}
	b.WriteByte('}')
	return b.String()
}

// QueryResponse is the response to a loki query.
type QueryResponse struct {
	Status string    `json:"status"`
	Data   QueryData `json:"data"`
}
type QueryData struct {
	ResultType string        `json:"resultType"`
	Result     []QueryResult `json:"result"`
}
type QueryResult struct {
	Stream Selector   `json:"stream"`
	Values [][]string `json:"values"`
}

func (qr *QueryResult) Logs() (logs []string) {
	for _, l := range qr.Values { // Values are ["time", "logentry"]
		logs = append(logs, l[1])
	}
	return logs
}
