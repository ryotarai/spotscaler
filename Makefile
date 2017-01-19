COMMIT = $$(git describe --always)

.PHONY: build install test genmock
build:
	go build -o bin/spotscaler -ldflags "-X github.com/ryotarai/spotscaler/lib.GitCommit=$(COMMIT)"

install:
	go install -ldflags "-X github.com/ryotarai/spotscaler/lib.GitCommit=$(COMMIT)"

test:
	go test -v github.com/ryotarai/spotscaler/lib

genmock:
	rm lib/mock_*.go
	./scripts/genmock.sh EC2ClientIface
	./scripts/genmock.sh StatusStoreIface
