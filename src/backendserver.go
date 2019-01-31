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

const (
	// HealthEndpoint is the backend server endpoint that provides the health status information
	HealthEndpoint string = "_health"

	DegradedServer HealthStatus = iota
	HealthyServer
)

type (
	BackendServer struct {
		Address       string
		URL           *url.URL
		Load          int
		Health        HealthStatus
		HealthUpdated time.Time
	}

	HealthStatus int

	HealthResponse struct {
		State   string
		Message string
	}
)

var (
	ErrEmptyStatusInHealthResponse   = errors.New("status field in the health response is empty")
	ErrInvalidStatusInHealthResponse = errors.New("status field in the health response is invalid")
)

func (s *BackendServer) IsHealthy() bool {
	if s.Health == HealthyServer {
		return true
	}
	return false
}

func (s *BackendServer) RefreshHealthStatus() error {
	// Get the new health & update the instance
	status, err := s.GetNewHealthStatus()
	s.SetStatus(status)
	return err
}

func (s *BackendServer) Degrade() {
	s.SetStatus(DegradedServer)
}

func (s *BackendServer) SetStatus(status HealthStatus) {
	if status == DegradedServer && s.Health == HealthyServer {
		clog.Warningf("A server has gone down: %s", s.Address)
	}
	if status == HealthyServer && s.Health == DegradedServer {
		clog.Noticef("A server has come back up: %s", s.Address)
	}
	s.Health = status
	s.HealthUpdated = time.Now()

}

func (s *BackendServer) GetNewHealthStatus() (HealthStatus, error) {

	// Make a get request to _health endpoint
	url := fmt.Sprintf("%s/%s", s.Address, HealthEndpoint)
	resp, err := http.Get(url)
	if err != nil {
		return DegradedServer, err
	}
	defer resp.Body.Close()

	// Read the response
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return DegradedServer, err
	}

	// Unmarshall the response into Json
	var hr HealthResponse
	err = json.Unmarshal(b, &hr)
	if err != nil {
		return DegradedServer, err
	}

	// Get the status from the response and return
	return getHealthStatusFromResponse(hr)
}

func getHealthStatusFromResponse(hr HealthResponse) (HealthStatus, error) {
	// Have a map that can link the response state to HealthStatus type
	var m = map[string]HealthStatus{
		"healthy":  HealthyServer,
		"degraded": DegradedServer,
	}

	if strings.TrimSpace(hr.State) == "" {
		return DegradedServer, ErrEmptyStatusInHealthResponse
	}

	status, ok := m[hr.State]
	if !ok {
		clog.Warningf("Status field in the health response is invalid: %s", hr.State)
		return DegradedServer, ErrInvalidStatusInHealthResponse
	}

	return status, nil
}
