package warden

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"strings"

	"github.com/jedib0t/go-pretty/v6/text"
)

const (
	_   = iota
	KiB = 1 << (10 * iota)
	MiB = 1 << (10 * iota)
	GiB = 1 << (10 * iota)
	TiB = 1 << (10 * iota)
)

func DefaultIfNil[T any](value interface{}, defaultValue interface{}) T {
	if value != nil {
		return value.(T)
	}
	return defaultValue.(T)
}

func MapToStruct[T any](value any) (T, error) {
	var res T

	b, err := json.Marshal(value)
	if err != nil {
		return res, err
	}

	err = json.Unmarshal(b, &res)
	if err != nil {
		return res, err
	}

	return res, nil
}

func Printf(format string, a ...interface{}) (n int, err error) {
	return fmt.Printf("\n"+format+"\n", a...)
}

func PPrint(data interface{}) (n int, err error) {
	transformer := text.NewJSONTransformer("", strings.Repeat(" ", 2))
	pJson := transformer(data)
	return fmt.Println(pJson)
}

func EnsureDir(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return os.MkdirAll(path, os.ModeDir|0755)
	}
	return nil
}

func LoadJSON[T any](filename string) (T, error) {
	var data T
	fileData, err := os.ReadFile(filename)
	if err != nil {
		return data, err
	}
	return data, json.Unmarshal(fileData, &data)
}

func Filter[T any](data []T, filterFn func(t T) bool) []T {
	filtered := []T{}

	for _, d := range data {
		if filterFn(d) {
			filtered = append(filtered, d)
		}
	}

	return filtered
}

func Cleanup(ctx context.Context) error {
	return nil
}

func GetSystemInfo() (string, string, error) {
	username, err := user.Current()
	if err != nil {
		err = fmt.Errorf("unable to get system user: %+v", err)
		return "", "", err
	}

	hostname, err := os.Hostname()
	if err != nil {
		err = fmt.Errorf("unable to get system hostname: %+v", err)
		return "", "", err
	}

	return username.Username, hostname, nil
}

func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		} else {
			return false, err
		}
	}

	return true, nil
}

func IntPtr(i int) *int {
	return &i
}

func StringPtr(s string) *string {
	return &s
}
