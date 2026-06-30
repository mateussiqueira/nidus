package nidus

import (
	"encoding/json"
	"fmt"
)

type AuthService struct {
	client *Client
}

func (s *AuthService) Login(email, password string) (*User, error) {
	resp, err := s.client.do("POST", "/api/auth/login", &LoginRequest{
		Email:    email,
		Password: password,
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("login failed: %s", resp.Status)
	}

	var user User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &user, nil
}

func (s *AuthService) Register(email, password, name string) (*User, error) {
	resp, err := s.client.do("POST", "/api/auth/register", &RegisterRequest{
		Email:    email,
		Password: password,
		Name:     name,
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("register failed: %s", resp.Status)
	}

	var user User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &user, nil
}
