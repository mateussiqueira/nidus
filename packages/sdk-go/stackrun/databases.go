package nidus

import (
	"encoding/json"
	"fmt"
)

type DatabasesService struct {
	client *Client
}

func (s *DatabasesService) List() ([]Database, error) {
	resp, err := s.client.do("GET", "/api/databases", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("list databases failed: %s", resp.Status)
	}

	var databases []Database
	if err := json.NewDecoder(resp.Body).Decode(&databases); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return databases, nil
}

func (s *DatabasesService) Create(name, projectID string) (*Database, error) {
	resp, err := s.client.do("POST", "/api/databases", &CreateDatabaseRequest{
		Name:      name,
		ProjectID: projectID,
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("create database failed: %s", resp.Status)
	}

	var database Database
	if err := json.NewDecoder(resp.Body).Decode(&database); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &database, nil
}
