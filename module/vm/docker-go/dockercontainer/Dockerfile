FROM golang:1.15-alpine AS build

WORKDIR /container
ENV CGO_ENABLED=0
ENV GOPROXY=https://goproxy.cn,direct
#COPY go.* ./
COPY . /container
RUN go mod download

RUN go build -o /bin/start

FROM ubuntu:20.04
#RUN rm /bin/sh && ln -s /bin/bash /bin/sh

COPY --from=build /bin/start /bin/start

ENTRYPOINT ["/bin/start"]