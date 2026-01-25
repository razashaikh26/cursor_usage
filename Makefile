.PHONY: build install clean test

build:
	go build -o bin/cursor-monitor ./cmd/cursor-monitor

install: build
	cp bin/cursor-monitor /usr/local/bin/cursor-monitor

clean:
	rm -rf bin/

test:
	go test ./...
