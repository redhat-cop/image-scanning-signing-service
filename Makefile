BUILD_COMMIT := $(shell ./scripts/build/get-build-commit.sh)
BUILD_TIMESTAMP := $(shell ./scripts/build/get-build-timestamp.sh)
BUILD_HOSTNAME := $(shell ./scripts/build/get-build-hostname.sh)

LDFLAGS := "-X github.com/redhat-cop/image-security/version.Version=$(VERSION) \
	-X github.com/redhat-cop/image-security/version.Vcs=$(BUILD_COMMIT) \
	-X github.com/redhat-cop/image-security/version.Timestamp=$(BUILD_TIMESTAMP) \
	-X github.com/redhat-cop/image-security/version.Hostname=$(BUILD_HOSTNAME)"

all: operator

# Build manager binary
operator: generate fmt vet
	go build -o build/_output/bin/image-security  -ldflags $(LDFLAGS) github.com/redhat-cop/image-security/cmd/manager

# Run go fmt against code
fmt:
	go fmt ./pkg/... ./cmd/...

# Run go vet against code
vet:
	go vet ./pkg/... ./cmd/...

# Generate code
generate:
	go generate ./pkg/... ./cmd/...

# Test
test: generate fmt vet
	go test ./pkg/... ./cmd/... -coverprofile cover.out