COMMIT = $$(git describe --always)

.PHONY: build install test mockgen
build:
	go build -o bin/spotscaler -ldflags "-X github.com/ryotarai/spotscaler/scaler.GitCommit=$(COMMIT)"

install:
	go install -ldflags "-X github.com/ryotarai/spotscaler/scaler.GitCommit=$(COMMIT)"

test:
	go test -v ./...

mockgen:
	mockgen -destination mock/ec2iface.go -package mock github.com/aws/aws-sdk-go/service/ec2/ec2iface EC2API
