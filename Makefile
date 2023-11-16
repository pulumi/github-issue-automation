default: build

.PHONY: build
build: .build/new-release-handler.zip pulumi/build

.build/new-release-handler.zip: .build/new-release-handler
	cd .build && zip new-release-handler.zip new-release-handler

.build/new-release-handler: lambda/new-release-handler/*
	cd lambda/new-release-handler && GOOS=linux GOARCH=amd64 go build -o ../../.build/new-release-handler main.go

lambda/new-release-handler/go.sum: lambda/new-release-handler/go.mod
	cd lambda/new-release-handler && go mod tidy

.PHONY: clean
clean:
	rm .build/*

# Intended for local deployment only
.PHONY: deploy-dev
deploy-dev: .build/new-release-handler.zip pulumi/*
	cd pulumi && pulumi up -s dev

.PHONY: pulumi/build
pulumi/build:
	cd pulumi && go build -o ../.build/pulumi-program

.PHONY: test
test: lambda/new-release-handler/go.sum
	cd lambda/new-release-handler && go test ./...

.PHONY: refresh
refresh:
	cd pulumi && pulumi refresh -s dev
