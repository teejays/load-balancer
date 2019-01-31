package main

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/teejays/clog"
)

// ServerPool is the primary data structure of this application. It holds an array of all the
// target servers, and allows picking of healthy target servers using round robin.
type ServerPool struct {
	Servers          []*TargetServer
	CurrentIndex     int
	PauseHealthCheck bool
	sync.Mutex
}

// HealthCheckInterval defines the interval between two subsequent health checks of all servers
var HealthCheckInterval time.Duration = time.Second * 10

var (
	ErrNoServerAddressForPool = errors.New("Empty server address list provided for pool")
	ErrDuplicateServerAddress = errors.New("More than one server found with the same address")
	ErrNoHealthyServer        = errors.New("No healthy servers found")
)

// NewServerPool creates a new ServerPool with it's servers array built from the addresses passed
// in the parameters. It also starts a goroutine to periodically check the health status of it's servers
func NewServerPool(addrs ServerAddresses) (*ServerPool, error) {
	// Validate that we have addresses availalble
	if len(addrs) < 1 {
		return nil, ErrNoServerAddressForPool
	}

	// Populate the pool with newly created TargetServer instances
	var pool ServerPool
	pool.Servers = make([]*TargetServer, len(addrs))

	var seen = make(map[string]bool)
	for i, s := range addrs {
		if seen[s] {
			return nil, ErrDuplicateServerAddress
		}
		seen[s] = true

		server, err := NewTargetServer(s)
		if err != nil {
			return nil, err
		}
		pool.Servers[i] = server

	}

	// goroutine to start the health check process for the pool servers
	go (&pool).RunHealthCheckProcess(HealthCheckInterval)

	return &pool, nil
}

// RunHealthCheck is blocking and should be run as a separate goroutine in most case.
// It's starts an infinite loop that periodically checks the health status of all the servers.
func (pool *ServerPool) RunHealthCheckProcess(interval time.Duration) {

	// Start an infinite loop
	for {
		// In the infinite loop, check health of all the servers,
		// one by one, after a set interval

		// Initiate updating health statuses for all servers
		if !pool.PauseHealthCheck {
			pool.RunHealthCheck()
		}

		time.Sleep(HealthCheckInterval)
	}
}

// RunHealthCheck runs a single iteration of going through all the servers and
// updating their health statuses.
func (pool *ServerPool) RunHealthCheck() {
	for _, server := range pool.Servers {
		err := server.RefreshHealthStatus()
		if err != nil {
			clog.Errorf("There was an error updating the health for server: %s\n%s", server.Address, err)
		}
	}
}

// GetServer uses the provided algo to pick and return a healthy target server from the pool.
func (pool *ServerPool) GetTargetServer(algo func(*ServerPool) (int, error)) (*TargetServer, error) {
	index, err := algo(pool)
	if err != nil {
		return nil, err
	}

	clog.Debugf("RoundRobin server selected: %d", index)

	return pool.Servers[index], nil
}

// RoundRobin is the default (and only) algorithm for picking a healthy server from the pool.
// It goes through the server in a loop and picks the next healthy server from the list.
func RoundRobin(pool *ServerPool) (int, error) {
	var cnt, index int
	for {
		// If we have looked at all the servers and haven't found any healthy,
		// we should just error out with no healthy servers.
		if cnt >= len(pool.Servers) {
			break
		}

		// Start from the index of the last used server and
		if pool.Servers[pool.CurrentIndex].IsHealthy() {
			index = pool.CurrentIndex
			pool.IncrementCurrentIndex()
			return index, nil
		}

		pool.IncrementCurrentIndex()
		cnt++
	}
	clog.Warn("No healthy servers found")
	return -1, ErrNoHealthyServer
}

// IncrementCurrentIndex atomically increments the current index pointer for the pool. Current index
// pointer is important as it provides a reference for what target server did we use last and where
// should we start searching for again.
func (pool *ServerPool) IncrementCurrentIndex() {
	pool.Lock()
	defer pool.Unlock()
	if pool.CurrentIndex+1 >= len(pool.Servers) {
		pool.CurrentIndex = 0
	} else {
		pool.CurrentIndex++
	}
}

// Functions to help mock change the state of the pool

func (pool *ServerPool) Delete() {
	pool = nil
}

func (pool *ServerPool) DegradeAll() {
	for _, t := range pool.Servers {
		t.Degrade()
	}
}

func (pool *ServerPool) HealthyAll() {
	for _, t := range pool.Servers {
		t.SetStatus(StatusHealthy)
	}
}

func (pool *ServerPool) Normalize() {
	pool.RunHealthCheck()
	pool.ResumeHealthChecks()
}

func (pool *ServerPool) PauseHealthChecks() {
	pool.PauseHealthCheck = true
}

func (pool *ServerPool) ResumeHealthChecks() {
	pool.PauseHealthCheck = false
}

// ServerAddresses implements flag.Var interface so it can allow us to pass multiple target server
// addresses in the command line while starting the load balancer.
type ServerAddresses []string

func (b *ServerAddresses) String() string {
	return "ServerAddresses"
}

func (b *ServerAddresses) Set(s string) error {
	if b == nil {
		return fmt.Errorf("Set() called on nil server flags")
	}
	*b = append(*b, s)
	return nil
}
