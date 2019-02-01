// +build pprof

// main package code in this file will only be included when pprof build tag is passed.
// It implements the pprof logic.
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
