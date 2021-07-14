FROM golang:1.15-alpine AS build

WORKDIR /dockervm
ENV CGO_ENABLED=0
ENV GOPROXY=https://goproxy.cn,direct
COPY . /dockervm
RUN go mod download

RUN go build -o /bin/startvm

FROM ubuntu:20.04
#RUN rm /bin/sh && ln -s /bin/bash /bin/sh

COPY --from=build /bin/startvm /bin/startvm

ENTRYPOINT ["/bin/startvm"]