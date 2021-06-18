package utils

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
)

// WriteToFile WriteFile write value to file
func WriteToFile(path string, value int) error {
	if err := ioutil.WriteFile(path, []byte(fmt.Sprintf("%d", value)), 0755); err != nil {
		return err
	}
	return nil
}

// RunCmd exec cmd
func RunCmd(command string) error {
	commands := strings.Split(command, " ")
	cmd := exec.Command(commands[0], commands[1:]...)

	cmd.Stdout = os.Stdout

	if err := cmd.Start(); err != nil {
		return err
	}

	if err := cmd.Wait(); err != nil {
		return err
	}

	return nil
}

// ConvertFileToBytes convert file to byte array
func convertFileToBytes(filePath string) ([]byte, error) {
	bytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}

func ConvertBytesToRunnableFile(bytes []byte, newFilePath string, userId int) error {
	if err := convertBytesToFile(bytes, newFilePath); err != nil {
		return err
	}

	if err := setFileRunnable(newFilePath, userId); err != nil {
		return err
	}

	return nil
}

// ConvertBytesToFile convert byte array to file
func convertBytesToFile(bytes []byte, newFilePath string) error {
	f, err := os.Create(newFilePath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(bytes)
	if err != nil {
		return err
	}
	return nil
}

// SetFileRunnable make file runnable, file permission is 700
func setFileRunnable(filePath string, userId int) error {

	err := os.Chmod(filePath, 0700)
	if err != nil {
		return err
	}

	err = os.Chown(filePath, userId, userId)
	if err != nil {
		return err
	}
	return nil
}
