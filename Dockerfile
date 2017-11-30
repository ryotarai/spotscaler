FROM golang:1.9
ADD . /go/src/github.com/ryotarai/spotscaler
WORKDIR /go/src/github.com/ryotarai/spotscaler
RUN make install
