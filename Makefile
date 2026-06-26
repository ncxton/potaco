.PHONY: build test vet fmt check cover complexity staticcheck tidy check-tidy

build:
	go build -o potaco .

test:
	go test ./... -v

cover:
	go test ./... -v -coverprofile=coverage.out -covermode=atomic
	go tool cover -func=coverage.out

vet:
	go vet ./...

fmt:
	gofmt -w .
	gofmt -l .

check: vet fmt test
	@echo "All checks passed"

complexity:
	@which gocyclo >/dev/null 2>&1 || go install github.com/fzipp/gocyclo/cmd/gocyclo@latest
	gocyclo -over 30 .

staticcheck:
	@which staticcheck >/dev/null 2>&1 || go install honnef.co/go/tools/cmd/staticcheck@latest
	staticcheck ./...

tidy:
	go mod tidy

check-tidy: tidy
	@git diff --exit-code go.mod || (echo "go.mod/go.sum not tidy" && exit 1)
