package dockercontroller

import (
	"bufio"
	docker_go "chainmaker.org/chainmaker-go/docker-go"
	"chainmaker.org/chainmaker-go/logger"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/go-connections/nat"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const (
	dockerDir     string   = "../module/vm/docker-go/dockercontainer"
	imageName     string   = "image1"
	containerName string   = "container1"
	indexName     string   = "/" + containerName
	openPort      nat.Port = "12355/tcp" // port for container
	hostPort      string   = "12355"     // port for host: 		host port <-> container port
	logTemplate   string   = "Docker Manager  -- %s"
)

type DockerManager struct {
	AttachStdOut bool
	AttachStderr bool
	ShowStdout   bool
	ShowStderr   bool
	ctx          context.Context
	client       *client.Client
	Log          *logger.CMLogger
	Instances    map[string]*docker_go.RuntimeInstance
}

// NewDockerManager return docker manager and running a default container
func NewDockerManager(chainId string) *DockerManager {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil
	}

	log := logger.GetLoggerByChain(logger.MODULE_VM, chainId)
	instances := make(map[string]*docker_go.RuntimeInstance)

	// build default docker go runtime instance
	defaultRuntimeInstance := &docker_go.RuntimeInstance{
		ContainerName: containerName,
		ChainId:       chainId,
	}

	instances[containerName] = defaultRuntimeInstance

	newDockerManager := &DockerManager{
		AttachStdOut: true,
		AttachStderr: true,
		ShowStdout:   true,
		ShowStderr:   true,
		ctx:          ctx,
		client:       cli,
		Log:          log,
		Instances:    instances,
	}

	err = newDockerManager.StartContainer()
	if err != nil {
		fmt.Println("---------------------------------------------------------")
		fmt.Println("problem when start container ", err)
	}

	return newDockerManager
}

// create container based on image
func (m *DockerManager) createContainer() error {

	_, err := m.client.ContainerCreate(m.ctx, &container.Config{
		Cmd:          nil,
		Image:        imageName,
		Env:          nil,
		AttachStdout: m.AttachStdOut,
		AttachStderr: m.AttachStderr,
		ExposedPorts: nat.PortSet{
			openPort: struct{}{},
		},
	}, &container.HostConfig{
		PortBindings: nat.PortMap{
			openPort: []nat.PortBinding{nat.PortBinding{
				HostIP:   "0.0.0.0",
				HostPort: hostPort,
			}},
		},
		Privileged: true,
	}, nil, nil, containerName)

	if err != nil {
		return err
	}

	m.Log.Infof(logTemplate, "Successfully Create Container")
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
		Tags:       []string{imageName},
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

	m.Log.Infof(logTemplate, "Successfully Build Image")

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

	// check image exist or not, if not exist, create new image
	imageExisted, err := m.imageExist()
	if err != nil {
		return err
	}
	if !imageExisted {

		m.Log.Infof("Docker Manager -- Starting building image --- %s", imageName)
		err = m.buildImage(dockerDir)
		if err != nil {
			return err
		}
	} else {
		m.Log.Infof("Docker Manager -- Image %s already exist", imageName)
	}

	// check container exist or not, if not exist, create new container
	containerExist, err := m.getContainer(true)
	if err != nil {
		return err
	}
	if !containerExist {
		m.Log.Infof("Docker Manager -- Container doesn't exist -- %s", containerName)
		err := m.createContainer()
		if err != nil {
			return err
		}
	}

	// check container is running or not
	// if running, stop it,
	isRunning, err := m.getContainer(false)
	if err != nil {
		return err
	}
	if isRunning {
		m.Log.Infof("Docker Manager -- Container is running -- %s", containerName)
		err := m.stopContainer()
		if err != nil {
			return err
		}
	}

	// running container
	m.Log.Infof("Docker Manager -- Start Running Container -- %s", containerName)
	if err := m.client.ContainerStart(m.ctx, containerName, types.ContainerStartOptions{}); err != nil {
		return err
	}

	// display container info in the console
	go m.displayInConsole(containerName)

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
			if currentImageName[0] == imageName {
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

	for _, v1 := range containerList {
		for _, v2 := range v1.Names {
			if v2 == indexName {
				return true, nil
			}
		}
	}
	return false, nil
}

// stop all containers
func stopAllContainers(ctx context.Context, cli *client.Client) error {

	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		return err
	}

	for _, specContainer := range containers {
		log.Print("Stopping container ", specContainer.ID[:10], "... ")
		if err := cli.ContainerStop(ctx, specContainer.ID, nil); err != nil {
			return err
		}
		log.Println("Success")
	}
	return nil
}

// remove all containers
func removeAllContainers(ctx context.Context, cli *client.Client) error {
	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		return err
	}

	for _, specContainer := range containers {
		log.Println("Removing container ", specContainer.ID[:10], "... ")
		if err := cli.ContainerRemove(ctx, specContainer.ID, types.ContainerRemoveOptions{}); err != nil {
			return err
		}
		log.Println("Success remove container ", specContainer.ID[:10])
	}
	return nil
}

// remove image
func (m *DockerManager) removeImage() error {

	m.Log.Infof("Docker Manager -- Removing image [%s] ...", imageName)
	if _, err := m.client.ImageRemove(m.ctx, imageName, types.ImageRemoveOptions{PruneChildren: true, Force: true}); err != nil {
		return err
	}
	m.Log.Infof("Docker Manager -- Successfully Remove Container [%s] ...", imageName)
	return nil
}

// stop container
func (m *DockerManager) stopContainer() error {
	m.Log.Infof("Docker Manager -- Stopping container [%s] ...", containerName)
	if err := m.client.ContainerStop(m.ctx, containerName, nil); err != nil {
		return err
	}
	m.Log.Infof("Docker Manager -- Successfully Stop container [%s] ...", containerName)
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
