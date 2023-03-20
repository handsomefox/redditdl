build:
	go build --ldflags "-s -w" .
build_win:
	GOOS=windows go build --ldflags "-s -w" .
build_linux:
	GOOS=linux go build --ldflags "-s -w" .
test:
	go test -race -v -count=1 ./...
bench:
	go test -race -v ./... -bench=. -run=xxx -benchmem
tidy:
	go mod tidy
vendor:
	go mod vendor
