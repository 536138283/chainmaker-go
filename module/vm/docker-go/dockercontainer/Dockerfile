FROM golang:1.15-alpine AS build

WORKDIR /dockervm
ENV CGO_ENABLED=0
ENV GOPROXY=https://goproxy.cn,direct
COPY go.* ./
RUN go mod download
COPY . /dockervm

#RUN go build -o /bin/manager /dockervm/manager/
RUN go build -o /bin/worker /dockervm/worker/
RUN go build -o /bin/start

FROM ubuntu:20.04

COPY --from=build /bin/start /bin/start
COPY --from=build /bin/worker /bin/worker
#COPY --from=build /bin/manager /bin/manager

ENTRYPOINT ["/bin/start"]