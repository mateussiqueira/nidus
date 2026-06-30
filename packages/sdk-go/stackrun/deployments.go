package nidus

import (
	"encoding/json"
	"fmt"
	"io"
)

type DeploymentsService struct {
	client *Client
}

func (s *DeploymentsService) List(projectID string) ([]Deployment, error) {
	resp, err := s.client.do("GET", "/api/projects/"+projectID+"/deployments", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("list deployments failed: %s", resp.Status)
	}

	var deployments []Deployment
	if err := json.NewDecoder(resp.Body).Decode(&deployments); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return deployments, nil
}

func (s *DeploymentsService) Get(projectID, deploymentID string) (*Deployment, error) {
	path := "/api/projects/" + projectID + "/deployments/" + deploymentID
	resp, err := s.client.do("GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("get deployment failed: %s", resp.Status)
	}

	var deployment Deployment
	if err := json.NewDecoder(resp.Body).Decode(&deployment); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &deployment, nil
}

func (s *DeploymentsService) Logs(projectID, deploymentID string) (string, error) {
	path := "/api/projects/" + projectID + "/deployments/" + deploymentID + "/logs"
	resp, err := s.client.do("GET", path, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("get logs failed: %s", resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read logs: %w", err)
	}
	return string(data), nil
}
