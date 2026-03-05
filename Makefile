BINARY     := kubes
CMD        := ./cmd/kubes
BUILD_DIR  := .

.PHONY: build run clean

build:
	go build -o $(BUILD_DIR)/$(BINARY) $(CMD)

run: build
	./$(BINARY)

clean:
	rm -f $(BUILD_DIR)/$(BINARY)
