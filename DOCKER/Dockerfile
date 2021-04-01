FROM golang:1.16.2 as builder
ENV GOPROXY=https://goproxy.cn,direct
COPY . /chainmaker-go
RUN cd /chainmaker-go && make chainmaker 

# the second stage
FROM ubuntu:20.04
RUN rm /bin/sh && ln -s /bin/bash /bin/sh
RUN apt-get update
RUN apt-get install -y vim net-tools tree
ENV TZ "Asia/Shanghai"
RUN DEBIAN_FRONTEND=noninteractive apt-get install -y tzdata
RUN echo $TZ > /etc/timezone
RUN ln -fs /usr/share/zoneinfo/$TZ /etc/localtime
RUN dpkg-reconfigure tzdata -f noninteractive

COPY --from=builder /chainmaker-go/main/libwasmer_runtime_c_api.so /usr/lib/libwasmer.so
COPY --from=builder /chainmaker-go/bin/chainmaker /chainmaker-go/bin/
COPY --from=builder /chainmaker-go/config /chainmaker-go/config/
RUN mkdir -p /chainmaker-go/log/

WORKDIR /chainmaker-go/bin
