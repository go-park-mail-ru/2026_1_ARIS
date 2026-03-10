.PHONY: test coverage clean

test:
	go test -v ./...

clean:
	rm -f coverage.out

coverage: clean
	go test -coverprofile=coverage.out -coverpkg=./internal/... ./...
	go tool cover -html=coverage.out