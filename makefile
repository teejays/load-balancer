PROJECT_NAME = load-balancer

SYS_TYPE = darwin # change to linux for linux systems

GO = go
GO_FMT = $(GO) fmt
GO_BUILD = $(GO) build

SRC_DIR = 
BIN_DIR = bin
BINARY_PATH = $(BIN_DIR)/$(PROJECT_NAME)-$(SYS-TYPE)

TARGET_SERVER_BIN_PATH = $(BIN_DIR)/challenge-$(SYS_TYPE)


all: clear build run-dev

build: go-fmt
	$(GO_BUILD) -o $(BINARY_PATH) ./$(SRC_DIR)

run-dev: run-targets
	./$(BINARY_PATH) -b http://localhost:9000 -b http://localhost:9001 -b http://localhost:9002 -b http://localhost:9003 -b http://localhost:9004 -b http://localhost:9005 -b http://localhost:9006 -b http://localhost:9007 -b http://localhost:9008 -b http://localhost:9009

run-targets: kill-targets
	for port in 9000 9001 9002 9003 9004 9005 9006 9007 9008 9009 ; do \
		($(TARGET_SERVER_BIN_PATH) server -p $$port &) && sleep 2; \
	done

kill:
	-pkill -f $(BINARY_PATH) || true

kill-targets:
	-pkill -f $(TARGET_SERVER_BIN_PATH) || true

go-test:
	$(GO) test -v

clear:
	rm $(BIN_DIR)/* && clear

go-fmt:
	$(GO_FMT)


## Profiling rules

BINARY_PPROF_PATH = $(BIN_DIR)/$(PROJECT_NAME)-pprof

pprof: kill-all build-with-pprof run-with-pprof start-load-test get-profile stop-load-test

build-with-pprof:
	$(GO_BUILD) -tags pprof -o $(BINARY_PPROF_PATH) ./$(SRC_DIR)

run-with-pprof: run-targets
	./$(BINARY_PPROF_PATH) -b http://localhost:9000 -b http://localhost:9001 -b http://localhost:9002 -b http://localhost:9003 -b http://localhost:9004 -b http://localhost:9005 -b http://localhost:9006 -b http://localhost:9007 -b http://localhost:9008 -b http://localhost:9009 > out.log 2> err.log &

start-load-test: 
	./scripts/load-test.sh > out.log 2> err.log &

stop-load-test: 
	-pkill -f ./scripts/load-test.sh || true

PPROF_PORT = 6060
PPROF_SECS = 30
get-profile:
	$(GO) tool pprof -png http://localhost:$(PPROF_PORT)/debug/pprof/profile?seconds=$(PPROF_SECS)

kill-all: kill kill-targets stop-load-test
	
