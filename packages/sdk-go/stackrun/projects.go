package nidus

import (
	"encoding/json"
	"fmt"
)

type ProjectsService struct {
	client *Client
}

func (s *ProjectsService) List() ([]Project, error) {
	resp, err := s.client.do("GET", "/api/projects", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("list projects failed: %s", resp.Status)
	}

	var projects []Project
	if err := json.NewDecoder(resp.Body).Decode(&projects); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return projects, nil
}

func (s *ProjectsService) Get(id string) (*Project, error) {
	resp, err := s.client.do("GET", "/api/projects/"+id, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("get project failed: %s", resp.Status)
	}

	var project Project
	if err := json.NewDecoder(resp.Body).Decode(&project); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &project, nil
}

func (s *ProjectsService) Create(req CreateProjectRequest) (*Project, error) {
	resp, err := s.client.do("POST", "/api/projects", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("create project failed: %s", resp.Status)
	}

	var project Project
	if err := json.NewDecoder(resp.Body).Decode(&project); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &project, nil
}

func (s *ProjectsService) Deploy(id, branch string) (*Deployment, error) {
	resp, err := s.client.do("POST", "/api/projects/"+id+"/deploy", &DeployRequest{
		Branch: branch,
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("deploy failed: %s", resp.Status)
	}

	var deployment Deployment
	if err := json.NewDecoder(resp.Body).Decode(&deployment); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &deployment, nil
}

func (s *ProjectsService) Envs(id string) ([]EnvVar, error) {
	resp, err := s.client.do("GET", "/api/projects/"+id+"/envs", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("list envs failed: %s", resp.Status)
	}

	var envs []EnvVar
	if err := json.NewDecoder(resp.Body).Decode(&envs); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return envs, nil
}

func (s *ProjectsService) EnvSet(id, key, value string) error {
	resp, err := s.client.do("POST", "/api/projects/"+id+"/envs", &EnvSetRequest{
		Key:   key,
		Value: value,
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("set env failed: %s", resp.Status)
	}
	return nil
}

func (s *ProjectsService) Volumes(id string) ([]Volume, error) {
	resp, err := s.client.do("GET", "/api/projects/"+id+"/volumes", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("list volumes failed: %s", resp.Status)
	}

	var volumes []Volume
	if err := json.NewDecoder(resp.Body).Decode(&volumes); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return volumes, nil
}

func (s *ProjectsService) VolumeCreate(id, name, mountPath string) (*Volume, error) {
	resp, err := s.client.do("POST", "/api/projects/"+id+"/volumes", &VolumeCreateRequest{
		Name:      name,
		MountPath: mountPath,
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("create volume failed: %s", resp.Status)
	}

	var volume Volume
	if err := json.NewDecoder(resp.Body).Decode(&volume); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &volume, nil
}
