default: .build/new-release-handler.zip .build/internal-release-handler.zip

.build/internal-release-handler.zip: .build/internal-release-handler
	cd .build && zip internal-release-handler.zip internal-release-handler

.build/internal-release-handler: lambda/internal-release-handler/*
	cd lambda/internal-release-handler && GOOS=linux GOARCH=amd64 go build -o ../../.build/internal-release-handler main.go

lambda/internal-release-handler/go.sum: lambda/internal-release-handler/go.mod
	cd lambda/internal-release-handler && go mod tidy

.build/new-release-handler.zip: .build/new-release-handler
	cd .build && zip new-release-handler.zip new-release-handler

.build/new-release-handler: lambda/new-release-handler/*
	cd lambda/new-release-handler && GOOS=linux GOARCH=amd64 go build -o ../../.build/new-release-handler main.go

lambda/new-release-handler/go.sum: lambda/new-release-handler/go.mod
	cd lambda/new-release-handler && go mod tidy

# Intended for local deployment only
.PHONY: deploy
deploy: .build/new-release-handler.zip .build/internal-release-handler.zip pulumi/*
	cd pulumi && pulumi up -s dev

.PHONY: test
test: lambda/new-release-handler/go.sum lambda/internal-release-handler/go.sum
	cd lambda/new-release-handler && go test && cd ../internal-release-handler && go test

.PHONY: refresh
refresh:
	cd pulumi && pulumi refresh