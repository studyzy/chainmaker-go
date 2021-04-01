FROM ubuntu:20.04
RUN rm /bin/sh && ln -s /bin/bash /bin/sh
RUN apt-get update
RUN apt-get install -y wget
ENV TZ "Asia/Shanghai"
RUN DEBIAN_FRONTEND=noninteractive apt-get install -y tzdata
RUN echo $TZ > /etc/timezone
RUN ln -fs /usr/share/zoneinfo/$TZ /etc/localtime
RUN dpkg-reconfigure tzdata -f noninteractive

ENV GO_VERSION "1.15.7"
RUN mkdir /opt/go && mkdir -p /workspace/go && \
        wget https://dl.google.com/go/go$GO_VERSION.linux-amd64.tar.gz -P /opt/go
RUN tar -zxvf /opt/go/go$GO_VERSION.linux-amd64.tar.gz -C /opt/go
RUN GOPATH=/workspace/go && GOROOT=/opt/go/go && \
        sed -i "/^export PATH/i export GOROOT=$GOROOT" /root/.bashrc &&\
        sed -i "/^export PATH/i export GOPATH=$GOPATH" /root/.bashrc &&\
        sed -i "s%^export PATH.*$%&:$GOROOT/bin%g" /root/.bashrc && \
        source /root/.bashrc

ENV GOROOT=/opt/go/go
ENV PATH=$PATH:$GOROOT/bin
ENV GO111MODULE="on"
ENV GOPROXY="https://goproxy.cn,direct"

RUN apt-get install -y git make gcc vim

ENV CM_VERSION "v1.0.0_r"
ENV CM_HOME "/data/chainmaker"

RUN mkdir -p $CM_HOME /root/.ssh

ADD ./id_rsa /root/.ssh
RUN chmod 600 /root/.ssh/id_rsa
RUN touch /root/.ssh/known_hosts
RUN ssh-keyscan git.code.tencent.com >> /root/.ssh/known_hosts

WORKDIR $CM_HOME
RUN git clone --recursive git@git.code.tencent.com:ChainMaker/chainmaker-go.git
WORKDIR $CM_HOME/chainmaker-go
RUN git checkout -t remotes/origin/$CM_VERSION
RUN make

ENV LD_LIBRARY_PATH=$CM_HOME/chainmaker-go/main:$LD_LIBRARY_PATH
WORKDIR $CM_HOME/chainmaker-go/bin
