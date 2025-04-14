package fdpassing_test

import (
	"bytes"
	"fdpassing"
	"io"
	"net"
	"os"
	"testing"
)

func TestFd(t *testing.T) {
	data := []byte("helloworld")

	// run server
	ready := make(chan struct{})
	go func(ready chan<- struct{}) {
		ln, err := net.Listen("unix", "@fdpassing-testfd")
		if err != nil {
			panic(err)
		}
		defer ln.Close()

		close(ready)
		conn, err := ln.Accept()
		if err != nil {
			panic(err)
		}
		defer conn.Close()

		// new file
		file, err := os.Create("/tmp/fdpassing-testfd.txt")
		if err != nil {
			panic(err)
		}
		defer file.Close()

		// write data
		file.Write(data)
		file.Sync()
		file.Seek(0, io.SeekStart)

		// passing fd
		unixConn := conn.(*net.UnixConn)
		fileFd := int(file.Fd())
		fdp := fdpassing.NewFd(unixConn)

		if err := fdp.Send(fileFd); err != nil {
			panic(err)
		}

		t.Logf("[S] fd send (%d)", fileFd)
	}(ready)

	// wait
	<-ready
	conn, err := net.Dial("unix", "@fdpassing-testfd")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	// recv fd
	unixConn := conn.(*net.UnixConn)
	fdp := fdpassing.NewFd(unixConn)
	fd, err := fdp.Recv()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("[C] fd recv (%d)", fd)

	file := os.NewFile(uintptr(fd), "recv-file")
	defer file.Close()

	// read data
	content, err := io.ReadAll(file)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(content, data) {
		t.Errorf("[want]: %s, [got]: %s", data, content)
	}
}
