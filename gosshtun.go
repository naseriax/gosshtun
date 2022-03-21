package gosshtun

import (
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

func errd(err error, openConns ...interface{}) error {
	if err != nil {
		for _, o := range openConns {
			if obj, ok := o.(net.Listener); ok {
				obj.Close()
			} else if obj, ok := o.(net.Conn); ok {
				obj.Close()
			} else if obj, ok := o.(*ssh.Client); ok {
				obj.Close()
			}
		}
		errContent := err.Error()
		if strings.Contains(errContent, "administratively prohibited (open failed)") {
			return fmt.Errorf("TCP forwarding failure! - Possibly the TCPForwarding is not allowed in the jump server! - Error details: %v", errContent)
		} else if strings.Contains(errContent, "i/o timeout") {
			return fmt.Errorf("jump server ip address is not reachable - Error details: %v", errContent)
		} else if strings.Contains(errContent, "No connection could be made because the target machine actively refused it") {
			return fmt.Errorf("jump server port number is not accessible - Error details: %v", errContent)
		} else if strings.Contains(errContent, "unable to authenticate") {
			return fmt.Errorf("jump server username or password is wrong - Error details: %v", errContent)
		} else if strings.Contains(errContent, "A socket operation was attempted to an unreachable host") {
			return fmt.Errorf("jump server is not routable - Error details: %v", errContent)
		} else if strings.Contains(errContent, "rejected: connect failed (Connection refused)") {
			return fmt.Errorf("connection from jump server to remote node is rejected. could it be wrong remote port number?! - Error details: %v", errContent)

		} else {
			return fmt.Errorf(errContent)
		}
	} else {
		return nil
	}
}

func Pipe(errch chan error, writer, reader net.Conn) {
	_, err := io.Copy(writer, reader)
	writer.Close()
	reader.Close()
	if err != nil {
		if strings.Contains(err.Error(), "An existing connection was forcibly closed by the remote host") {
			errch <- nil
		} else {
			errch <- err
		}
	} else {
		errch <- err
	}
}

func Tunnel(jumpserver map[string]string, remoteAddr string, localPortNo chan<- string, tunnelDone chan<- error) {
	var remoteCon net.Conn
	defer func(tunnelDone chan<- error) { tunnelDone <- nil }(tunnelDone)
	sshConfig := &ssh.ClientConfig{
		User:            jumpserver["USER"],
		Auth:            []ssh.AuthMethod{ssh.Password(jumpserver["PASSW"])},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         time.Duration(10) * time.Second,
	}
	lst, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		tunnelDone <- errd(err, nil)
		return
	}
	defer lst.Close()

	//Connect from Local to jump server
	l1, err := ssh.Dial("tcp", jumpserver["ADDR"], sshConfig)
	if err != nil {
		tunnelDone <- errd(err, lst)
		return
	}
	defer l1.Close()

	//Connect from Jumpserver to remote
	remoteDialChan := make(chan net.Conn)
	go func(remoteDialChan chan<- net.Conn) {
		remoteCon, err := l1.Dial("tcp", remoteAddr)
		if err != nil {
			tunnelDone <- errd(err, l1, lst)
			return
		}
		remoteDialChan <- remoteCon
	}(remoteDialChan)
	select {
	case remoteCon = <-remoteDialChan:
	case <-time.After(time.Duration(10) * time.Second):
		tunnelDone <- errd(fmt.Errorf("connection timeout (from Jumpserver to remote node)"), lst, l1)
		return
	}
	defer remoteCon.Close()

	//localPortNo contains the opened port on localhost to accept the connect.
	localPortNo <- lst.Addr().String()

	//Wait for connection to localport
	localCon, err := lst.Accept()
	if err != nil {
		tunnelDone <- errd(err, remoteCon, l1, lst)
		return
	}
	defer localCon.Close()

	//Copy data from localport to tunnel
	errch := make(chan error)
	go Pipe(errch, remoteCon, localCon)
	go Pipe(errch, localCon, remoteCon)
	for i := 0; i < 2; i++ {
		if err := <-errch; err != nil {
			tunnelDone <- errd(err, localCon, remoteCon, l1, lst)
			return
		}
	}
}
