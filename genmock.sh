#!/bin/sh
set -ex
echo "// +build test" > lib/mock_$1.go.tmp
echo >> lib/mock_$1.go.tmp
mockery -name=$1 -inpkg -dir=lib -print >> lib/mock_$1.go.tmp
mv lib/mock_$1.go.tmp lib/mock_$1.go
