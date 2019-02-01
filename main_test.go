package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"sync"
	"testing"
	"time"
)

var targetPorts = []int{9000, 9001, 9002, 9003, 9004, 9005}
var serverAddrs ServerAddresses

func init() {
	// Make the interval smaller for testing
	HealthCheckInterval = time.Second * 2

	// Initialize the ServerAddress instance, just like if someone has passed all these args
	for _, p := range targetPorts {
		serverAddrs = append(serverAddrs, fmt.Sprintf("http://localhost:%d", p))
	}
}

// TestNewServerPool tests that we can successfully create a new ServerPool instance.
func TestNewServerPool(t *testing.T) {
	err := startTargetServers()
	if err != nil {
		t.Fatalf("failed to start target servers: %s", err)
	}
	defer stopTargetServers()

	// Initialize ServerAddresses & ServerPool
	pool, err = NewServerPool(serverAddrs)
	if err != nil {
		t.Error(err)
	}

	pool.CancelHealthCheck()
	pool = nil
}

// TestNewServerPool tests that successfully get a 503 when there are no healthy servers
func TestNoHealthyServer(t *testing.T) {
	var err error

	// Initialize the pool
	pool, err = NewServerPool(serverAddrs)
	if err != nil {
		t.Error(err)
	}
	defer func() {
		pool.CancelHealthCheck()
		pool = nil
	}()

	// Degrade all the servers and keep in that state
	pool.PauseHealthChecks()
	pool.DegradeAll()

	// Create a request to pass to our handler.
	r := httptest.NewRequest("GET", fmt.Sprintf("localhost:%d", listenerPortDeault), nil)
	w := httptest.NewRecorder()

	listenerHandler(w, r)

	// We should expect a 503
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected a 503 status code but got %d", w.Code)
	}

}

// TestNewServerPool makes concurrent requests to the load balancer and fails if it receives anything
// other than a 503 or 200
func TestConcurrent(t *testing.T) {
	var err error

	// Start Target servers
	err = startTargetServers()
	if err != nil {
		t.Fatalf("failed to start target servers: %s", err)
	}
	defer stopTargetServers()

	time.Sleep(5 * time.Second)

	// Initialize the pool
	pool, err = NewServerPool(serverAddrs)
	if err != nil {
		t.Error(err)
	}
	defer func() {
		pool.CancelHealthCheck()
		pool = nil
	}()

	time.Sleep(5 * time.Second)

	// Create some load to pass to our handler
	var concurrency = make(chan int, 1)
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		concurrency <- 4

		go func(i int) {
			defer func() {
				if r := recover(); r != nil {
					log.Fatalf("Panic (recovered) during TestConcurrent(): %s", r)
				}
				<-concurrency
				wg.Done()
			}()
			r := httptest.NewRequest("GET", fmt.Sprintf("http://localhost:%d", listenerPortDeault), nil)
			w := httptest.NewRecorder()
			listenerHandler(w, r)

			// We should expect a 200 or 503
			if w.Code != http.StatusServiceUnavailable && w.Code != http.StatusOK {
				t.Errorf("[%d] Expected a 200 or 503 status code but got %d", i, w.Code)
			}
		}(i)

		time.Sleep(time.Millisecond * 200)
	}

	wg.Wait()

}

// Todo: Write test for the case when the state of a healthy server changes after it has been picked up
// in the round robin. It is difficult to deterministically create this scenario

// TestRoundRobin tests that Round Robin behaves as expected, returning the next healthy server.
func TestRoundRobin(t *testing.T) {
	var err error

	// Initialize the pool
	pool, err = NewServerPool(serverAddrs)
	if err != nil {
		t.Error(err)
	}
	defer func() {
		pool.CancelHealthCheck()
		pool = nil
	}()

	// Test 1: When all servers are healthy
	pool.HealthyAll()

	for i := 0; i < len(pool.Servers); i++ {
		rrIdx, err := RoundRobin(pool)
		if err != nil {
			t.Error(err)
		}
		if rrIdx != i {
			t.Errorf("Expected RoundRobin to choose index %d but it chose %d", i, rrIdx)
		}
	}

	// Test 2: When server at index K is unhealthy
	k := 2
	if k >= len(pool.Servers) {
		log.Fatalf("Invalid test: index is server to be marked unhealthy is out of bounds")
	}
	pool.Servers[k].Degrade()
	for i := 0; i < len(pool.Servers); i++ {
		rrIdx, err := RoundRobin(pool)
		if err != nil {
			t.Error(err)
		}
		if rrIdx == k {
			t.Errorf("Expected RoundRobin to to never chose unhealthy server at index %d but it did", k)
		}
	}

}

// Functions to start/stop the target servers `go test`

func startTargetServers() (err error) {
	for _, p := range targetPorts {
		err = startTargetServer(p)
		if err != nil {
			return err
		}
		time.Sleep(2)
	}
	return nil
}

func stopTargetServers() {
	cmd := exec.Command("pkill", "-f", targetBinaryName)

	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func startTargetServer(port int) error {
	cmd := exec.Command(targetBinaryName, "server", "-p", fmt.Sprintf("%d", port))

	err := cmd.Start()
	if err != nil {
		return err
	}

	return nil
}
