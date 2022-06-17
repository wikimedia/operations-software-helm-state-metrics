MAKEFLAGS += --warn-undefined-variables
SHELL := bash
.SHELLFLAGS := -eu -o pipefail -c
.DELETE_ON_ERROR:
.SUFFIXES:
.ONESHELL:

# The version which will be reported by the --version argument
VERSION ?= $(shell git describe --tags 2>/dev/null || echo development)
VERSION := $(VERSION)

# BIN is the directory where tools will be installed
export BIN ?= ${CURDIR}/bin

OS := $(shell go env GOOS)
ARCH := $(shell go env GOARCH)

all: helm-state-metrics

.PHONY: vendor
vendor:
	go mod tidy
	go mod vendor

# Run tests
.PHONY: test
test: fmt vet
	go test ./... -coverprofile cover.out
	go tool cover -html=cover.out -o cover.html

# Build binary
helm-state-metrics: fmt vet vendor
	go build \
		-ldflags="-X=main.Version=${VERSION}" \
		-o bin/helm-state-metrics

# Run against the configured Kubernetes cluster in ~/.kube/config
.PHONY: run
run: fmt vet
	go run ./main.go

# Run go fmt against code
.PHONY: fmt
fmt:
	go fmt ./...

# Run go vet against code
.PHONY: vet
vet:
	go vet ./...
