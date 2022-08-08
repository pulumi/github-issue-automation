default: .build/new-release-handler

.build/new-release-handler: lambda/*
	cd lambda/new-release-handler && GOOS=linux GOARCH=amd64 go build -o ../../.build/new-release-handler main.go

lambda/new-release-handler/go.sum: lambda/new-release-handler/go.mod
	cd lambda/new-release-handler && go mod tidy

# Intended for local deployment only
.PHONY: deploy
deploy: .build/new-release-handler pulumi/*
	cd pulumi && pulumi up -s dev

.PHONY: test
test: lambda/new-release-handler/go.sum
	cd lambda/new-release-handler && go test

.PHONY: refresh
refresh:
	cd pulumi && pulumi refresh