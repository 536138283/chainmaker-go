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
func ConvertFileToBytes(filePath string) ([]byte, error) {
	bytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}

// ConvertBytesToFile convert byte array to file
func ConvertBytesToFile(bytes []byte, newFilePath string) error {
	f, err := os.Create(newFilePath)
	if err != nil {
		return err
	}
	defer f.Close()

	n, err := f.Write(bytes)
	if err != nil {
		return err
	}

	fmt.Println("-------------")
	fmt.Println(n)
	return nil
}

// SetFileRunnable make file runnable, file permission is 700
func SetFileRunnable(filePath string, userId int) error {

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
