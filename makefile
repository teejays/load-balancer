PROJECT_NAME = load-balancer

GO = go
GO_BUILD = $(GO) build

SRC_DIR = src
BIN_DIR = bin
BINARY_PATH = $(BIN_DIR)/$(PROJECT_NAME)

BACKEND_SERVER_BIN_PATH = $(BIN_DIR)/challenge-darwin

all: clear build run-dev

build:
	$(GO_BUILD) -o $(BINARY_PATH) ./$(SRC_DIR)

run-dev: run-dev-backend-servers
	./$(BINARY_PATH) -b http://localhost:9000 -b http://localhost:9001

run-dev-backend-servers:
	$(BACKEND_SERVER_BIN_PATH) server -p 9000 &

clear:
	clear
