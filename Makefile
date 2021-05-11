include embedded-bins/Makefile.variables
include inttest/Makefile.variables

GO_SRCS := $(shell find . -type f -name '*.go' -a ! -name 'zz_generated*')

# EMBEDDED_BINS_BUILDMODE can be either:
#   docker	builds the binaries in docker
#   fetch	fetch precompiled binaries from internet
#   none	does not embed any binaries

EMBEDDED_BINS_BUILDMODE ?= docker

# k0s runs on linux even if its built on mac or windows
TARGET_OS ?= linux
GOARCH ?= $(shell go env GOARCH)
GOPATH ?= $(shell go env GOPATH)
DEBUG ?= false

VERSION ?= $(shell git describe --tags)
ifeq ($(DEBUG), false)
LD_FLAGS ?= -w -s
endif

LD_FLAGS += -X github.com/k0sproject/k0s/pkg/build.Version=$(VERSION)
LD_FLAGS += -X github.com/k0sproject/k0s/pkg/build.RuncVersion=$(runc_version)
LD_FLAGS += -X github.com/k0sproject/k0s/pkg/build.ContainerdVersion=$(containerd_version)
LD_FLAGS += -X github.com/k0sproject/k0s/pkg/build.KubernetesVersion=$(kubernetes_version)
LD_FLAGS += -X github.com/k0sproject/k0s/pkg/build.KineVersion=$(kine_version)
LD_FLAGS += -X github.com/k0sproject/k0s/pkg/build.EtcdVersion=$(etcd_version)
LD_FLAGS += -X github.com/k0sproject/k0s/pkg/build.KonnectivityVersion=$(konnectivity_version)
LD_FLAGS += -X \"github.com/k0sproject/k0s/pkg/build.EulaNotice=$(EULA_NOTICE)\"
LD_FLAGS += -X github.com/k0sproject/k0s/pkg/telemetry.segmentToken=$(SEGMENT_TOKEN)

golint := $(shell which golangci-lint)
ifeq ($(golint),)
golint := go get github.com/golangci/golangci-lint/cmd/golangci-lint@v1.31.0 && "${GOPATH}/bin/golangci-lint"
endif

go_bindata := $(shell which go-bindata)
ifeq ($(go_bindata),)
go_bindata := go get github.com/kevinburke/go-bindata/...@v3.22.0 && "${GOPATH}/bin/go-bindata"
endif

GOLANG_IMAGE = golang:1.16-alpine
GO ?= GOCACHE=/tmp/.cache docker run --rm -v "$(CURDIR)":/go/src/github.com/k0sproject/k0s \
	-w /go/src/github.com/k0sproject/k0s \
	-e GOOS \
	-e CGO_ENABLED \
	-e GOARCH \
	-e GOCACHE \
	$(GOLANG_IMAGE) go

.PHONY: build
ifeq ($(TARGET_OS),windows)
build: k0s.exe
else
build: k0s
endif

.PHONY: all
all: k0s k0s.exe

zz_os = $(patsubst pkg/assets/zz_generated_offsets_%.go,%,$@)
ifeq ($(EMBEDDED_BINS_BUILDMODE),none)
pkg/assets/zz_generated_offsets_linux.go pkg/assets/zz_generated_offsets_windows.go:
	rm -f bindata_$(zz_os) && touch bindata_$(zz_os)
	printf "%s\n\n%s\n%s\n" \
		"package assets" \
		"var BinData = map[string]struct{ offset, size int64 }{}" \
		"var BinDataSize int64 = 0" \
		> $@
else
pkg/assets/zz_generated_offsets_linux.go: .bins.linux.stamp
pkg/assets/zz_generated_offsets_windows.go: .bins.windows.stamp
pkg/assets/zz_generated_offsets_linux.go pkg/assets/zz_generated_offsets_windows.go: gen_bindata.go
	GOOS=${GOHOSTOS} $(GO) run gen_bindata.go -o bindata_$(zz_os) -pkg assets \
	     -gofile pkg/assets/zz_generated_offsets_$(zz_os).go \
	     -prefix embedded-bins/staging/$(zz_os)/ embedded-bins/staging/$(zz_os)/bin
endif

k0s: TARGET_OS = linux
k0s: pkg/assets/zz_generated_offsets_linux.go

k0s.exe: TARGET_OS = windows
k0s.exe: pkg/assets/zz_generated_offsets_windows.go

k0s.exe k0s: static/gen_manifests.go

k0s.exe k0s: $(GO_SRCS)
	CGO_ENABLED=1 GOOS=$(TARGET_OS) GOARCH=$(GOARCH) $(GO) build -v -ldflags='$(LD_FLAGS) -extldflags "-static"' -buildmode=pie -o $@.code main.go
	cat $@.code bindata_$(TARGET_OS) > $@.tmp \
		&& rm -f $@.code \
		&& chmod +x $@.tmp \
		&& mv $@.tmp $@

.bins.windows.stamp .bins.linux.stamp: embedded-bins/Makefile.variables
	$(MAKE) -C embedded-bins buildmode=$(EMBEDDED_BINS_BUILDMODE) TARGET_OS=$(patsubst .bins.%.stamp,%,$@)
	touch $@

.PHONY: lint
lint: pkg/assets/zz_generated_offsets_$(TARGET_OS).go
	$(golint) run ./...

.PHONY: $(smoketests)
check-airgap: image-bundle/bundle.tar
$(smoketests): k0s
	$(MAKE) -C inttest $@

.PHONY: smoketests
smoketests:  $(smoketests)


.PHONY: check-unit
check-unit: pkg/assets/zz_generated_offsets_$(TARGET_OS).go static/gen_manifests.go
	go test -race ./pkg/... ./internal/...

.PHONY: clean
clean:
	rm -f pkg/assets/zz_generated_offsets_*.go k0s k0s.exe .bins.*stamp bindata* static/gen_manifests.go
	$(MAKE) -C embedded-bins clean
	$(MAKE) -C image-bundle clean

.PHONY: manifests
manifests:
	controller-gen crd paths="./..." output:crd:artifacts:config=static/manifests/helm/CustomResourceDefinition object

static/gen_manifests.go: $(shell find static/manifests -type f)
	$(go_bindata) -o static/gen_manifests.go -pkg static -prefix static static/...

.PHONY: generate-bindata
generate-bindata: pkg/assets/zz_generated_offsets_$(TARGET_OS).go

image-bundle/image.list: k0s
	./k0s airgap list-images > image-bundle/image.list

image-bundle/bundle.tar: image-bundle/image.list
	$(MAKE) -C image-bundle bundle.tar
