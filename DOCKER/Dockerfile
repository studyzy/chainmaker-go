FROM golang:1.16 as builder
ENV GOPROXY=https://goproxy.cn,direct
COPY . /chainmaker-go
RUN cd /chainmaker-go && make chainmaker && ls /chainmaker-go/bin

# the second stage
FROM ubuntu
COPY --from=builder /chainmaker-go/module/vm/wasmer/wasmer-go/libwasmer.so /usr/lib/libwasmer.so
COPY --from=builder /chainmaker-go/bin/chainmaker /usr/bin/chainmaker

WORKDIR /chainmaker-go
CMD ["chainmaker", "-e", "wx-org1", "start"]
