# the first stage
FROM ubuntu:21.10
RUN rm /bin/sh && ln -s /bin/bash /bin/sh
RUN apt-get update && apt-get install -y vim net-tools tree gcc g++ curl wget gnupg
ENV TZ "Asia/Shanghai"
RUN DEBIAN_FRONTEND=noninteractive apt-get install -y tzdata && \
    echo $TZ > /etc/timezone && \
    ln -fs /usr/share/zoneinfo/$TZ /etc/localtime && \
    dpkg-reconfigure tzdata -f noninteractive

RUN curl -LO http://archive.ubuntu.com/ubuntu/pool/main/libf/libffi/libffi6_3.2.1-8_amd64.deb
RUN dpkg -i libffi6_3.2.1-8_amd64.deb

COPY ./DOCKER/llvm.list /etc/apt/sources.list.d/
RUN wget -q -O - http://apt.llvm.org/llvm-snapshot.gpg.key | apt-key add -
RUN apt-get update
RUN apt-get install -y llvm-8 libllvm8

# the second stage
RUN apt-get install -y golang
RUN apt-get install -y build-essential
ENV GOPROXY=https://goproxy.cn,direct
COPY . /data/chainmaker-go
RUN cd /data/chainmaker-go && make chainmaker

RUN cp /data/chainmaker-go/main/libwasmer_runtime_c_api.so /usr/lib/libwasmer.so
RUN cp /data/chainmaker-go/main/prebuilt/linux/wxdec /usr/bin
RUN mkdir -p /chainmaker-go/bin
RUN cp /data/chainmaker-go/bin/chainmaker /chainmaker-go/bin/
RUN mkdir -p /chainmaker-go/config
RUN cp -r /data/chainmaker-go/config /chainmaker-go/config
RUN mkdir -p /chainmaker-go/log/
RUN chmod 755 /usr/bin/wxdec

WORKDIR /chainmaker-go/bin
