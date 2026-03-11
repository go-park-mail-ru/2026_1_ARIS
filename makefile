.PHONY: test coverage clean

test:
	go test -v ./...

clean:
	if exist coverage.out del /f coverage.out

coverage: clean
	go test -coverprofile=coverage.out -coverpkg=./internal/... ./...
	go tool cover -html=coverage.out