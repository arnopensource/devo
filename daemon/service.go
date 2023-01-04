package daemon

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/arnopensource/devo/config"
)

type Service struct {
	// Conf
	binaryStorageFolder string
	killDelay           int
	conf                config.Service

	// State
	command       *exec.Cmd
	binaryName    string
	isRunningFlag int32
	logFiles      struct {
		stdout *os.File
		stderr *os.File
	}
}

func NewService(binaryStorageFolder string, killDelay int, conf config.Service) *Service {
	service := &Service{
		binaryStorageFolder: binaryStorageFolder,
		killDelay:           killDelay,
		conf:                conf,
	}
	service.setRunning(false)
	return service
}

func (s *Service) Start() {
	if s.IsRunning() {
		log.Printf("Service %v is already running, double run not supported yet\n", s.conf.Name)
		return
	}

	log.Printf("Starting service %v\n", s.conf.Name)

	err := s.copyBinary()
	if err != nil {
		log.Printf("Cannot start service %v: %v\n", s.conf.Name, err)
		return
	}

	binary := path.Clean(s.binaryStorageFolder + "/" + s.binaryName)
	if s.conf.Command != "" {
		command := strings.Split(strings.ReplaceAll(s.conf.Command, "{binary}", binary), " ")
		name := command[0]
		args := command[1:]
		s.command = exec.Command(name, args...)
	} else {
		s.command = exec.Command(binary)
	}
	s.command.Dir = s.conf.Dir

	env := make([]string, 0, len(s.conf.Env))
	for key, value := range s.conf.Env {
		env = append(env, fmt.Sprintf("%v=%v", key, value))
	}
	s.command.Env = append(os.Environ(), env...)

	//Stdout
	if s.conf.Log.Stdout != "" {
		filename := config.UseDateInFilename(s.conf.Log.Stdout)
		s.logFiles.stdout, err = os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			log.Printf("Service %v cannot open stdout log file %v: %v. defaulting to daemon log\n", s.conf.Name, filename, err)
			s.command.Stdout = os.Stdout
		} else {
			s.command.Stdout = s.logFiles.stdout
		}
	} else {
		s.command.Stdout = os.Stdout
	}

	//Stderr
	if s.conf.Log.Stderr != "" {
		filename := config.UseDateInFilename(s.conf.Log.Stderr)
		s.logFiles.stderr, err = os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			log.Printf("Service %v cannot open stderr log file %v: %v. defaulting to daemon log\n", s.conf.Name, filename, err)
			s.command.Stderr = os.Stderr
		} else {
			s.command.Stderr = s.logFiles.stderr
		}
	} else {
		s.command.Stderr = os.Stderr
	}

	s.setRunning(true)
	go func() {
		defer s.setRunning(false)
		err = s.command.Run()

		if _, errorIsExitError := err.(*exec.ExitError); err != nil && !errorIsExitError {
			log.Println("Error running service:", err)
			return
		}

		log.Printf("Service %v exited with exit code %v\n", s.conf.Name, s.command.ProcessState.ExitCode())
	}()
}

func (s *Service) Stop() {
	if !s.IsRunning() {
		log.Printf("Service %v is not running\n", s.conf.Name)
		return
	}

	log.Printf("Stopping service %v\n", s.conf.Name)

	err := s.command.Process.Signal(syscall.SIGTERM)
	if err != nil {
		log.Printf("Error stopping service %v: %v\n", s.conf.Name, err)
		return
	}

	// Wait for the process to exit for 3 seconds
	for i := 0; i < 30; i++ {
		if !s.IsRunning() {
			break
		}
		time.Sleep(time.Millisecond * 100)
	}

	if s.IsRunning() {
		log.Printf("Service %v did not stop, sending SIGKILL\n", s.conf.Name)
		err = s.command.Process.Signal(syscall.SIGKILL)
		if err != nil {
			log.Printf("Error killing service %v: %v\n", s.conf.Name, err)
			return
		}
	}

	// Close log files
	if s.logFiles.stdout != nil {
		s.logFiles.stdout.Close()
	}
	if s.logFiles.stderr != nil {
		s.logFiles.stderr.Close()
	}
}

func (s *Service) Restart() {
	if s.IsRunning() {
		s.Stop()
	}
	s.Start()
}

func (s *Service) IsRunning() bool {
	return atomic.LoadInt32(&s.isRunningFlag) == 1
}

func (s *Service) setRunning(value bool) {
	if value {
		atomic.StoreInt32(&s.isRunningFlag, 1)
	} else {
		atomic.StoreInt32(&s.isRunningFlag, 0)
	}
}

func (s *Service) copyBinary() error {
	// This will change with hotswap or history
	if s.IsRunning() {
		return fmt.Errorf("cannot change a binaryName of a running service")
	}

	// Generate binary name
	binaryName := ""
	for nameIsUsed := true; nameIsUsed; {
		binaryName = s.conf.Name + "-" + fmt.Sprintf("%08x", rand.Uint32())

		if _, err := os.Stat(path.Clean(s.binaryStorageFolder + "/" + binaryName)); err != nil {
			// file does not exist
			nameIsUsed = false
		}
	}
	s.binaryName = binaryName

	// Copy binary
	err := copyFile(path.Clean(s.binaryStorageFolder+"/"+s.binaryName), s.conf.BinaryPath)
	if err != nil {
		return fmt.Errorf("error copying binary %s : %s", s.binaryName, err)
	}

	err = os.Chmod(path.Clean(s.binaryStorageFolder+"/"+s.binaryName), 0755)
	if err != nil {
		return fmt.Errorf("error making binary executable %s : %s", s.binaryName, err)
	}

	return nil
}

func copyFile(dest string, source string) error {
	destFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer func() {
		err = destFile.Close()
		if err != nil {
			panic(err)
		}
	}()

	sourceFile, err := os.Open(source)
	if err != nil {
		return err
	}
	defer func() {
		err = sourceFile.Close()
		if err != nil {
			panic(err)
		}
	}()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	return nil
}
