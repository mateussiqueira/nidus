package nidus

import (
	"encoding/json"
	"fmt"
)

type DomainsService struct {
	client *Client
}

func (s *DomainsService) List(projectID string) ([]DomainEntry, error) {
	resp, err := s.client.do("GET", "/api/projects/"+projectID+"/domains", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("list domains failed: %s", resp.Status)
	}

	var domains []DomainEntry
	if err := json.NewDecoder(resp.Body).Decode(&domains); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return domains, nil
}

func (s *DomainsService) Add(projectID, domain string) (*DomainEntry, error) {
	req := struct {
		Domain string `json:"domain"`
	}{Domain: domain}

	resp, err := s.client.do("POST", "/api/projects/"+projectID+"/domains", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("add domain failed: %s", resp.Status)
	}

	var entry DomainEntry
	if err := json.NewDecoder(resp.Body).Decode(&entry); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &entry, nil
}

func (s *DomainsService) Delete(projectID, domainID string) error {
	path := "/api/projects/" + projectID + "/domains/" + domainID
	resp, err := s.client.do("DELETE", path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("delete domain failed: %s", resp.Status)
	}
	return nil
}
