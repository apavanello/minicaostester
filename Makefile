.PHONY: build run clean

APP_NAME = chaos-target

build: test
	@go install golang.org/x/tools/cmd/deadcode@latest
	@go install github.com/mgechev/revive@latest
	@gofmt -s -w .
	@revive ./...
	@go mod tidy
	@deadcode ./...
	@go env -w CGO_ENABLED=0
	@go vet ./...
	@go install ./...
	@go env -u CGO_ENABLED
	@echo "Building $(APP_NAME)..."
	@go build -o build/$(APP_NAME) main.go

run: build
	@echo "Running $(APP_NAME)..."
	@./build/$(APP_NAME)

clean:
	@echo "Cleaning up..."
	@rm -f build/$(APP_NAME)
	@rm -f build/$(APP_NAME).exe

test:
	@go test -v ./...

cover:
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated at coverage.html"
