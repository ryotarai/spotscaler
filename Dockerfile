FROM golang:1.7
ADD . /go/src/github.com/ryotarai/spot-autoscaler
WORKDIR /go/src/github.com/ryotarai/spot-autoscaler
RUN make install
