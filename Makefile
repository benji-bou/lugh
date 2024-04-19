# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build 
GOBUILD_DEBUG=$(GOCMD) build -gcflags="all=-N -l"
GOMOD=$(GOCMD) mod

# Protobuf parameters
PROTOC=protoc
PROTO_DIR=./pluginctl
PROTO_SRC=$(PROTO_DIR)/plugins.proto

# Directories
CMD_DIR=./cmd/cli
PLUGINS_DIR=./plugins
BIN_DIR=./bin
PLUGIN_BIN_DIR=$(BIN_DIR)/plugins

# Program names
CMD_OUTPUT=$(BIN_DIR)/cli
# PLUGINS_GO_SOURCES := $(shell find $(PLUGINS_DIR) -type f -name '*.go' -exec grep -l 'package main' {} \;)
# PLUGINS_OUTPUT := $(patsubst $(PLUGINS_DIR)/%,$(PLUGIN_BIN_DIR)/%,$(dir $(PLUGINS_GO_SOURCES)))
# Targets
.PHONY: all build clean

all: plugins_pb build build_plugins

build: 
	@echo "Building main CLI..."
	@$(GOBUILD_DEBUG) -o $(CMD_OUTPUT) $(CMD_DIR)/main.go

build_plugins: 
	@echo "Building plugin..."
	@mkdir -p $(BIN_DIR)/plugins
		@find $(PLUGINS_DIR) -type f -name '*.go' -exec sh -c 'if grep -q "^package main$$" "{}"; then dir=$$(dirname {}); output=$$(basename $$dir); $(GOBUILD_DEBUG) -o $(BIN_DIR)/plugins/$$output {}; fi' \;

plugins_pb:
	@echo "Compiling protobuf files..."
	@$(PROTOC) --go_out=./ --go_opt=paths=source_relative \
	    --go-grpc_out=./ --go-grpc_opt=paths=source_relative $(PROTO_SRC)

clean:
	@echo "Cleaning..."
	@rm -rf $(BIN_DIR)
