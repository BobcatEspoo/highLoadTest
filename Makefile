.PHONY: build build-linux clean

build:
	go build -o highLoadTest main.go

build-linux:
	GOOS=linux GOARCH=amd64 go build -o highLoadTest main.go

clean:
	rm -f highLoadTest