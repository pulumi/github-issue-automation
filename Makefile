default: .build/new-release-handler

.build/new-release-handler: lambda/*
	cd lambda && GOOS=linux GOARCH=amd64 go build -o ../.build/new-release-handler main.go

lambda/go.sum: lambda/go.mod
	cd lambda && go mod tidy

# Intended for local deployment only
.PHONY: deploy
deploy: .build/new-release-handler pulumi/*
	cd pulumi && pulumi up -s dev

.PHONY: test
test: lambda/go.sum
	cd lambda && go test

.PHONY: refresh
refresh:
	cd pulumi && pulumi refresh