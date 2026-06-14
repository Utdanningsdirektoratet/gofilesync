.PHONY: build test test-features test-projects clean

# GOEXE is ".exe" on Windows and empty elsewhere, so `make build` produces a
# runnable binary on every platform.
BIN := bin/gofilesync$(shell go env GOEXE)

build:
	go build -o $(BIN) ./cmd/gofilesync

# -count=1 disables the test cache: the harness rebuilds the binary itself, so
# caching by Go source alone could otherwise hide internal/ changes.
test:
	go test -count=1 ./test/...

test-features:
	go test -count=1 ./test/features/ -v

test-projects:
	go test -count=1 ./test/ -v

clean:
	rm -rf bin
