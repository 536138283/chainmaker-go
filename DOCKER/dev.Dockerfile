# the second stage
FROM ubuntu:20.04
RUN rm /bin/sh && ln -s /bin/bash /bin/sh
RUN apt-get update && apt-get install -y vim net-tools tree wget curl
ENV TZ "Asia/Shanghai"
RUN DEBIAN_FRONTEND=noninteractive apt-get install -y tzdata && \
    echo $TZ > /etc/timezone && \
    ln -fs /usr/share/zoneinfo/$TZ /etc/localtime && \
    dpkg-reconfigure tzdata -f noninteractive

RUN curl -LO http://archive.ubuntu.com/ubuntu/pool/main/libf/libffi/libffi6_3.2.1-8_amd64.deb
RUN dpkg -i libffi6_3.2.1-8_amd64.deb

RUN echo deb http://apt.llvm.org/xenial/ llvm-toolchain-xenial-8 main >> /etc/apt/sources.list.d/llvm.list
RUN echo deb-src http://apt.llvm.org/xenial/ llvm-toolchain-xenial-8 main >> /etc/apt/sources.list.d/llvm.list
RUN wget -q -O - http://apt.llvm.org/llvm-snapshot.gpg.key | apt-key add -
RUN apt-get install llvm-8 libllvm8

COPY ./main/libwasmer_runtime_c_api.so /usr/lib/libwasmer.so
COPY ./main/prebuilt/linux/wxdec /usr/bin
COPY ./bin/chainmaker /chainmaker-go/bin/
COPY ./config /chainmaker-go/config/
RUN mkdir -p /chainmaker-go/log/
RUN chmod 755 /usr/bin/wxdec

WORKDIR /chainmaker-go/bin
