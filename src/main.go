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

	var listenerPort int
	var serverAddrs ServerAddresses
	flag.IntVar(&listenerPort, "p", listenerPortDeault, "The port at which the load balancer server will listen.")
	flag.Var(&serverAddrs, "b", "One of more backend server addresses")
	flag.Parse()

	clog.Infof("Flags succesfully parsed: port=%d, addresses=%s", listenerPort, serverAddrs)

	// Initialize the pool of servers
	clog.Info("Creating a new load balancer server pool...")
	pool, err = NewServerPool(serverAddrs)
	if err != nil {
		clog.FatalErr(err)
	}
	clog.Infof("Load balancer server pool created.")

	// Initialize the http Proxy
	// clog.Infof("Initializing the reverse proxy...")
	// proxy = NewLoadBalancerReverseProxy(pool)

	// Set up
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
	server, err := pool.GetServer()
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

// proxyHandler handles all the requests to the server, and forwards them to the
// func loadbalancerHandlerV1(w http.ResponseWriter, r *http.Request) {
// 	proxy.ServeHTTP(w, r)
// }

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

// func NewLoadBalancerReverseProxy(pool *ServerPool) *httputil.ReverseProxy {

// 	director := func(req *http.Request) {
// 		// Get a new server
// 		server, err := pool.GetServer()
// 		if err != nil {
// 			clog.Errorf("%v", err)
// 			return
// 		}
// 		target := server.URL
// 		targetQuery := target.RawQuery
// 		req.URL.Scheme = target.Scheme
// 		req.URL.Host = target.Host
// 		req.URL.Path = singleJoiningSlash(target.Path, req.URL.Path)
// 		if targetQuery == "" || req.URL.RawQuery == "" {
// 			req.URL.RawQuery = targetQuery + req.URL.RawQuery
// 		} else {
// 			req.URL.RawQuery = targetQuery + "&" + req.URL.RawQuery
// 		}
// 		if _, ok := req.Header["User-Agent"]; !ok {
// 			// explicitly disable User-Agent so it's not set to default value
// 			req.Header.Set("User-Agent", "")
// 		}
// 	}
// 	return &httputil.ReverseProxy{Director: director}
// }

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
