.PHONY: build test vet fmt check cover complexity staticcheck tidy check-tidy setup

setup:
	sh scripts/install-hooks.sh
	@echo "Hooks installed. Run 'make build' to build the binary."

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

duplicates:
	@which jscpd >/dev/null 2>&1 || npm install -g jscpd
	jscpd --config .jscpd.json ./internal ./main.go

tech-debt:
	@if grep -rnE '(TODO|FIXME|XXX|HACK)\b' --include='*.go' . \
			| grep -v '_test.go' \
			| grep -vE '(TODO|FIXME|XXX|HACK)\(#[0-9]+\)' \
			| grep -q .; then \
		echo "ERROR: Found tech debt markers without issue references." && \
		echo "Use TODO(#123) or FIXME(#456) to link to an issue." && \
		grep -rnE '(TODO|FIXME|XXX|HACK)\b' --include='*.go' . \
			| grep -v '_test.go' \
			| grep -vE '(TODO|FIXME|XXX|HACK)\(#[0-9]+\)' && \
		exit 1; \
	else \
		echo "All tech debt markers reference issues."; \
	fi

tidy:
	go mod tidy

check-tidy: tidy
	@git diff --exit-code go.mod || (echo "go.mod/go.sum not tidy" && exit 1)
