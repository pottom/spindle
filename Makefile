.PHONY: run run-mock login build cross lint tidy

# Platforms spindle is expected to build for.
#
# The embedded playback device needs CGO — libvorbis and libFLAC to decode,
# ALSA or AudioToolbox to play — so these cannot be cross-compiled without a
# toolchain per target. go-librespot solves that upstream with Docker and vcpkg;
# until that is wired in here, `make cross` only vets the host.
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64

run:
	go run ./cmd/spindle

run-mock:
	go run ./cmd/spindle --mock

login:
	go run ./cmd/spindle login

# One binary, one place: ./spindle, and nowhere else. Builds scattered under
# /tmp under half a dozen names cost a morning — a picture was fixed and went on
# looking broken because an older one was being run. The version it prints is
# the commit it was built from; see internal/build.
build:
	@go build -o spindle ./cmd/spindle && ./spindle version && echo "  -> ./spindle"

cross:
	@echo "targets: $(PLATFORMS)"
	@echo "cross-compiling them needs a C toolchain per target (see DESIGN.md);"
	@echo "building for the host instead:"
	@go build -o /dev/null ./... && echo "ok   $$(go env GOOS)/$$(go env GOARCH)"

lint:
	go vet ./... && staticcheck ./...

tidy:
	go mod tidy
