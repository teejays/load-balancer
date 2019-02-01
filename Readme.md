# Simple Load Balancer

_This project is attempted as a take-home assignment for a job application. Please do not share this in a public forum. This is a part of a private Github repo._

### Introduction

This Golang application implements a simple HTTP load balancer (LB). It is fully stateful, maintaining a frequently updated list of all the target servers and their states (whether the target serves are healthy or degraded). The target servers are all expected to have a health check endpoint ?\\\_health that informs on the state of that server. All the requests to the load balancer are forwarded to one of the healthy target servers, or a 503 is returned suggesting that no healthy server was available. In the rare case where a selected server becomes unhealthy before it can handle a request, load balancer retries by choosing a different healthy server.

## Getting Started

The project comes with a makefile that simplifies the process of building, running and testing it. 

**_Note:_** Although the project is designed to be built and run on both linux and darwin systems, it was primarily developed and tested on a darwin system. 

#### Prerequisites

* **_Golang_**: You need have Golang (>= 1.10) installed on your system. You can install it by following the instructions from [Go official website](https://golang.org/).

* **_Dependencies_**: This package is a go module so it can automatically install the required dependencies. Currently, it only depends on an excellent colored logging package, [Clog](https://github.com/teejays/clog), written by Talha Ansari (wow, that's me).

#### Build

The project can be built using the command: ```make build```. The compiled binaries go into the ${project-root}/bin directory.

#### Run

**_Without Makefile_**: You should be using the makefile in most circumstances but if you want to run the application manually, you can do it using:
 ```./bin/load-balancer -p <port> -b <server 1> -b <server 2>  ```
You should replace <port> with the port number you want load-balancer to listen on, and <server _n_> with the address of each of the target servers that you are running.

The application accepts two different kinds of parameters:

* **_-p_** : port at which the run the listener server
* **_-b_** : address for each of the backend target servers

**_Using Makefile_**: Use ```make run-dev```. Running this _make_ command downloads the target server binaries from Google Drive (if they haven't already been downloaded), start nine target servers with ports starting from 9000 to 9009. It also starts the load-balancer by passing it the addresses for all the target servers.

Once you've successfully run ```make run-dev```, the load balancer is on and running. You will be able to see its output in stdout. 

Use ```make kill``` to stop any running processes.

#### Testing

**_Golang's Testing package:_** The project tests are written using Go's standard _testing_ package. They can be run using ```make go-test ```. The benchmark can be run using ```make benchmark```.

**_Load Test:_** There is a bash script that simulates load by calling the load balancer sequentially. You can run it by calling ```make start-loadtest```. You can turn it off by calling ```make kill-loadtest```.

**_Profiling:_** The application is also set up for easy system profiling. Running ```make pprof``` compiles a _pprof_ ready version of the program, which is then put under high load. A 30sec CPU profile is then generated (more about it, and sample profile later).

_Note:_ ```make pprof``` will only work if you have graphviz installed.


## Description

This project code has three main components/entities:

1. **_ServerAddresses_** []string: This implements the flag.Var interface, and therefore allows capturing multiple -b flags from the command line.

2. **_TargetServer_** struct: This represents one target server, with struct fields to keep track of the health status of that server and various struct methods implemented, including for checking and updating the health status.

3. **_ServerPool_** struct: Holds all the (healthy and degraded) target servers in an array, and allows selection of a healthy server using Round Robin for forwarding the http requests. Whenever a new ServerPool is created, a secondary go-routine is setup that, after every set interval, goes through all the TargetServer in the pool and updates their health status. For the purpose of this load balancer, ServerPool implements a **singleton pattern**, where we have only one instance of it, called _pool_.

**_Initialization:_** Upon initialization, load balancer parses the command line arguments to get the port and all the target server addresses. It uses the target server addresses to create an instance of type ServerPool, _pool_. This is also starts a goroutine to periodically check the health status of the target servers.

Eventually, the load balancer starts it's own server to listen for requests. The listener server has a handler that implements the logic of load-balancing, and redirects the request to appropriate target servers.

**_Handling Request:_** When a HTTP request is made to the load balancer, the listener server accepts the requests and forwards it to the HTTP handler. The handler uses a Round Robin type algorithm to get a healthy target server from the _pool_. If there is no healthy server, it returns a 503. If there is a healthy server available, it redirects the request to the healthy target server by making use of Go's http.DefaultTransport. If the target server returns a 500, it marks that server as degraded and retries by selecting a newer server.


## Discussion

**_Server Selection Algorithms:_** I decided to implement a simple Round Robin algorithm because of its simplicity and popularity. However, the code is designed to allow for other and more complicated algorithms e.g. least connection, least response time, least bandwidth etc to be easily incorporated. More fields could be added to the TargetServer type to hold info necessary to implement such algorithms e.g. we can store the number of active connections for a target server and implement least connection algorithm.


**_Round Robin Implementation:_** Current round robin implementation is very naive, where we basically store the state (healthy vs degraded) of each target server in its instance. During a round robin run, we loop through all the servers in the pool and return the first healthy server. We keep track of the index for the next round robin, so the system knows where to start. If the loop goes through all the servers without finding a healthy server, we return 503.

This implementation can potentially be optimized if I can categorize healthy and degraded servers in separate arrays, removing the need for looping through degraded servers. However, maintaining and keeping those arrays up to date will have its own costs e.g. healthy server list being changed/updated while a request is made concurrently, in which case we might to have to implement locks that can hold the system regularly.

**_Proxy_**: There were a few different ways to implement this. Golang's httputil.ReverseProxy does implement a solution for this but using that makes it difficult to implement any logic between getting a response from the target server and returning the response to the client. Hence, I end up changing the http.Request manually to redirect it to the target server and then making a http.Transport roundtrip call to the target server to get the response. The response is then directly copied into the response for the original request.


**_Testing_**: Ideally, we would  want to have tests that can mimic the desired scenario in a deterministic way. For example, I wanted to write a test that can mimic the case where a healthy server returns a 500. Although such tests could be implemented if I write a mock target server handler that I can control, but I felt that was perhaps beyond the scope of the project. If I had more time, I would probably do more intensive testing. I implemented a load test using a simple bash script. Since this is a load-balancer, I felt it made sense to see how it performs under load.

I was able to add a simple benchmarking function to test requests to the load balancer server.
![Benchmark Output](https://i.imgur.com/EjMtChV.png)

**_Profiling_**: A 30sec CPU profile for the load-balancer under moderate load looks okay. It seems like that there are no clear bottlenecks in the code. The diagram is shown below and can also be accessed [here](https://imgur.com/a/Qadz6ZD).

![30sec CPU Profile](https://i.imgur.com/9gRIVt7.png)

It will be interesting to see how the graph looks like extremely high load. If I had a little more time, I would've set up a testing profile in JMeter to better load test.

## Contact
For any issues, please feel free to reach out to me at hello@talha.io.
