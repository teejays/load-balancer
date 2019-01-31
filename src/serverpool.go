package main

import (
	"errors"
	"fmt"
	//"net/http"
	// "net/http/httputil"
	"net/url"
	//"strings"
	"sync"
	"time"

	"github.com/teejays/clog"
)

const HealthCheckInterval time.Duration = time.Second * 10

type ServerPool struct {
	Servers      []*BackendServer
	CurrentIndex int
	sync.Mutex
}

var (
	ErrNoServerAddressForPool = errors.New("Empty server address list provided for pool")
	ErrDuplicateServerAddress = errors.New("More than one server found with the same address")
	ErrNoHealthyServer        = errors.New("No healthy servers found")
)

func NewServerPool(addrs ServerAddresses) (*ServerPool, error) {
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

func (pool *ServerPool) GetServer(handler func(*ServerPool) (*BackendServer, error)) (*BackendServer, error) {
	return handler(pool)
}

func RoundRobin(pool *ServerPool) (*BackendServer, error) {
	var cnt int
	for {
		// If we have looked at all the servers and haven't found any healthy,
		// we should just error out with no healthy servers.
		if cnt >= len(pool.Servers) {
			break
		}

		// Start from the index of the last used server and
		if pool.Servers[pool.CurrentIndex].IsHealthy() {
			server := pool.Servers[pool.CurrentIndex]
			pool.IncrementCurrentIndex()
			return server, nil
		}

		pool.IncrementCurrentIndex()
		cnt++
	}

	return nil, ErrNoHealthyServer
}

func (pool *ServerPool) IncrementCurrentIndex() {
	pool.Lock()
	defer pool.Unlock()
	if pool.CurrentIndex+1 >= len(pool.Servers) {
		pool.CurrentIndex = 0
	} else {
		pool.CurrentIndex++
	}
}

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
