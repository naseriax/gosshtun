# gosshtun

# Usage:
```
package main

import (
	"log"

	"github.com/naseriax/gosshtun"
)

func main() {
	jumpserver := map[string]string{
		"ADDR":  "172.172.172.172:22",
		"USER":  "root",
		"PASSW": "toor",
	}
	remoteAddr := "192.168.0.1:22"

	tunnelDone := make(chan error)
	localPortNo := make(chan string)
	go gosshtun.Tunnel(jumpserver, remoteAddr, localPortNo, tunnelDone)

	select {
	case tunPort := <-localPortNo:
		log.Printf("Accepting connection on localhost:%v", tunPort)
	case err := <-tunnelDone:
		log.Fatalln(err)
	}
    
	//Tunnel is open! your ssh connection codes should be here:
    //SSH CODES!
	//the tunnel will be closed the moment ssh connection to the remote node is closed.

	if err := <-tunnelDone; err != nil {
		log.Fatalln(err)
	}
}
```