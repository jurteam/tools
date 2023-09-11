#!/usr/bin/make -f

PKGS := $(shell go list ./cmd/...)
BINS =  $(shell basename $(PKGS))
COVERAGE_REPORT_FILENAME ?= coverage.out
BUILDDIR ?= _build/
PKG_DIST_NAME ?= jurtools
DMG_FILE_NAME := $(PKG_DIST_NAME).dmg
CODESIGN_IDENTIY ?= none

ifeq (,$(findstring nostrip,$(BUILD_OPTIONS)))
  ldflags += -w -s
endif
ldflags += $(LDFLAGS)
ldflags := $(strip $(ldflags))

ifneq (,$(ldflags))
  BUILD_FLAGS += -ldflags '$(ldflags)'
endif

# check for nostrip option
ifeq (,$(findstring nostrip,$(BUILD_OPTIONS)))
  BUILD_FLAGS += -trimpath
endif

# Check for debug option
ifeq (debug,$(findstring debug,$(BUILD_OPTIONS)))
  BUILD_FLAGS += -gcflags "all=-N -l"
endif

# Check for the verbose option
ifdef verbose
VERBOSE = -v
endif

all: build check

BUILD_TARGETS := build install

build: BUILD_ARGS=-o $(BUILDDIR)

$(BUILD_TARGETS): generate $(BUILDDIR)
	go $@ $(VERBOSE) -mod=readonly $(BUILD_FLAGS) $(BUILD_ARGS) ./...

$(BUILDDIR):
	mkdir -p $@

check: $(COVERAGE_REPORT_FILENAME)

$(COVERAGE_REPORT_FILENAME): generate
	go test $(VERBOSE) -mod=readonly -race -cover -covermode=atomic -coverprofile=$@ ./...

go.sum: go.mod
	@echo "Ensure dependencies have not been modified ..." >&2
	go mod verify
	go mod tidy
	touch $@

generate: generate-stamp
generate-stamp: go.sum
	go generate ./...
	touch $@

distclean: clean
	rm -rf dist/
	rm -rf $(DMG_FILE_NAME)

clean:
	rm -rf $(BUILDDIR)
	rm -f \
	   $(COVERAGE_REPORT_FILENAME) \
	   macos-codesign-stamp \
	   generate-stamp

list:
	@echo $(BINS) | tr ' ' '\n'

macos-codesign: $(BUILDDIR)
ifneq (,$(CODESIGN_IDENTITY))
	codesign --verbose -s $(CODESIGN_IDENTITY) --options=runtime $(BUILDDIR)/*
else
	@echo Skipping codesigning
endif

$(DMG_FILE_NAME): macos-codesign
ifneq (,$(CODESIGN_IDENTITY))
	create-dmg --volname $(PKG_DIST_NAME) --codesign $(CODESIGN_IDENTITY) --sandbox-safe $@ $(BUILDDIR)
else
	create-dmg --volname $(PKG_DIST_NAME) --sandbox-safe $@ $(BUILDDIR)
endif

.PHONY: all clean check distclean build list macos-codesign
