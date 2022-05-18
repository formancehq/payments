BINARY_NAME=payments
PKG=./...
COVERAGE_FILE=coverage.out
FAILFAST=-failfast
TIMEOUT=10m
RUN=".*"
ENABLED_LINTERS=gofmt,contextcheck,bodyclose,dupl,errchkjson,errname,durationcheck,exportloopref,gci,importas,nilerr,noctx,unconvert,unparam,wastedassign

all: lint test

build:
	go build -o $(BINARY_NAME)

install: build
	go install -o $(BINARY_NAME)

lint:
	golangci-lint run -v -E $(ENABLED_LINTERS) --fix $(PKG)

test:
	go test -v $(FAILFAST) -coverpkg $(PKG) -coverprofile $(COVERAGE_FILE) -covermode atomic -run $(RUN) -timeout $(TIMEOUT) $(PKG) \
		| sed ''/PASS/s//$(shell printf "\033[32mPASS\033[0m")/'' \
		| sed ''/FAIL/s//$(shell printf "\033[31mFAIL\033[0m")/'' \
		| sed ''/RUN/s//$(shell printf "\033[34mRUN\033[0m")/''

bench:
	go test -bench=. -run=^a $(PKG)

clean:
	go clean
	rm -f $(BINARY_NAME) $(COVERAGE_FILE)
