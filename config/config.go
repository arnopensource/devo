package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	KillDelay int `toml:"kill_delay"`
	Storage   Storage
	Services  []Service `toml:"service"`
}

type Storage struct {
	PidFile  string `toml:"pid_file"`
	SockFile string `toml:"sock_file"`
	Binaries string
	Log      string
}

type Service struct {
	Name       string
	BinaryPath string `toml:"binary_path"`
	Command    string
	Dir        string
	Restart    struct {
		OnChange bool `toml:"on_change"`
		OnError  bool `toml:"on_error"`
		OnExit   bool `toml:"on_exit"`
	}
	Caddy struct {
		Enable bool
		Host   string
	}
	Log struct {
		Stdout string
		Stderr string
	}
	Env map[string]string
}

func Parse(configFilename string) (*Config, error) {
	// Default options
	config := &Config{
		KillDelay: 5,
		Storage: Storage{
			PidFile:  "~/.devo/devo.pid",
			SockFile: "~/.devo/devo.sock",
			Binaries: "~/.devo/bin/",
			Log:      "~/.devo/devo.log",
		},
	}

	var configData io.Reader
	configData, err := os.Open(configFilename)
	if err != nil {
		fmt.Println("Note : No devo.toml file found, using default options")
		configData = bytes.NewReader(nil)
	}

	err = toml.NewDecoder(configData).SetStrict(true).Decode(config)
	if err != nil {
		return nil, errors.New(errorMessage(err))
	}

	err = check(config)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func errorMessage(err error) string {
	sb := strings.Builder{}

	switch err.(type) {
	case *toml.DecodeError:
		sb.WriteString(err.(*toml.DecodeError).String())
	case *toml.StrictMissingError:
		err := err.(*toml.StrictMissingError)
		for _, description := range err.Errors {
			line, _ := description.Position()
			sb.WriteString(fmt.Sprintf("Line %v : %v is not a valid configuration key\n", line, strings.Join(description.Key(), ".")))
		}
	default:
		sb.WriteString("Unexpected error: " + err.Error())
	}
	return sb.String()
}

func check(devoConfig *Config) error {
	if devoConfig.KillDelay <= 0 {
		return errors.New("kill_delay must be greater than 0")
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	if devoConfig.Storage.PidFile[0] == '~' {
		devoConfig.Storage.PidFile = path.Join(homeDir, devoConfig.Storage.PidFile[1:])
	}
	devoConfig.Storage.PidFile = path.Clean(devoConfig.Storage.PidFile)
	_, err = os.Stat(path.Dir(devoConfig.Storage.PidFile))
	if err != nil {
		return errors.New("pid_file directory does not exist: " + devoConfig.Storage.PidFile)
	}

	if devoConfig.Storage.SockFile[0] == '~' {
		devoConfig.Storage.SockFile = path.Join(homeDir, devoConfig.Storage.SockFile[1:])
	}
	devoConfig.Storage.SockFile = path.Clean(devoConfig.Storage.SockFile)
	_, err = os.Stat(path.Dir(devoConfig.Storage.SockFile))
	if err != nil {
		return errors.New("sock_file directory does not exist: " + devoConfig.Storage.SockFile)
	}

	if devoConfig.Storage.Binaries[0] == '~' {
		devoConfig.Storage.Binaries = path.Join(homeDir, devoConfig.Storage.Binaries[1:])
	}
	devoConfig.Storage.Binaries = path.Clean(devoConfig.Storage.Binaries)
	stat, err := os.Stat(devoConfig.Storage.Binaries)
	if err != nil {
		return errors.New("binaries directory does not exist: " + devoConfig.Storage.Binaries)
	} else if !stat.IsDir() {
		return errors.New("binaries is not a directory: " + devoConfig.Storage.Binaries)
	}

	if devoConfig.Storage.Log[0] == '~' {
		devoConfig.Storage.Log = path.Join(homeDir, devoConfig.Storage.Log[1:])
	}
	devoConfig.Storage.Log = path.Clean(devoConfig.Storage.Log)
	_, err = os.Stat(path.Dir(devoConfig.Storage.Log))
	if err != nil {
		return errors.New("log file directory does not exist: " + devoConfig.Storage.Log)
	}
	devoConfig.Storage.Log = UseDateInFilename(devoConfig.Storage.Log)

	serviceNames := make(map[string]bool)
	for i, service := range devoConfig.Services {
		if service.Name == "" {
			return errors.New("Service name is empty" + fmt.Sprintf(" at index %d", i))
		} else if serviceNames[service.Name] {
			return errors.New("Service name is not unique: " + service.Name)
		}
		serviceNames[service.Name] = true

		if devoConfig.Services[i].BinaryPath == "" {
			return errors.New("Service binary is empty for" + service.Name)
		}
		if devoConfig.Services[i].BinaryPath[0] == '~' {
			devoConfig.Services[i].BinaryPath = path.Join(homeDir, devoConfig.Services[i].BinaryPath[1:])
		}
		devoConfig.Services[i].BinaryPath = path.Clean(devoConfig.Services[i].BinaryPath)
		if _, err = os.Stat(devoConfig.Services[i].BinaryPath); err != nil {
			return errors.New("Service binary does not exist for " + service.Name)
		}

		if service.Log.Stdout != "" {
			if service.Log.Stdout[0] == '~' {
				devoConfig.Services[i].Log.Stdout = path.Join(homeDir, service.Log.Stdout[1:])
			}
			devoConfig.Services[i].Log.Stdout = path.Clean(devoConfig.Services[i].Log.Stdout)
			_, err = os.Stat(path.Dir(devoConfig.Services[i].Log.Stdout))
			if err != nil {
				return errors.New("Service stdout log file directory does not exist: " + devoConfig.Services[i].Log.Stdout)
			}
		}

		if service.Log.Stderr != "" {
			if service.Log.Stderr[0] == '~' {
				devoConfig.Services[i].Log.Stderr = path.Join(homeDir, service.Log.Stderr[1:])
			}
			devoConfig.Services[i].Log.Stderr = path.Clean(devoConfig.Services[i].Log.Stderr)
			_, err = os.Stat(path.Dir(devoConfig.Services[i].Log.Stderr))
			if err != nil {
				return errors.New("Service stderr log file directory does not exist: " + devoConfig.Services[i].Log.Stderr)
			}
		}

		if service.Caddy.Enable && service.Caddy.Host == "" {
			return errors.New("Caddy host is empty for " + service.Name)
		}

		if service.Dir != "" {
			if service.Dir[0] == '~' {
				devoConfig.Services[i].Dir = path.Join(homeDir, service.Dir[1:])
			}
			devoConfig.Services[i].Dir = path.Clean(devoConfig.Services[i].Dir)
			stat, err = os.Stat(devoConfig.Services[i].Dir)
			if err != nil {
				return errors.New("Service execution directory does not exist: " + devoConfig.Services[i].Dir)
			}
			if !stat.IsDir() {
				return errors.New("Service execution directory is not a directory: " + devoConfig.Services[i].Dir)
			}
		}
	}

	return nil
}

var dateParamRegex = regexp.MustCompile(`{([^}]*)}`)

func UseDateInFilename(filename string) string {
	base := path.Base(filename)

	dateParamResult := dateParamRegex.FindAllStringSubmatch(base, -1)
	if dateParamResult == nil {
		return filename
	}

	now := time.Now()
	for _, result := range dateParamResult {
		dateParam := result[0]
		dateExpression := result[1]
		base = strings.Replace(base, dateParam, now.Format(dateExpression), 1)
	}

	return path.Dir(filename) + "/" + base
}
