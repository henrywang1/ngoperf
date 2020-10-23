-include .envco
PROJECTNAME := $(shell basename "$(PWD)")
VERSION := 0.1.0

compile:
	@echo "  >  Checking if there is any missing dependencies..."
	@go get
	@echo "  >  Building binary..."
	@go build -o ./bin/"$(PROJECTNAME)"
	@echo "  >  done"

clean:
	@echo "  >  Cleaning build cache"
	@go clean
	@echo "  >  done"

.PHONY: help
all: help
help: Makefile
	@echo
	@echo " Choose a command run in "$(PROJECTNAME)":"
	@echo
	@sed -n 's/^##//p' $< | column -t -s ':' |  sed -e 's/^/ /'
	@echo