default: test

build:
	go build

build:
	./autocmd

test:
	go test -v

.PHONY: default build run test
