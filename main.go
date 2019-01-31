// main package implements a sample load balancer in Golang. The program
// accepts two different kinds of parameters:
// -p: port at which the run the listener server
// -b: address for backend servers
//
// The application has three main components:
// 1. ServerAddresses []string: It implements the flag.Var interface, and allows
//    capturing multiple -b flags from the command line
// 2. TargetServer struct: It represents a target server, with fields to keep track of the health
//    and functions implemented for checking and updating the health status
// 3. ServerPool struct: Holds all the (healthy or degraded) backend servers in an array, and allows
//    picking of healthy server for forwarding the http requests.
//
// When you start the application, it does five main things:
// 1. Parse the command line arguments to get ServerAddresses
// 2. Create a ServerPool from the ServerAddresses instance, in the process creating a TargetServer
//    instance for each of the server address
// 3. Start a goroutine to periodically check the health status of each TargetServer
// 4. Start a listener webserver on the port specified (or default 8888) that listens for requests and
//    proxies them to the target servers
//
// When you make a http request to the load balancer, the following logic takes place:
// 1. Listener webserver accepts the request
// 2. It uses a Round Robin type algorithm to get a healthy target server from the pool. If
//    no healthy server, return error.
// 3. Make a request to the healthy target server. If status code is 500, repeat from 1.
//    To-do: Implement a limit on how many retries on a 500 response.
// 4. Copy the response from the target server to the resonse for the client http request.
//
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
	"strings"
	"time"

	"github.com/teejays/clog"
)

const (
	// listenerPostDefault is the port that is used by listener webserver when a port is not explicitly specified in the command line.
	listenerPortDeault int = 8888

	// listenerReadTimeout is the listener server timeout for reading the request.
	listenerReadTimeout time.Duration = 10 * time.Second
)

// pool is the singleton pattern instance of ServerPool. This holds all our target servers, and is the main
// load balancer entity.
var pool *ServerPool

func main() {
	var err error

	// Step 1: Process the flags
	var listenerPort int
	var serverAddrs ServerAddresses
	flag.IntVar(&listenerPort, "p", listenerPortDeault, "The port at which the load balancer server will listen.")
	flag.Var(&serverAddrs, "b", "One of more target server addresses")
	flag.Parse()
	clog.Infof("Flags succesfully parsed: port=%d, addresses=%s", listenerPort, serverAddrs)

	// Step 2: Initialize the pool of target servers
	clog.Info("Creating a new load balancer server pool...")
	pool, err = NewServerPool(serverAddrs)
	if err != nil {
		clog.FatalErr(err)
	}
	clog.Infof("Load balancer server pool created.")

	// Step 3: Run the listener server
	err = startListener(listenerPort)
	if err != nil {
		clog.FatalErr(err)
	}
}

// startListener starts a webserver that listens on the localhost at the provided port. The
// function call is blocking as it only returns if there is an error while starting the server.
func startListener(port int) error {

	// Create a http.Server instance & start it
	server := &http.Server{
		Addr:        fmt.Sprintf(":%d", port),
		ReadTimeout: listenerReadTimeout,
		Handler:     http.HandlerFunc(listenerHandler),
	}
	clog.Infof("Staring the server: %d", port)
	return server.ListenAndServe()
}

// listenerHandler handles all the http requests to listenere server. It implements the logic for
// load-balancing, where it finds a healthy target server from the pool, forwards the request to it, and
// copies over its response to the response for the client request.
func listenerHandler(w http.ResponseWriter, req *http.Request) {

	// Get a healthy target server from pool so we can forward the request to it
	target, err := pool.GetTargetServer(RoundRobin)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	clog.Debug("Forwarding request to the target server...")

	proxyRequestToTarget(w, req, target)

}

// proxyRequestToTarget reverse proxy a request to the target server, handling the case where
// the target server becomes unhealthy by the time the request is made.
func proxyRequestToTarget(w http.ResponseWriter, req *http.Request, target *TargetServer) {

	// Make changes to the http.Request instance so we can point it to the target server
	redirectRequestToServer(req, target)

	// Make a request to target server
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
		clog.Warning("The target server returned a 500, which means it is unhealthy...")
		target.Degrade()
		listenerHandler(w, req)
		return
	}

	// In a normal case, copy the response into the response for the original request
	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

// copyHeader copies all the http headers from src to dest
func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

// redirectRequestToServer modifies a request so it can be redirected to the target server.
// The logic here has been inspired from Go's official net/http/httputil package.
func redirectRequestToServer(req *http.Request, server *TargetServer) {

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

// singleJoiningSlash is a util function for redirectRequestToServer function. It is copied from
// Go's official net/http/httputil package.
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
