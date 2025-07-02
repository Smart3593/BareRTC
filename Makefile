.PHONY: run
run:
	go run cmd/BareRTC/main.go -debug

.PHONY: build
build:
	go build -o BareRTC cmd/BareRTC/main.go
	go build -o BareBot cmd/BareBot/main.go
