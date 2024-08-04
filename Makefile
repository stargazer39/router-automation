EXECUTABLE := rautomate
SRC := .

GOFLAGS :=  -ldflags "-w -s"

.PHONY: all clean

all: build-arm64 build-x86

build-arm64:
	@echo "Building the executable for ARM64..."
	GOARCH=arm64 go build $(GOFLAGS) -o $(EXECUTABLE)-arm64-linux $(SRC)

build-x86:
	@echo "Building the executable for x86..."
	GOARCH=386 go build $(GOFLAGS) -o $(EXECUTABLE)-x86-linux $(SRC)

clean:
	@echo "Cleaning up..."
	rm -f $(EXECUTABLE)-arm64 $(EXECUTABLE)-x86

run-arm64: build-arm64
	./$(EXECUTABLE)-arm64

run-x86: build-x86
	./$(EXECUTABLE)-x86
