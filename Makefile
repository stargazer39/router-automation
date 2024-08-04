EXECUTABLE := rautomate

SRC := .

GOFLAGS :=  -ldflags "-w -s"

.PHONY: all clean

all: build

build:
	@echo "Building the executable..."
	go build $(GOFLAGS) -o $(EXECUTABLE) $(SRC)

clean:
	@echo "Cleaning up..."
	rm -f $(EXECUTABLE)

run: build
	./$(EXECUTABLE)
