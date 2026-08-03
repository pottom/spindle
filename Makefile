.PHONY: run run-mock login build cross lint tidy

# Platforms spindle is expected to build for. Kept in the Makefile so a POSIX-only
# syscall cannot creep in unnoticed.
PLATFORMS := linux/amd64 linux/arm64 linux/arm darwin/amd64 darwin/arm64 \
             windows/amd64 windows/arm64 freebsd/amd64

run:
	go run ./cmd/spindle

run-mock:
	go run ./cmd/spindle --mock

login:
	go run ./cmd/spindle login

build:
	go build -o spindle ./cmd/spindle

cross:
	@for p in $(PLATFORMS); do \
		GOOS=$${p%/*} GOARCH=$${p#*/} go build -o /dev/null ./... \
			&& echo "ok   $$p" || { echo "FAIL $$p"; exit 1; }; \
	done

lint:
	go vet ./... && staticcheck ./...

tidy:
	go mod tidy
