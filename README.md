# gosshtun

# Usage:
```
package main

import (
	"fmt"
	"log"
	"strings"

    //gosshtun module import
	"github.com/naseriax/gosshtun"

    //Nokia 1830PSS SSH Module import
	"github.com/naseriax/pssh"
)

func main() {

    //We create a new object from pssh module to ssh to Nokia1830PSS Node via cli interface.
    //The IP will be set as 127.0.0.1 since the connection will go through the tunnel. (Enhancement is planned.)
	node := pssh.Nokia_1830PSS{
		Ip:       "127.0.0.1",
		UserName: "admin",
		Password: `admin`,
	}


    //Jumpserver info
	jumpserver := map[string]string{
		"ADDR":  "172.172.172.172:22",
		"USER":  "root",
		"PASSW": "toor",
	}

    //Actual Remote node address to be used for ssh tunnel creation.
	remoteAddr := "192.168.10.123:22"

    //Tunnel control channels.
	tunnelDone := make(chan error)
	localPortNo := make(chan string)

    //Initiating the tunnel by calling Tunnel method from gosshtun module.
    //The expected outcome is to receive a localport number which we can get in the select statement.
	go gosshtun.Tunnel(jumpserver, remoteAddr, localPortNo, tunnelDone)


    //If everything goes well, we will get the local port number to be used to connect to the remote node.
	select {
	case tunPort := <-localPortNo:
		log.Printf("Accepting connection on localhost:%v", strings.Split(tunPort, ":")[1])
		node.Port = strings.Split(tunPort, ":")[1]
	case err := <-tunnelDone:
		log.Fatalln(err)
	}

	//Tunnel is open! your ssh connection codes will be here!
    //I wrapped the normal ssh part to have a better view.
    //It will ssh to the localhost:<port we got from the tunnel>, will execute "show slot *" command on the cli environment and will close the connection.
	func() {
		if err := node.Connect(); err != nil {
			log.Fatalln(err)
		}

		defer node.Disconnect()

		res, err := node.Run("cli", "show slot *", "show version")
		if err != nil {
			log.Fatalln(err)
		}

		fmt.Printf("%+v", res)
	}()

	//the tunnel will be closed the moment ssh connection to the remote node is closed.
	if err := <-tunnelDone; err != nil {
		log.Fatalln(err)
	}
}

```