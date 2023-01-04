package daemon

import (
	"fmt"
	"log"
	"os"

	"github.com/arnopensource/devo/config"

	"github.com/fsnotify/fsnotify"
)

func run(config *config.Config, exitSignal chan os.Signal) {
	fmt.Println()
	log.Println("Starting devo daemon")

	watcher := NewWatcher()
	defer watcher.Close()

	var services = make(map[string]*Service)
	var servicesWatches = make(map[string]string)

	for _, service := range config.Services {
		services[service.Name] = NewService(config.Storage.Binaries, config.KillDelay, service)
		services[service.Name].Start()
		defer services[service.Name].Stop()

		if service.Restart.OnChange {
			servicesWatches[service.BinaryPath] = service.Name
			watcher.Add(service.BinaryPath)
		}
	}

	done := make(chan bool)
	go func() {
		for {
			select {
			case event := <-watcher.Events:
				if event.Op&fsnotify.Write == fsnotify.Write {
					if serviceName, ok := servicesWatches[event.Name]; ok {
						log.Printf("Restarting service %v (file changed)\n", serviceName)
						services[serviceName].Restart()
					} else {
						log.Println("Error : File watched is not linked to any service : ", event.Name)
					}
				}
			case signal := <-exitSignal:
				log.Printf("Received stop signal (%v), cleaning up and exiting\n", signal)
				done <- true
			}
		}
	}()
	<-done
}
