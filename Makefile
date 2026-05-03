
GOCMD=./cmd/gotex


.PHONY: all build dist dist-all build-all clean clean-all

all: build
	
build: dist cmd/gotex/dist
	go build -o dist/gotex $(GOCMD)

build-all: dist-all
	GOOS=darwin GOARCH=arm64 go build -o dist/darwin-arm64/gotex $(GOCMD)
	GOOS=darwin GOARCH=amd64 go build -o dist/darwin-amd64/gotex $(GOCMD)
	GOOS=linux GOARCH=amd64 go build -o dist/linux-amd64/gotex $(GOCMD)
	GOOS=linux GOARCH=arm64 go build -o dist/linux-arm64/gotex $(GOCMD)
	GOOS=windows GOARCH=amd64 go build -o dist/windows-amd64/gotex.exe $(GOCMD)


cmd/gotex/dist: buildtool
	./dist/buildtool

buildtool: dist dist/buildtool

dist/buildtool:
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