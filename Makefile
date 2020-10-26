-include .envco
PROJECTNAME := ngoperf
VERSION := 0.2.0

install:
	@go mod tidy && go mod vendor
	@echo "  >  Checking if there is any missing dependencies..."
	@go get
	@echo "  >  Building binary..."
	@go build -o ./bin/"$(PROJECTNAME)"
	@go install
	@echo "  >  done"

clean:
	@echo "  >  Cleaning build cache"
	@go clean
	@go clean -modcache
	@go clean -i
	@rm -rf vendor
	@rm ./bin/"$(PROJECTNAME)"
	@echo "  >  done"