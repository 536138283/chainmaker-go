# the first stage
FROM ubuntu:21.10
RUN rm /bin/sh && ln -s /bin/bash /bin/sh
RUN apt-get update && apt-get install -y vim net-tools tree gcc g++ curl wget gnupg libtinfo5
ENV TZ "Asia/Shanghai"
RUN DEBIAN_FRONTEND=noninteractive apt-get install -y tzdata && \
    echo $TZ > /etc/timezone && \
    ln -fs /usr/share/zoneinfo/$TZ /etc/localtime && \
    dpkg-reconfigure tzdata -f noninteractive


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
RUN cp -r /data/chainmaker-go/config /chainmaker-go/
RUN mkdir -p /chainmaker-go/log/
RUN chmod 755 /usr/bin/wxdec

WORKDIR /chainmaker-go/bin

