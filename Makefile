
.PHONY: fmt build debug test


build:
	@go build -o ./bin/warden ./cmd/warden/
	@chmod +x ./bin/warden

debug:
	@go build -gcflags=all="-N -l" -o ./bin/warden ./cmd/warden
	@chmod +x ./bin/warden
	@./bin/warden init -s ./tmp/test
	@./bin/warden show -s ./tmp/test masterkey

fmt:
	@go mod tidy -v
	@go fmt ./...

test:
	@xgo test -v -cover -coverprofile=cover.out ./...
