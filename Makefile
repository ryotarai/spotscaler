COMMIT = $$(git describe --always)

.PHONY: build install test genmock
build:
	go build -ldflags "-X github.com/ryotarai/spot-autoscaler/lib.GitCommit=$(COMMIT)"

install:
	go install -ldflags "-X github.com/ryotarai/spot-autoscaler/lib.GitCommit=$(COMMIT)"

test:
	go test -v -tags test github.com/ryotarai/spot-autoscaler/lib

genmock:
	./genmock.sh EC2ClientIface
	./genmock.sh StatusStoreIface
	./genmock.sh MetricProvider
