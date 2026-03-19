build:
	go build -ldflags="-s -w" -trimpath -o today .

install:
	go build -ldflags="-s -w" -trimpath -o ~/bin/today .

.PHONY: build install
