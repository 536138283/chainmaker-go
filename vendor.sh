cmd=$1
set -x
export GOROOT=/data/go18/go
export GOPATH=/data/go18/gosrc
export GOPROXY=https://goproxy.cn,direct
export GOTOOLS=$GOROOT/pkg/tool
export PATH=$GOROOT/bin:$GOPATH/bin:$PATH

go env
go version
$cmd
