PROJECT_NAME = load-balancer

GO = go
GO_BUILD = $(GO) build

SRC_DIR = 
BIN_DIR = bin
BINARY_PATH = $(BIN_DIR)/$(PROJECT_NAME)

BACKEND_SERVER_BIN_PATH = $(BIN_DIR)/challenge-darwin

all: clear build run-dev

build:
	$(GO_BUILD) -o $(BINARY_PATH) ./$(SRC_DIR)

run-dev: run-dev-backend-servers
	./$(BINARY_PATH) -b http://localhost:9000 -b http://localhost:9001 -b http://localhost:9002 -b http://localhost:9003 -b http://localhost:9004 -b http://localhost:9005 -b http://localhost:9006 -b http://localhost:9007 -b http://localhost:9008 -b http://localhost:9009

run-dev-backend-servers:
	for port in 9000 9001 9002 9003 9004 9005 9006 9007 9008 9009 ; do \
		($(BACKEND_SERVER_BIN_PATH) server -p $$port &) ; \
	done

clear:
	clear
