.PHONY: build clean

build:
	go build -o suptunnels-server cmd/server/main.go
	go build -o suptunnels-client cmd/client/main.go

clean:
	rm -f suptunnels-server suptunnels-client
