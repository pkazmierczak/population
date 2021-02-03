PROJECT  		  = population

all:
	@echo "make release                 # Build $(PROJECT) for release"
	@echo "make release_linux           # Build $(PROJECT) for release LINUX"
	@echo "make release_windows         # Build $(PROJECT) for release WINDOWS"
	@echo
	@echo "make pkged                   # Package $(PROJECT) with the DB file"

release:
	@echo "* Building $(PROJECT) for release"
	@go build ./cmd/population

release_linux: export GOOS=linux
release_linux: export GOARCH=amd64
release_linux:
	@echo "* Building $(PROJECT) for linux release"
	@go build ./cmd/population

release_windows: export GOOS=windows
release_windows: export GOARCH=amd64
release_windows:
	@echo "* Building $(PROJECT) for windows release"
	@go build ./cmd/population

pkged:
	@echo "* Packaging DB file"
	@pkger
	@echo

