package daemon

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/arnopensource/devo/config"

	"github.com/sevlyar/go-daemon"
)

// To terminate the daemon use:
//  kill `cat ~/.devo/devo.pid`

// Fork is responsible for forking the process and starting the daemon
func Fork(conf *config.Config) {
	daemonCtx := &daemon.Context{
		PidFileName: conf.Storage.PidFile,
		PidFilePerm: 0644,
		LogFileName: conf.Storage.Log,
		LogFilePerm: 0640,
		WorkDir:     "./",
		Umask:       027,
		Args:        []string{"devo-daemon"},
	}

	d, err := daemonCtx.Reborn()
	if err != nil {
		log.Fatal("Unable to run: ", err)
	}

	// If parent
	if d != nil {
		return
	}

	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, syscall.SIGTERM, syscall.SIGINT)

	defer func() {
		err = daemonCtx.Release()
		if err != nil {
			log.Fatal("Unable to release pid file: ", err)
		}
	}()

	run(conf, signalChannel)
}

// RunDaemon checks if the code executes in the child (daemon) process
// If it is the case, it hijacks the execution flow to run Fork directly, without using the CLI
// This function should be called at the top of the main function from the main package
func RunDaemon(configFileName string) {
	// Check if the code is executed in the child process
	if !daemon.WasReborn() {
		return
	}

	conf, err := config.Parse(configFileName)
	if err != nil {
		// This should not happen since the config file is already validated before the fork
		log.Fatal("Unable to parse config file: ", err)
	}

	Fork(conf)
	// Do not execute the rest of the main function
	os.Exit(0)
}

func GetProcess(config *config.Config) (*os.Process, error) {
	open, err := os.Open(config.Storage.PidFile)
	if err != nil {
		return nil, fmt.Errorf("unable to open pid file: %s", err)
	}

	pidString, err := io.ReadAll(open)
	if err != nil {
		return nil, fmt.Errorf("unable to read pid file: %s", err)
	}

	pid, err := strconv.ParseInt(string(pidString), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("unable to parse pid file: %s", err)
	}

	process, err := os.FindProcess(int(pid))
	if err != nil {
		return nil, fmt.Errorf("cannot find process: %s", err)
	}

	err = process.Signal(syscall.Signal(0))
	if err != nil {
		return nil, fmt.Errorf("daemon is not running: %s", err)
	}

	return process, nil
}

func KillProcess(process *os.Process) error {
	err := process.Signal(syscall.SIGTERM)
	if err != nil {
		return fmt.Errorf("unable to kill process: %s", err)
	}
	return nil
}
