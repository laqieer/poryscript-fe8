# Makefile for poryscript-fe8.
# Provides a default target to compile the tool, a target to (re)generate the
# sample header, an end-to-end check target, and a "clean" target.

.PHONY: all sample check install clean

TARGET := poryscript-fe8
ifeq ($(OS),Windows_NT)
    TARGET := $(TARGET).exe
endif

# Add any new packages to this variable to pick up underlying source files
PACKAGES := ast emitter lexer parser
GOFILES  := main.go $(foreach package,$(PACKAGES),$(wildcard $(package)/*.go))
SOURCES  := $(filter-out %_test.go,$(GOFILES))

$(TARGET): $(SOURCES)
	go build -o $@

all: $(TARGET)

# Regenerate examples/sample.h from examples/sample.pory.
sample: $(TARGET)
	./$(TARGET) -i examples/sample.pory -o examples/sample.h -fcc command_config.fe8.json

# End-to-end validation against a read-only fireemblem8u checkout.
# Set the decomp location with: make check FE8_DIR=/path/to/fireemblem8u
check: $(TARGET)
	FE8_DIR="$(FE8_DIR)" ./check.sh

# Install the binary/configs into a fireemblem8u checkout under tools/poryscript/.
# Set the decomp location with: make install FE8_DIR=/path/to/fireemblem8u
install: $(TARGET)
	FE8_DIR="$(FE8_DIR)" ./install.sh

clean:
	go clean
	rm -rf $(TARGET) check/sample.h .check
