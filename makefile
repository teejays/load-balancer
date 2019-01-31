PROJECT_NAME = "load-balancer"

GO = go
GO_BUILD = $(GO) build

BUILD_DIR = "_build"
BINARY_PATH = $(BUILD_DIR)/$(PROJECT_NAME)

all: clear build run-dev

build:
	$(GO_BUILD) -o $(BINARY_PATH)

run-dev:
	./$(BINARY_PATH) -b http://localhost:9000

clear:
	clear
