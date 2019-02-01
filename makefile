PROJECT_NAME = load-balancer

# Detected System type
DARWIN=darwin
LINUX=linux
ifeq ($(shell uname),Darwin)
SYS_TYPE=$(DARWIN)
endif
ifeq ($(shell uname),Linux)
SYS_TYPE=$(LINUX)
endif

# Go commands
GO = go
GO_FMT = $(GO) fmt
GO_BUILD = $(GO) build

# Directory vars
SRC_DIR = 
BIN_DIR =bin
OUT_DIR =_out
SCRIPTS_DIR=scripts
BINARY_PATH = $(BIN_DIR)/$(PROJECT_NAME)-$(SYS_TYPE)

setup-bin:
	@mkdir -p $(BIN_DIR)

build: go-fmt setup-bin
	@echo "build: Compiling the load balancer"; \
	$(GO_BUILD) -o $(BINARY_PATH) ./$(SRC_DIR) ; \
	echo "build: Build successful: $(BINARY_PATH)"; \

run-dev: kill-lb start-targets
	@echo "run-dev: Starting the load balancer server on default port..."; \
	./$(BINARY_PATH) -b http://localhost:9000 -b http://localhost:9001 -b http://localhost:9002 -b http://localhost:9003 -b http://localhost:9004 -b http://localhost:9005 -b http://localhost:9006 -b http://localhost:9007 -b http://localhost:9008 -b http://localhost:9009

kill-lb:
	@echo "kill-lb: Stopping load balancer (if any running)..."; \
	pkill -f $(BINARY_PATH) || true


# Target Servers
TARGET_SERVER_BIN_PATH = $(BIN_DIR)/challenge-$(SYS_TYPE)
GOOGLE_DRIVE_ID_DARWIN=1nRF5ZfdT9-AEh1kKwJiRygd0aOquh_K3
GOOGLE_DRIVE_ID_LINUX=1Czv-3lgoIrc2fOZi2lj6oRAO3EYT321b

ifeq ($(SYS_TYPE),$(DARWIN))
TARGET_BINARY_GOOGLE_FILE_ID=$(GOOGLE_DRIVE_ID_DARWIN)
endif

ifeq ($(SYS_TYPE),$(LINUX))
TARGET_BINARY_GOOGLE_FILE_ID=$(GOOGLE_DRIVE_ID_LINUX)
endif 

get-target-binary: setup-bin
	@echo "get-target-binary: Downloading target server binary from Google Drive..." ; \
	./$(SCRIPTS_DIR)/get-challenge-binary.sh $(TARGET_BINARY_GOOGLE_FILE_ID) $(TARGET_SERVER_BIN_PATH)

start-targets: kill-targets get-target-binary
	@echo "start-targets: Starting Target Servers. This may take a while as we sleep for a bit after each server..." ; \
	for port in 9000 9001 9002 9003 9004 9005 9006 9007 9008 9009 ; do \
		($(TARGET_SERVER_BIN_PATH) server -p $$port &) && sleep 2; \
	done; \
	echo "Target Servers successfuly started." ; \

kill-targets:
	@echo "kill-targets: Stopping all existing Target Servers (if any running)..."; \
	pkill -f $(TARGET_SERVER_BIN_PATH) || true

# Others
go-test: get-target-binary
	$(GO) test -v

benchmark: get-target-binary
	$(GO) test -v -bench=. -benchtime=20s

clean:
	rm $(BIN_DIR)/*

go-fmt:
	@$(GO_FMT)


## Profiling rules & Load Testing

BINARY_PPROF_PATH = $(BIN_DIR)/$(PROJECT_NAME)-pprof

pprof: kill build-with-pprof run-with-pprof start-loadtest get-profile kill-loadtest kill-lb-with-pprof
	@echo "pprof: Completed."

build-with-pprof:
	@echo "build-with-pprof: Compiling the application with pprof enabled"; \
	$(GO_BUILD) -tags pprof -o $(BINARY_PPROF_PATH) ./$(SRC_DIR)

run-with-pprof: start-targets
	@echo "run-with-pprof: Running the pprof-enabled binary in the background"; \
	./$(BINARY_PPROF_PATH) -b http://localhost:9000 -b http://localhost:9001 -b http://localhost:9002 -b http://localhost:9003 -b http://localhost:9004 -b http://localhost:9005 -b http://localhost:9006 -b http://localhost:9007 -b http://localhost:9008 -b http://localhost:9009 > $(OUT_DIR)/out.log 2> $(OUT_DIR)/err.log &

kill-lb-with-pprof:
	@echo "kill-lb-with-pprof: Stopping pprof-enabled load balancer (if any running)..."; \
	pkill -f $(BINARY_PPROF_PATH) || true

start-loadtest: 
	@echo "start-loadtest: Starting the load test..."; \
	./scripts/load-test.sh > $(OUT_DIR)/out.log 2> $(OUT_DIR)/err.log &

kill-loadtest: 
	@echo "kill-loadtest: Stopping the load test (if any running)..."; \
	pkill -f ./scripts/load-test.sh || true

PPROF_PORT = 6060
PPROF_SECS = 30
get-profile:
	@echo "get-profile: Fetching the pprof CPU $(PPROF_SECS)secs profile..."; \
	$(GO) tool pprof -png -output $(OUT_DIR)/profile_cpu_$(PPROF_SECS)secs.png http://localhost:$(PPROF_PORT)/debug/pprof/profile?seconds=$(PPROF_SECS)

kill: kill-lb kill-lb-with-pprof kill-targets kill-loadtest
	
