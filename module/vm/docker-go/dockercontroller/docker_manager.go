package dockercontroller

import (
	"bufio"
	"chainmaker.org/chainmaker-go/docker-go/dockercontroller/module"
	"chainmaker.org/chainmaker-go/localconf"
	"chainmaker.org/chainmaker-go/logger"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/go-connections/nat"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
)

type DockerManager struct {
	AttachStdOut  bool
	AttachStderr  bool
	ShowStdout    bool
	ShowStderr    bool
	imageName     string
	containerName string
	sourceDir     string
	targetDir     string
	openPort      nat.Port
	hostPort      string
	dockerDir     string

	lock   sync.Mutex
	ctx    context.Context
	client *client.Client

	Log       *logger.CMLogger
	CDMClient *module.CDMClient
	CDMState  bool
}

// NewDockerManager return docker manager and running a default container
func NewDockerManager(chainId string) *DockerManager {

	// if open docker vm is false, docker manager is nil
	startDockerVm := localconf.ChainMakerConfig.DockerConfig.OpenDockerVM
	if !startDockerVm {
		return nil
	}
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil
	}

	Logger := logger.GetLoggerByChain("[Docker Manager]", chainId)
	cdmClient := module.NewCDMClient(chainId)

	dockerConfig := localconf.ChainMakerConfig.DockerConfig

	hostPort := strconv.Itoa(int(dockerConfig.DockerRpcConfig.Port))
	openPort := nat.Port(hostPort + "/tcp")

	var dockerContainerDir string
	var sourceDir string
	var targetDir string
	var imageName string
	var containerName string

	if dockerConfig.DockerContainerDir == "" {
		Logger.Errorf("doesn't set docker container path")
		return nil
	} else {
		dockerContainerDir = dockerConfig.DockerContainerDir
	}

	if dockerConfig.HostMountDir == "" {
		Logger.Errorf("doesn't set host mount directory path")
		return nil
	} else {
		sourceDir = dockerConfig.HostMountDir
		if !filepath.IsAbs(sourceDir) {
			sourceDir, err = filepath.Abs(sourceDir)
			if err != nil {
				Logger.Errorf("doesn't set host mount directory path correctly")
				return nil
			}
		}
	}

	if dockerConfig.DockerMountDir == "" {
		Logger.Errorf("doesn't set docker mount directory path")
		return nil
	} else {
		targetDir = dockerConfig.DockerMountDir
	}

	if dockerConfig.ImageName == "" {
		Logger.Infof("image name doesn't set, set as default: image1")
		imageName = "image1"
	} else {
		imageName = dockerConfig.ImageName
	}

	if dockerConfig.ContainerName == "" {
		Logger.Infof("container name doesn't set, set as default: container1")
		containerName = "container1"
	} else {
		containerName = dockerConfig.ContainerName
	}

	newDockerManager := &DockerManager{
		AttachStdOut:  true,
		AttachStderr:  true,
		ShowStdout:    true,
		ShowStderr:    true,
		ctx:           ctx,
		client:        cli,
		Log:           Logger,
		CDMClient:     cdmClient,
		CDMState:      false,
		lock:          sync.Mutex{},
		imageName:     imageName,
		containerName: containerName,
		sourceDir:     sourceDir,
		targetDir:     targetDir,
		hostPort:      hostPort,
		openPort:      openPort,
		dockerDir:     dockerContainerDir,
	}

	return newDockerManager
}

func (m *DockerManager) StartCDMClient() {
	m.lock.Lock()
	defer m.lock.Unlock()
	state := m.CDMClient.StartClient()

	m.CDMState = state

	m.Log.Debugf("cdm client state is: %v", state)
}

func (m *DockerManager) constructEnvs() []string {
	dockerConfig := localconf.ChainMakerConfig.DockerConfig

	configsMap := make(map[string]string)

	m.convertConfigToMap(dockerConfig.DockerLogConfig, configsMap)

	m.convertConfigToMap(dockerConfig.DockerVmConfig, configsMap)

	m.convertConfigToMap(dockerConfig.DockerRpcConfig, configsMap)

	configsMap["DockerMountDir"] = fmt.Sprintf("%s=%s", "DockerMountDir", dockerConfig.DockerMountDir)

	configs := make([]string, len(configsMap))
	index := 0
	for _, value := range configsMap {
		configs[index] = value
		index++
	}

	return configs
}

func (m *DockerManager) convertConfigToMap(config interface{}, configsMap map[string]string) {
	v := reflect.ValueOf(config)
	typeOfS := v.Type()

	for i := 0; i < v.NumField(); i++ {
		fieldName := typeOfS.Field(i).Name
		value := v.Field(i).Interface()

		env := fmt.Sprintf("%s=%v", fieldName, value)
		configsMap[fieldName] = env
	}
}

// create container based on image
func (m *DockerManager) createContainer() error {

	envs := m.constructEnvs()

	_, err := m.client.ContainerCreate(m.ctx, &container.Config{
		Cmd:          nil,
		Image:        m.imageName,
		Env:          envs,
		AttachStdout: m.AttachStdOut,
		AttachStderr: m.AttachStderr,
		ExposedPorts: nat.PortSet{
			m.openPort: struct{}{},
		},
	}, &container.HostConfig{
		PortBindings: nat.PortMap{
			m.openPort: []nat.PortBinding{nat.PortBinding{
				HostIP:   "0.0.0.0",
				HostPort: m.hostPort,
			}},
		},
		Privileged: true,
		Mounts: []mount.Mount{
			{
				Type:        mount.TypeBind,
				Source:      m.sourceDir,
				Target:      m.targetDir,
				ReadOnly:    false,
				Consistency: mount.ConsistencyFull,
				BindOptions: &mount.BindOptions{
					Propagation:  mount.PropagationRPrivate,
					NonRecursive: false,
				},
				VolumeOptions: nil,
				TmpfsOptions:  nil,
			},
		},
	}, nil, nil, m.containerName)

	if err != nil {
		return err
	}

	m.Log.Infof("create container [%s] success :)", m.containerName)
	return nil
}

// BuildImage build image based on Dockerfile
func (m *DockerManager) buildImage(dockerFolderRelPath string) error {

	// get absolute path for docker folder
	dockerFolderPath, err := filepath.Abs(dockerFolderRelPath)

	if err != nil {
		return err
	}

	// tar whole directory
	buildCtx, err := archive.TarWithOptions(dockerFolderPath, &archive.TarOptions{})
	if err != nil {
		return err
	}

	buildOpts := types.ImageBuildOptions{
		Dockerfile: "Dockerfile",
		Tags:       []string{m.imageName},
		Remove:     true,
	}

	// build image
	resp, err := m.client.ImageBuild(m.ctx, buildCtx, buildOpts)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	err = displayBuildProcess(resp.Body)
	if err != nil {
		return err
	}

	m.Log.Infof("build image [%s] success :)", m.imageName)

	return nil
}

type ErrorLine struct {
	Error       string      `json:"error"`
	ErrorDetail ErrorDetail `json:"errorDetail"`
}

type ErrorDetail struct {
	Message string `json:"message"`
}

// display build process
func displayBuildProcess(rd io.Reader) error {
	var lastLine string

	scanner := bufio.NewScanner(rd)
	for scanner.Scan() {

		lastLine = scanner.Text()
		lastLine = strings.TrimLeft(lastLine, "{\"stream\":\"")
		lastLine = strings.TrimRight(lastLine, "\"}")
		lastLine = strings.TrimRight(lastLine, "\\n")
		if len(lastLine) == 0 {
			continue
		}
		if strings.Contains(lastLine, "---\\u003e") {
			continue
		}

		if strings.Contains(lastLine, "Removing intermediate") {
			continue
		}

		if strings.Contains(lastLine, "ux\":{\"ID\":\"sha256") {
			continue
		}
		fmt.Println(lastLine)
	}

	errLine := &ErrorLine{}
	json.Unmarshal([]byte(lastLine), errLine)
	if errLine.Error != "" {
		return errors.New(errLine.Error)
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

// StartContainer Start Container
func (m *DockerManager) StartContainer() error {
	m.Log.Info("start docker vm...")
	var err error

	// check container is running or not
	// if running, stop it,
	isRunning, err := m.getContainer(false)
	if err != nil {
		return err
	}
	if isRunning {
		m.Log.Debugf("stop running container [%s]", m.containerName)
		err = m.stopContainer()
		if err != nil {
			return err
		}
	}

	// check container exist or not, if not exist, create new container
	containerExist, err := m.getContainer(true)
	if err != nil {
		return err
	}

	if containerExist {
		m.Log.Debugf("remove container [%s]", m.containerName)
		err = m.removeContainer()
		if err != nil {
			return err
		}
	}

	// check image exist or not, if not exist, create new image
	imageExisted, err := m.imageExist()
	if err != nil {
		return err
	}

	if imageExisted {
		err = m.removeImage()
		if err != nil {
			return err
		}
	}

	m.Log.Debugf("Starting building image --- %s", m.imageName)
	err = m.buildImage(m.dockerDir)
	if err != nil {
		return err
	}

	m.Log.Debugf("create container [%s]", m.containerName)
	err = m.createContainer()
	if err != nil {
		return err
	}

	// running container
	m.Log.Infof("start running container [%s]", m.containerName)
	if err = m.client.ContainerStart(m.ctx, m.containerName, types.ContainerStartOptions{}); err != nil {
		return err
	}

	m.Log.Info("docker vm start success :)")
	// display container info in the console
	//go m.displayInConsole(m.containerName)

	return nil
}

func (m *DockerManager) StopAndRemoveVM() error {
	var err error

	err = m.stopContainer()
	if err != nil {
		return err
	}

	err = m.removeContainer()
	if err != nil {
		return err
	}

	err = m.removeImage()
	if err != nil {
		return err
	}

	m.Log.Info("stop and remove docker vm")
	return nil
}

// check docker m image exist or not
func (m *DockerManager) imageExist() (bool, error) {
	imageList, err := m.client.ImageList(m.ctx, types.ImageListOptions{All: true})
	if err != nil {
		return false, err
	}

	for _, v1 := range imageList {
		for _, v2 := range v1.RepoTags {
			currentImageName := strings.Split(v2, ":")
			if currentImageName[0] == m.imageName {
				return true, nil
			}
		}
	}
	return false, nil
}

// check container status: exist, not exist, running, or not running
// all true: docker ps -a
// all false: docker ps
func (m *DockerManager) getContainer(all bool) (bool, error) {
	containerList, err := m.client.ContainerList(m.ctx, types.ContainerListOptions{All: all})
	if err != nil {
		return false, err
	}

	indexName := "/" + m.containerName
	for _, v1 := range containerList {
		for _, v2 := range v1.Names {
			if v2 == indexName {
				return true, nil
			}
		}
	}
	return false, nil
}

// remove image
func (m *DockerManager) removeImage() error {

	m.Log.Infof("Removing image [%s] ...", m.imageName)
	if _, err := m.client.ImageRemove(m.ctx, m.imageName, types.ImageRemoveOptions{PruneChildren: true, Force: true}); err != nil {
		return err
	}

	_, err := m.client.ImagesPrune(m.ctx, filters.Args{})
	if err != nil {
		return err
	}
	return nil
}

// remove container
func (m *DockerManager) removeContainer() error {
	m.Log.Infof("Removing container [%s] ...", m.containerName)
	if err := m.client.ContainerRemove(m.ctx, m.containerName, types.ContainerRemoveOptions{}); err != nil {
		return err
	}
	return nil
}

// stop container
func (m *DockerManager) stopContainer() error {
	if err := m.client.ContainerStop(m.ctx, m.containerName, nil); err != nil {
		return err
	}
	return nil
}

// display container std out in host std out -- need finish loop accept
func (m *DockerManager) displayInConsole(containerID string) error {
	//display container std out
	out, err := m.client.ContainerLogs(m.ctx, containerID, types.ContainerLogsOptions{
		ShowStdout: m.ShowStdout,
		ShowStderr: m.ShowStderr,
		Follow:     true,
		Timestamps: false,
	})
	if err != nil {
		return err
	}
	defer out.Close()

	hdr := make([]byte, 8)
	for {
		_, err := out.Read(hdr)
		if err != nil {
			if err == io.EOF {
				break
			}
		}
		var w io.Writer
		switch hdr[0] {
		case 1:
			w = os.Stdout
		default:
			w = os.Stderr
		}
		count := binary.BigEndian.Uint32(hdr[4:])
		dat := make([]byte, count)
		_, err = out.Read(dat)
		fmt.Fprint(w, string(dat))
	}

	return nil
}
