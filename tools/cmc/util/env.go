package util

import "os"

const (
	envKeySdkConfPath = "CMC_SDK_CONF_PATH"
)

var (
	//EnvSdkConfPath 环境变量中的配置文件路径
	EnvSdkConfPath string
)

func init() {
	EnvSdkConfPath = os.Getenv(envKeySdkConfPath)
}
