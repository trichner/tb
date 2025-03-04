

.PHONY: install tidy build dist format

build: tidy format
	@go build ./...

install: tidy
	@go install ./...

dist:
	@goreleaser --skip-publish --snapshot --rm-dist

tidy:
	@go get
	@go mod tidy

format:
	@gofumpt -w .