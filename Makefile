build:
	go build -o today .

install:
	go build -o ~/bin/today .

.PHONY: build install
