build:
	go build --ldflags "-s -w" .
test:
	go test -v ./...
bench:
	go test ./... -bench=. -run=xxx -benchmem
tidy:
	go mod tidy
vendor:
	go mod vendor
