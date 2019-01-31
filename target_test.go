package main

// import (
// 	"testing"
// )

// var testListenerPort int = 7474

// var mockTargetServers = []TargetServer

// func TestMockServer(t *testing.T) {

// 	port = 9000
// 	target, err := NewTargetServer("http://localhost:9000")
// 	if err != nil {
// 		t.Error(err)
// 	}

// 	err = startListener(testListenerPort)
// 	if err != nil {
// 		t.Errorf("Error starting listener server: %s", err)
// 	}

// }

// func startMockTestServer(port int) error {

// 	http.Handle("/_health", mockTargetServerHealthHandler)

// 	// Create a http.Server instance & start it
// 	server := &http.Server{
// 		Addr:        fmt.Sprintf(":%d", port),
// 		ReadTimeout: listenerReadTimeout,
// 		Handler:     http.HandlerFunc(listenerHandler),
// 	}
// 	clog.Infof("Staring the server: %d", port)
// 	return server.ListenAndServe(fmt.Sprintf("%d", testListenerPort), mockTargetServerHandler)
// }

// func mockTargetServerHandler(w http.ResponseWriter, r *http.Request) {
// 	fmt.Fprintf(w, "hello, you've hit %s\n", r.URL.Path)
// }

// func mockTargetServerHealthHandler(w http.ResponseWriter, r *http.Request) {
// 	var resp HealthResponse

// 	rand.Seed(time.Now().UTC().UnixNano())

// 	if rand.Intn(2) < 1 {
// 		resp.State = "degraded"
// 	} else {
// 		resp.State = "healthy"
// 	}
// 	body, err := json.Marshal(resp)
// 	if err != nil {
// 		http.Error(w, err.Error, http.StatusInternalServerError)
// 		return
// 	}
// }
