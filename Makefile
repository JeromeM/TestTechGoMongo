.DEFAULT_GOAL := build
.PHONY := build

build:
	mkdir -p build
	go build $(LDFLAGS) -o build/selector main.go

run:
	@for l in `cat .env`; do export $$l; done && \
	go run main.go
