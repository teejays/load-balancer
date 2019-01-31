# Simple Load Balancer

_This project is done as a take-home assignment for a job application at a tech company. Please do not share this in a public forum._

### Introduction

This Golang executable implements a simple HTTP load balancer (LB). It is fully stateful, maintaining a frequently updated list of all the target servers and their states (whether the target serves are healthy or not). The target servers are all expected to have an health check endpoint ?\\\_health that informs on the state of that server. All the requests to the load balancer are forwarded to one of the healthy target servers, or a 503 is returned suggesting that no healthy server was available. In the rare case where a selected server becomes unhealthy before a request is forwarded to it, load balancer retries with another healthy server.


## Getting Started

The project comes with a makefile that simplifies the process of building, running and testing it.

#### Prerequisites

* **_Golang_**: You need have Golang (>= 1.10) installed on your system. You can install it by following the instructions from [Go official website](https://golang.org/).

* **_Dependencies_**: This package is a go module so it can automatically install the required dependencies. Currently, it only depends on a colored logging package (github.com/teejays/clog).

#### Build

The project can built using the command: ```make build```. The compiled binaries go into the ${project-root}/bin directory.

#### Run

**_Makefile_**: Use ```make run-dev```. Running the _make_ command starts nine target servers with ports starting from 9000 to 9009. It also starts the load-balancer by passing it the addresses for all the target severs running.

**_Without Makefile_**: If you want to run the application manually, you can do it using: ```./bin/load-balancer -p <port> -b <server 1> -b <server 2>  ```. You should replace <port> with the port number you want load-balancer to listen on, and <server _n_> with the address of each of the target servers that you are running.

The application accepts two different kinds of parameters:

* **_-p_** : port at which the run the listener server
* **_-b_** : address for each of the backend target servers

#### Testing

The project tests are written using Go's standard _testing_ package. They can be run using ```make go-test ```

## Description

The project code has three main components/entities:
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
