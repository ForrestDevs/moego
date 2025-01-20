test:
	go test ./... -v

install:
	go get ./... && go mod vendor && go mod tidy && go mod download