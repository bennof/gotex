# edotex build configuration
GOCMD=./cmd/gotex
PREFIX ?= /usr/local
INSTALL_USER := $(or $(SUDO_USER),$(USER))
INSTALL_GROUP := $(shell id -gn $(INSTALL_USER) 2>/dev/null || id -gn)
INSTALL_HOME := $(or $(SUDO_HOME),$(HOME))

.PHONY: all build build-all dist-all clean install install-linux install-darwin

# default action builds the local gotex binary
all: build

# build the primary gotex binary
build: dist cmd/gotex/dist
	go build -o dist/gotex $(GOCMD)

# build binaries for multiple platforms
build-all: dist-all
	GOOS=darwin GOARCH=arm64 go build -o dist/darwin-arm64/gotex $(GOCMD)
	GOOS=darwin GOARCH=amd64 go build -o dist/darwin-amd64/gotex $(GOCMD)
	GOOS=linux GOARCH=amd64 go build -o dist/linux-amd64/gotex $(GOCMD)
	GOOS=linux GOARCH=arm64 go build -o dist/linux-arm64/gotex $(GOCMD)
	GOOS=linux GOARCH=arm go build -o dist/linux-arm/gotex $(GOCMD)
	GOOS=windows GOARCH=amd64 go build -o dist/windows-amd64/gotex.exe $(GOCMD)
	#GOOS=windows GOARCH=arm64 go build -o dist/windows-arm64/gotex.exe $(GOCMD)

# create platform dist folders for cross builds
dist-all: dist
	mkdir -p dist/darwin-arm64 dist/darwin-amd64 dist/linux-amd64 dist/linux-arm64 dist/linux-arm dist/windows-amd64


# Additional targets for gotex to build
cmd/gotex/dist: dist/buildtool
	./dist/buildtool

dist/buildtool: dist
	go build -o $@ ./cmd/buildtool

dist:
	mkdir -p dist	

# remove generated build artifacts
clean:
	rm -rf dist
	rm -f *.pdf
	rm -rf cmd/gotex/dist

# install the binary for this host platform
install:
	@uname_s=$$(uname -s); \
	case "$$uname_s" in \
		Linux*)  $(MAKE) install-linux INSTALL_USER="$(INSTALL_USER)" INSTALL_HOME="$(INSTALL_HOME)" TEXMF="$(TEXMF)";; \
		Darwin*) $(MAKE) install-darwin INSTALL_USER="$(INSTALL_USER)" INSTALL_HOME="$(INSTALL_HOME)" TEXMF="$(TEXMF)";; \
		*) echo "Unsupported OS: $$uname_s"; exit 1 ;; \
	esac

install-linux: dist/gotex
	install -Dm755 dist/gotex $(DESTDIR)$(PREFIX)/bin/gotex

install-darwin: dist/gotex
	install -d $(DESTDIR)$(PREFIX)/bin
	install -m755 dist/gotex $(DESTDIR)$(PREFIX)/bin/gotex
