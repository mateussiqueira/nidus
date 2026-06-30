package nidus

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Client struct {
	apiURL     string
	token      string
	httpClient *http.Client

	Projects    *ProjectsService
	Deployments *DeploymentsService
	Domains     *DomainsService
	Databases   *DatabasesService
	Auth        *AuthService
}

func NewClient(apiURL string) *Client {
	if apiURL == "" {
		apiURL = "https://api.stackrun.vercel.app"
	}
	c := &Client{
		apiURL:     apiURL,
		httpClient: &http.Client{},
	}
	c.Projects = &ProjectsService{client: c}
	c.Deployments = &DeploymentsService{client: c}
	c.Domains = &DomainsService{client: c}
	c.Databases = &DatabasesService{client: c}
	c.Auth = &AuthService{client: c}
	return c
}

func (c *Client) SetToken(token string) {
	c.token = token
}

func (c *Client) do(method, path string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal body: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	url := c.apiURL + path
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	return c.httpClient.Do(req)
}
