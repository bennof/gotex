GOCMD=./cmd/gotex
PREFIX ?= /usr/local
INSTALL_USER := $(or $(SUDO_USER),$(USER))
INSTALL_HOME := $(or $(SUDO_HOME),$(HOME))
TEXMF := $(INSTALL_HOME)/.gotex/texmf

.PHONY: install install-system install-user install-linux install-darwin

.PHONY: all build dist dist-all build-all clean clean-all

all: build
	
$(GOCMD): build 

build: dist cmd/gotex/dist
	go build -o dist/gotex $(GOCMD)

build-all: dist-all
	GOOS=darwin GOARCH=arm64 go build -o dist/darwin-arm64/gotex $(GOCMD)
	GOOS=darwin GOARCH=amd64 go build -o dist/darwin-amd64/gotex $(GOCMD)
	GOOS=linux GOARCH=amd64 go build -o dist/linux-amd64/gotex $(GOCMD)
	GOOS=linux GOARCH=arm64 go build -o dist/linux-arm64/gotex $(GOCMD)
	#GOOS=windows GOARCH=amd64 go build -o dist/windows-amd64/gotex.exe $(GOCMD)
	#GOOS=windows GOARCH=arm64 go build -o dist/windows-arm64/gotex.exe $(GOCMD)


cmd/gotex/dist: dist/buildtool
	./dist/buildtool


dist/buildtool: dist
	go build -o $@ ./cmd/buildtool

dev: 
	go run -tags dev  $(GOCMD) serve

dist:
	mkdir -p dist	

dist-all:
	mkdir -p dist/darwin-arm64 dist/darwin-amd64 dist/linux-amd64 dist/linux-arm64 dist/windows-amd64

clean:
	rm -rf dist
	rm -f *.pdf
	rm -rf cmd/gotex/dist

clean-all: clean
	rm -rf .tectonic-cache

install: install-system install-user

install-system:
	@uname_s=$$(uname -s); \
	case "$$uname_s" in \
		Linux*)  $(MAKE) install-linux INSTALL_USER="$(INSTALL_USER)" INSTALL_HOME="$(INSTALL_HOME)" TEXMF="$(TEXMF)";; \
		Darwin*) $(MAKE) install-darwin INSTALL_USER="$(INSTALL_USER)" INSTALL_HOME="$(INSTALL_HOME)" TEXMF="$(TEXMF)";; \
		*) echo "Unsupported OS: $$uname_s"; exit 1 ;; \
	esac

install-linux: $(GOCMD)
	install -Dm755 dist/gotex $(DESTDIR)$(PREFIX)/bin/gotex

install-darwin: $(GOCMD)
	install -d $(DESTDIR)$(PREFIX)/bin
	install -m755 dist/gotex $(DESTDIR)$(PREFIX)/bin/gotex


install-user:
	install -d "$(TEXMF)"
	cp -r texmf/* "$(TEXMF)/"
	chown -R $(INSTALL_USER):$(INSTALL_USER) "$(INSTALL_HOME)/.gotex"

	@echo ""
	@echo "Add the following to your shell profile:"
	@echo ""
	@echo "export GOTEX_PATH=$(INSTALL_HOME)/.gotex"
	@echo ""
	@echo "For Bash:"
	@echo "  echo 'export GOTEX_PATH=$(INSTALL_HOME)/.gotex' >> ~/.bashrc"
	@echo ""
	@echo "For Zsh:"
	@echo "  echo 'export GOTEX_PATH=$(INSTALL_HOME)/.gotex' >> ~/.zshrc"