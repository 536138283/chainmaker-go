package security

import (
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/config"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/utils"
	"os"
	"path/filepath"
)

type CGroup struct {
	// memory state + monitor
	// cpu state + monitor

}

func SetCGroup() error {
	if _, err := os.Stat(config.CGroupRoot); os.IsNotExist(err) {
		os.Mkdir(config.CGroupRoot, 0755)
	}

	err := setMemoryList()
	if err != nil {
		return err
	}
	return nil
}

func setMemoryList() error {
	// set memroy limit
	mPath := filepath.Join(config.CGroupRoot, config.MemoryLimitFile)
	err := utils.WriteToFile(mPath, config.RssLimit*1024*1024)
	if err != nil {
		return err
	}

	// set swap memory limit to zero
	sPath := filepath.Join(config.CGroupRoot, config.SwapLimitFile)
	err = utils.WriteToFile(sPath, 0)
	if err != nil {
		return err
	}

	return nil
}
