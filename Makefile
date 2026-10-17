.PHONY: build test vet tidy
build:
	go build -o fontctl ./cmd/fontctl
test:
	go test ./...
vet:
	go vet ./...
tidy:
	go mod tidy
