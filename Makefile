BINARY := a3k
BUILD_FLAGS := CGO_ENABLED=0

.PHONY: build run clean

build:
	$(BUILD_FLAGS) go build -o $(BINARY) ./cmd/a3k/

run:
	$(BUILD_FLAGS) go run ./cmd/a3k/ $(ARGS)

clean:
	rm -f $(BINARY)
