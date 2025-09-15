.PHONY: build build-linux clean

build:
	go build -o highLoadTest main.go

build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -ldflags '-extldflags "-static"' -o highLoadTest main.go

clean:
	rm -f highLoadTest