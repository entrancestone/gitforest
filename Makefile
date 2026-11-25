.PHONY: build test fmt lint dist install clean demo

VERSION ?= dev
LDFLAGS := -X main.version=$(VERSION)

build:
	go build -ldflags "$(LDFLAGS)" -o gitforest .

test:
	go test -timeout 20m ./...

fmt:
	gofmt -w .

lint:
	gofmt -l .
	go vet ./...

dist:
	@mkdir -p dist
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/gitforest-darwin-arm64 .
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/gitforest-darwin-amd64 .
	GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/gitforest-linux-arm64 .
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/gitforest-linux-amd64 .
	GOOS=windows GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/gitforest-windows-arm64.exe .
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/gitforest-windows-amd64.exe .

install: build
	cp gitforest $(GOPATH)/bin/ 2>/dev/null || cp gitforest /usr/local/bin/

demo: build
	vhs demo.tape

clean:
	rm -rf gitforest dist/ coverage.out
