package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"testing"
	"time"
)

var targetPorts = []int{9000, 9001, 9002, 9003, 9004, 9005}
var runningTargetPipes []io.WriteCloser
var serverAddrs ServerAddresses

func init() {
	// Make the interval smaller for testing
	HealthCheckInterval = time.Second * 2

	// Initialize the ServerAddress instance, just like if someone has passed all these args
	for _, p := range targetPorts {
		serverAddrs = append(serverAddrs, fmt.Sprintf("http://localhost:%d", p))
	}
}

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
	pool.Delete()
}

func TestNoHealthyServer(t *testing.T) {
	var err error

	// Initialize the pool
	pool, err = NewServerPool(serverAddrs)
	if err != nil {
		t.Error(err)
	}
	defer pool.Delete()

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

// Todo: Write test for the case when the state of a healthy server changes after it has been picked up
// in the round robin. It is difficult to deterministically create this scenario

func TestRoundRobin(t *testing.T) {
	var err error

	// Initialize the pool
	pool, err = NewServerPool(serverAddrs)
	if err != nil {
		t.Error(err)
	}
	defer pool.Delete()

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

// Functions to start the target servers

func startTargetServers() (err error) {
	for _, p := range targetPorts {
		err = startTargetServer(p)
		if err != nil {
			return err
		}
	}
	return nil
}

func stopTargetServers() {
	// Stop the servers
	for _, stdin := range runningTargetPipes {
		//log.Print("Closing target server")
		stdin.Close()
	}
}

func startTargetServer(port int) error {
	cmd := exec.Command(targetBinaryName, "-p", fmt.Sprintf("%d", port))

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	err = cmd.Start()
	if err != nil {
		return err
	}

	runningTargetPipes = append(runningTargetPipes, stdin)
	//log.Print(fmt.Sprintf("Starting target server at %d", port))

	return nil
}
