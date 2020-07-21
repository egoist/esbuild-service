.PHONY: test run build

test:
	GO111MODULE=on go test ./...

run:
	GO111MODULE=on go run main.go

build:
	GO111MODULE=on go build .