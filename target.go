package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/teejays/clog"
)

// HealthEndpoint is the backend server endpoint that provides the health status information
const HealthEndpoint string = "_health"

// Health Status identifiers
const (
	StatusDegraded HealthStatus = iota
	StatusHealthy
)

type (
	TargetServer struct {
		Address       string
		URL           *url.URL
		Load          int
		Health        HealthStatus
		HealthUpdated time.Time
	}

	// HealthStatus is a type alias to better handle target server states.
	HealthStatus int

	// HealthResponse is the structure of response received from the /_health endpoint of the target servers.
	HealthResponse struct {
		State   string
		Message string
	}
)

var (
	ErrEmptyAddress                  = errors.New("address passed for NewTargetServer is empty")
	ErrEmptyStatusInHealthResponse   = errors.New("status field in the health response is empty")
	ErrInvalidStatusInHealthResponse = errors.New("status field in the health response is invalid")
)

func NewTargetServer(address string) (*TargetServer, error) {
	if strings.TrimSpace(address) == "" {
		return nil, ErrEmptyAddress
	}

	// Create a url.URL for the address
	_url, err := url.Parse(address)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse to URL: %s", err)
	}

	server := TargetServer{
		Address: address,
		URL:     _url,
	}

	return &server, nil

}

// IsHealthy returns true if the target server s is in a healthy state.
func (s *TargetServer) IsHealthy() bool {
	if s.Health == StatusHealthy {
		return true
	}
	return false
}

// RefreshHealthStatus refreshes the health status record of the target server s by making a fresh call
// to the health endpoint for the target server.
func (s *TargetServer) RefreshHealthStatus() error {
	// Get the new health & update the instance
	status, err := s.GetNewHealthStatus()
	s.SetStatus(status)
	return err
}

// Degrade marks the target server s as degraded. It is equivalent to calling SetStatus(StatusDegraded).
// A degraded server is excluded while selecting target servers for forwarding client requests.
func (s *TargetServer) Degrade() {
	s.SetStatus(StatusDegraded)
}

// SetStatus sets the health to status.
func (s *TargetServer) SetStatus(status HealthStatus) {
	if status == StatusDegraded && s.Health == StatusHealthy {
		clog.Warningf("A server is being unhealthy: %s", s.Address)
	}
	if status == StatusHealthy && s.Health == StatusDegraded {
		clog.Noticef("A server is being marked healthy: %s", s.Address)
	}
	s.Health = status
	s.HealthUpdated = time.Now()

}

// GetNewHealthStatus returns a new HealthStatus for the target server. It does not update
// the state for the server, only fetches a new state. It returns a StatusDegraded and an error
// if it encounters an error.
func (s *TargetServer) GetNewHealthStatus() (HealthStatus, error) {

	// Make a get request to _health endpoint
	url := fmt.Sprintf("%s/%s", s.Address, HealthEndpoint)
	resp, err := http.Get(url)
	if err != nil {
		return StatusDegraded, err
	}
	defer resp.Body.Close()

	// Read the response
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return StatusDegraded, err
	}

	// Unmarshall the response into Json
	var hr HealthResponse
	err = json.Unmarshal(b, &hr)
	if err != nil {
		return StatusDegraded, err
	}

	// Get the status from the response and return
	return getHealthStatusFromResponse(hr)
}

// getHealthStatusFromResponse is a util function for GetNewHealthStatus. It maps the response
// from the health endpoint of the target server to a HealthStatus type.
func getHealthStatusFromResponse(hr HealthResponse) (HealthStatus, error) {
	// Have a map that can link the response state to HealthStatus type
	var m = map[string]HealthStatus{
		"healthy":  StatusHealthy,
		"degraded": StatusDegraded,
	}

	if strings.TrimSpace(hr.State) == "" {
		return StatusDegraded, ErrEmptyStatusInHealthResponse
	}

	status, ok := m[hr.State]
	if !ok {
		clog.Warningf("Status field in the health response is invalid: %s", hr.State)
		return StatusDegraded, ErrInvalidStatusInHealthResponse
	}

	return status, nil
}
