#!/bin/sh
set -ex
f=lib/mock_$1_test.go
mockery -name=$1 -inpkg -dir=lib -print > $f.tmp
mv $f.tmp $f
