NAME=kube-pod-decorator
EXE:=dist/$(NAME)$(shell go env GOEXE)

# Dependencies handled by go+glide. Requires 1.5+
export GO15VENDOREXPERIMENT=1

GLIDE_VERSION?=v0.12.3

GOOS=$(shell go env GOOS)
GOARCH=$(shell go env GOARCH)

GLIDE_TGZ=glide-$(GLIDE_VERSION)-$(GOOS)-$(GOARCH).tar.gz
ifeq ($(GOOS),windows)
GLIDE_EXEC=glide-$(GLIDE_VERSION).exe
else
GLIDE_EXEC=glide-$(GLIDE_VERSION)
endif

SHELL:=/bin/bash

ifeq ($(GOOS),windows)
export CC:=x86_64-w64-mingw32-gcc
export CXX:=x86_64-w64-mingw32-g++
endif

# determine archives to make for the build job which determines what will be uploaded/archived by the CI tool
ifeq ($(GOOS),linux)
UPLOADS:=build/$(NAME)-linux-amd64.tgz build/$(NAME)-windows-amd64.zip
else
  ifeq ($(GOOS),darwin)
UPLOADS:=build/$(NAME)-darwin-amd64.tgz build/$(NAME)-linux-amd64.tgz
  else
    ifeq ($(GOOS),windows)
UPLOADS:=build/$(NAME)-windows-amd64.zip
    else
UPLOADS:=build/$(NAME)-$(GOOS)-$(GOARCH).tgz
    endif
  endif
endif

# the default target builds a binary in the top-level dir for whatever the local OS is
default: $(EXE)
$(EXE): *.go version
	go build -o $(EXE) 

install: $(EXE)
	go install

format:
	go fmt ./...

# the standard build produces a "local" executable, a linux tgz, and a darwin (macos) tgz
build: $(EXE) $(UPLOADS)

# create a tgz with the binary and any artifacts that are necessary
# note the hack to allow for various GOOS & GOARCH combos
build/$(NAME)-%.tgz: *.go version
	rm -rf build/$(NAME)
	mkdir -p build/$(NAME)
	tgt=$*; CGO_ENABLED=0 GOOS=$${tgt%-*} GOARCH=$${tgt#*-} go build -tags netgo -a -v -o build/$(NAME)/$(NAME) .
	chmod +x build/$(NAME)/$(NAME)
	tar -zcf $@ -C build $(NAME)
	rm -r build/$(NAME)

# create a zip with the binary and any artifacts that are necessary
# note the hack to allow for various GOOS & GOARCH combos, sigh
build/$(NAME)-%.zip: *.go version
	rm -rf build/$(NAME)
	mkdir -p build/$(NAME)
	tgt=$*; GOOS=$${tgt%-*} GOARCH=$${tgt#*-} go build -o build/$(NAME)/$(NAME).exe .
	cd build; zip -r $(notdir $@) $(NAME)
	rm -r build/$(NAME)

bin/$(GLIDE_EXEC):
	mkdir -p bin tmp
	cd tmp && curl -sSfL --tlsv1 --connect-timeout 30 --max-time 180 --retry 3 \
	  -O https://github.com/Masterminds/glide/releases/download/$(GLIDE_VERSION)/$(GLIDE_TGZ)
	cd tmp && tar xzvf $(GLIDE_TGZ)
	mv tmp/$(GOOS)-$(GOARCH)/glide* bin/$(GLIDE_EXEC)
	rm -rf tmp

# Handled natively in GO now for 1.5! Use glide to manage!
depend: bin/$(GLIDE_EXEC)
	./bin/$(GLIDE_EXEC) --quiet install --cache
	for d in $(INSTALL_DEPEND); do (cd vendor/$$d && go install); done

docker:
	docker build -t kube-pod-decorator:latest .

clean:
	rm -rf build $(EXE)
	rm -f version.go

# produce a version string that is embedded into the binary that captures the branch, the date
# and the commit we're building
version:
	@echo -e "// +build $(NAME)\n\npackage main\n\nconst VV = \"$(NAME) $(BRANCH_NAME) - $(DATE) - $(CHANGE_TITLE)\"" \
	  >version.go
	@echo "version.go: `tail -1 version.go`"

test:
	go test ./...