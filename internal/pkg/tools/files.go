package tools

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"syscall"
)

var (
	ErrEmptyString        = fmt.Errorf("empty string")
	ErrExpextedFileNotDir = fmt.Errorf("expected file and not a directory")
)

// ReadFileToString returns the content of the file the argument names. If no such file
// exists, the argument itself is returned: callers accept "a file or a string".
// A text that cannot be a path at all (too long, containing a NUL byte) is a string too.
func ReadFileToString(fileOrString string) (result string, err error) {
	if fileOrString == "" {
		return "", ErrEmptyString
	}

	fileInfo, err := os.Stat(fileOrString)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ENAMETOOLONG) || errors.Is(err, syscall.EINVAL) {
			return fileOrString, nil
		}
		return "", err
	}
	if fileInfo.IsDir() {
		return "", ErrExpextedFileNotDir
	}

	file, err := os.ReadFile(fileOrString)
	if err != nil {
		return "", err
	}
	return string(file), nil
}

func CheckIsFile(name string) (body string, err error) {
	data, err := os.ReadFile(name)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func CheckFileExists(filePath string) bool {
	_, error := os.Stat(filePath)
	//return !os.IsNotExist(err)
	return !errors.Is(error, os.ErrNotExist)
}
