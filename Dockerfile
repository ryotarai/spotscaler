FROM golang:1.7
WORKDIR /tmp
RUN curl -o glide.tar.gz -L https://github.com/Masterminds/glide/releases/download/v0.12.3/glide-v0.12.3-linux-amd64.tar.gz && \
  tar xvf glide.tar.gz
ADD glide.yaml .
ADD glide.lock .
RUN ./linux-amd64/glide install

ADD . /go/src/github.com/ryotarai/spot-autoscaler
WORKDIR /go/src/github.com/ryotarai/spot-autoscaler
RUN mv /tmp/vendor . && make install
