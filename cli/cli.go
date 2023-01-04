package cli

import (
	"errors"
	"fmt"

	"github.com/arnopensource/devo/config"
	"github.com/arnopensource/devo/daemon"
)

func Run(args []string, configFileName string) error {

	if len(args) < 1 {
		return StartDaemon(configFileName)
	}

	switch args[0] {
	case "quit":
		return StopDaemon(configFileName)
	case "reload":
		return ReloadConfiguration()
	case "restart":
		return RestartService()
	case "status":
		return DisplayStatus()
	case "debug":
		return Debug()
	case "check":
		return CheckConfiguration(configFileName)
	case "help":
		return DisplayHelp()
	default:
		return errors.New("Unknown command. Try 'devo help'")
	}
}

func StartDaemon(configFileName string) error {
	devoConfig, err := getConfig(configFileName)
	if err != nil {
		return err
	}

	_, err = daemon.GetProcess(devoConfig)
	if err != nil {
		fmt.Println("Starting daemon")
		daemon.Fork(devoConfig)
	} else {
		fmt.Println("Daemon running")
	}

	return nil
}

func StopDaemon(configFileName string) error {
	devoConfig, err := getConfig(configFileName)
	if err != nil {
		return err
	}

	process, err := daemon.GetProcess(devoConfig)
	if err != nil {
		return fmt.Errorf("Daemon is not running: %s", err)
	}

	err = daemon.KillProcess(process)
	if err != nil {
		return fmt.Errorf("Could not kill daemon: %s", err)
	}

	fmt.Println("Stopped daemon")
	return nil
}

func ReloadConfiguration() error {
	return errors.New("Not implemented")
}

func RestartService() error {
	return errors.New("Not implemented")
}

func Debug() error {
	return errors.New("Not implemented")
}

func DisplayStatus() error {
	return errors.New("Not implemented")
}

func CheckConfiguration(configFileName string) error {
	_, err := getConfig(configFileName)
	if err == nil {
		fmt.Println("Configuration OK")
	}
	return err
}

func DisplayHelp() error {
	return errors.New("Not implemented")
}

func getConfig(configFileName string) (*config.Config, error) {
	conf, err := config.Parse(configFileName)
	if err != nil {
		return nil, errors.New("Configuration error in devo.toml:\n\n" + err.Error())
	}
	return conf, nil
}
