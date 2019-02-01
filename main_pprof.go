// +build pprof

package main

import (
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
)

var profilePort int = 6060

func init() {
	go func() {
		profilerAddress := fmt.Sprintf(":%d", profilePort)
		log.Printf("Starting profiler on: %s", profilerAddress)
		log.Println(http.ListenAndServe(profilerAddress, nil))
	}()
}
