FROM golang:1.8
ADD . /go/src/github.com/ryotarai/spotscaler
WORKDIR /go/src/github.com/ryotarai/spotscaler
RUN make install
