package main

import (
	"fmt"
	"os"

	"github.com/arnopensource/devo/cli"
	"github.com/arnopensource/devo/daemon"
)

const devoConfigFile = "devo.toml"

func main() {
	// Hijack execution flow in child process
	daemon.RunDaemon(devoConfigFile)

	if uid := os.Getuid(); uid == 0 {
		fmt.Println("Devo can't run as root for now")
		return
	}

	// Cli mode

	err := cli.Run(os.Args[1:], devoConfigFile)
	if err != nil {
		fmt.Println(err)
	}

	// Dev mode

	//conf, err := config.Parse(devoConfigFile)
	//if err != nil {
	//	panic(err)
	//}
	//
	//process, err := daemon.GetProcess(conf)
	//if err != nil {
	//	fmt.Println("Could not find daemon:", err)
	//} else {
	//	err = daemon.KillProcess(process)
	//	if err != nil {
	//		fmt.Println("Could not kill daemon:", err)
	//	}
	//	fmt.Println("Stopped daemon")
	//}
	//
	//time.Sleep(1 * time.Second)
	//
	//fmt.Println("Starting daemon")
	//daemon.Fork(conf)
}
