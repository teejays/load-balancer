package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/teejays/clog"
)

type (
	ServerPool struct {
		Servers      []*BackendServer
		CurrentIndex int
		L            sync.Mutex
	}

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

func NewServerPool(addrs []string) (*ServerPool, error) {
	// Validate that we have addresses availalble
	if len(addrs) < 1 {
		return nil, ErrNoServerAddressForPool
	}

	// Populate the pool with newly created BackendServer instances
	var pool ServerPool
	pool.Servers = make([]*BackendServer, len(addrs))

	var seen = make(map[string]bool)
	for i, s := range addrs {
		if seen[s] {
			return nil, ErrDuplicateServerAddress
		}
		_url, err := url.Parse(s)
		if err != nil {
			return nil, fmt.Errorf("Failed to parse to URL: %s", err)
		}

		pool.Servers[i] = &BackendServer{
			Address: s,
			URL:     _url,
		}
		seen[s] = true
	}

	// Goroutine: Start background health check process for pool
	go func() {
		// Start an infinite loop
		for {
			// In the infinite loop, check health of all the servers,
			// one by one, after a set interval

			// Initiate updating health statuses for all servers
			for _, server := range pool.Servers {
				err := server.RefreshHealthStatus()
				if err != nil {
					clog.Errorf("There was an error updating the health for server: %s\n%s", server.Address, err)
				}
			}

			time.Sleep(HealthCheckInterval)
		}
	}()

	return &pool, nil
}

func (pool *ServerPool) GetServer() (*BackendServer, error) {
	var index int

	// Default Method: Round Robin
	var cnt int
	for {
		// If we have looked at all the servers and haven't found any healthy,
		// we should just error out with no healthy servers.
		if cnt >= len(pool.Servers) {
			return nil, ErrNoHealthyServer
		}

		// Start from the index of the last used server and
		clog.Debugf("Current index is: %d", pool.CurrentIndex)
		if pool.Servers[pool.CurrentIndex].IsHealthy() {
			index = pool.CurrentIndex
			pool.IncrementCurrentIndex()
			break
		}

		pool.IncrementCurrentIndex()
		cnt++
	}

	return pool.Servers[index], nil

}

func (pool *ServerPool) IncrementCurrentIndex() {
	pool.L.Lock()
	defer pool.L.Unlock()
	if pool.CurrentIndex+1 >= len(pool.Servers) {
		pool.CurrentIndex = 0
	} else {
		pool.CurrentIndex++
	}

	clog.Debugf("Pool's current index set to: %d", pool.CurrentIndex)
}

// HealthEndpoint defines the backend server endpoint
// that provides the health status information
const HealthEndpoint string = "_health"
const HealthCheckInterval time.Duration = time.Second * 10

// Health Status
const (
	UnknownHealth HealthStatus = iota
	HealthyServer
	DegradedServer
)

var (
	ErrNoServerAddressForPool        = errors.New("Empty server address list provided for pool")
	ErrDuplicateServerAddress        = errors.New("More than one server found with the same address")
	ErrEmptyStatusInHealthResponse   = errors.New("status field in the health response is empty")
	ErrInvalidStatusInHealthResponse = errors.New("status field in the health response is invalid")
	ErrNoHealthyServer               = errors.New("No healthy servers found")
)

func (s *BackendServer) IsHealthy() bool {
	if s.Health == HealthyServer {
		return true
	}
	return false
}

func (s *BackendServer) SetStatus(status HealthStatus) {
	s.Health = status
	s.HealthUpdated = time.Now()
}

func (s *BackendServer) RefreshHealthStatus() error {
	// Get the new health & update the instance
	status, err := s.GetNewHealthStatus()
	s.SetStatus(status)
	return err
}

func (s *BackendServer) GetNewHealthStatus() (HealthStatus, error) {

	// Make a get request to _health endpoint
	url := fmt.Sprintf("%s/%s", s.Address, HealthEndpoint)
	resp, err := http.Get(url)
	if err != nil {
		return UnknownHealth, err
	}
	defer resp.Body.Close()

	// Read the response
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return UnknownHealth, err
	}

	// Unmarshall the response into Json
	var hr HealthResponse
	err = json.Unmarshal(b, &hr)
	if err != nil {
		return UnknownHealth, err
	}

	// Get the status from the response and return
	return hr.GetHealthStatusFromMap()
}

func (hr HealthResponse) GetHealthStatusFromMap() (HealthStatus, error) {
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

type ServerAddrsType []string

func (b *ServerAddrsType) String() string {
	return "ServerAddrsType"
}

func (b *ServerAddrsType) Set(s string) error {
	if b == nil {
		return fmt.Errorf("Set() called on nil server flags")
	}
	clog.Debug("ServerAddrsType.Set() called.")
	*b = append(*b, s)
	return nil
}
