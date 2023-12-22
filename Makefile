.PHONY: build clean

build: seer

seer:
	go build .

clean:
	rm -f seer

rebuild: clean build