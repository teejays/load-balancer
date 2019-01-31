// package main implements a sample load balancer in Golang. The program
// accepts two different kinds of parameters:
// -p: port at which the run the listener server
// -b: address for backend servers
//
// The application does five main things:
// 1. Parse the command line arguments to get a list of all the backend server addresses
// 2. Create a pool of those servers
// 3. Start a secondary goroutine to periodically check the health status of each server
// 4. Implement a simple Round Robin based algorithm to get a healthy server from the pool
// 5. Start a websever that listens to requests, and forwards them to one of the backend servers
//
// The application has three main entiies:
// 1. ServerAddresses []string: It implements the flag.Var interface, and allows capturing multiple -b flags
// 2. BackendServer struct: It represents a backend server, with functions implemented for checking and updating it's health status
// 3. ServerPool struct: This holds all the (healthy or degraded) backend servers in a pool, and allows picking of server for forwarding requests.
//
// Reverse Proxy: All the incoming requests have their http.Request instance changed
// and are forwarded to a backend server. The response is copied over into the response for
// the original request.
package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"github.com/teejays/clog"
)

const (
	listenerPortDeault  int           = 8888 // 8888 is the default, but it'll be overwritten is a flag is passed.
	listenerReadTimeout time.Duration = 10 * time.Second
)

var pool *ServerPool // singleton patter
var proxy *httputil.ReverseProxy

func main() {
	var err error

	// Process the flags
	var listenerPort int
	var serverAddrs ServerAddresses
	flag.IntVar(&listenerPort, "p", listenerPortDeault, "The port at which the load balancer server will listen.")
	flag.Var(&serverAddrs, "b", "One of more backend server addresses")
	flag.Parse()

	clog.Infof("Flags succesfully parsed: port=%d, addresses=%s", listenerPort, serverAddrs)

	// Initialize the pool of backend servers
	clog.Info("Creating a new load balancer server pool...")
	pool, err = NewServerPool(serverAddrs)
	if err != nil {
		clog.FatalErr(err)
	}
	clog.Infof("Load balancer server pool created.")

	// Run the listener server
	err = startServer(listenerPort)
	if err != nil {
		clog.FatalErr(err)
	}
}

func startServer(port int) error {
	// Create a http.Server instance
	server := &http.Server{
		Addr:        fmt.Sprintf(":%d", port),
		ReadTimeout: listenerReadTimeout,
		Handler:     http.HandlerFunc(handlerV2),
	}
	clog.Infof("Staring the server: %d", port)
	return server.ListenAndServe()
}

func handlerV2(w http.ResponseWriter, req *http.Request) {
	// Get a server from pool to forward the request
	server, err := pool.GetServer(RoundRobin)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	// Update the http.Request instance to point to backend server
	redirectRequestToServer(req, server)

	// Make a request to backend server
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()

	// Special case: if resp.StatusCode is 500, that means the server is in degrade status.
	// In this case, as suggested by the question prompt, we should redirect the request to
	// use a different server.
	if resp.StatusCode == http.StatusInternalServerError {
		// This means the server is down! Degrade and try again
		server.Degrade()
		handlerV2(w, req)
		return
	}

	// In a normal case, copy the response into the response for the original request
	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func redirectRequestToServer(req *http.Request, server *BackendServer) {

	target := server.URL
	targetQuery := target.RawQuery
	req.URL.Scheme = target.Scheme
	req.URL.Host = target.Host
	req.URL.Path = singleJoiningSlash(target.Path, req.URL.Path)
	if targetQuery == "" || req.URL.RawQuery == "" {
		req.URL.RawQuery = targetQuery + req.URL.RawQuery
	} else {
		req.URL.RawQuery = targetQuery + "&" + req.URL.RawQuery
	}
	if _, ok := req.Header["User-Agent"]; !ok {
		// explicitly disable User-Agent so it's not set to default value
		req.Header.Set("User-Agent", "")
	}
}

func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}
