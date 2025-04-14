# fdpassing

A Go library for file descriptor passing. Utilizing Unix domain sockets, this library enables safe file descriptor transfers between processes.

## What is File Descriptor Passing?

File Descriptor Passing (FD passing) is a technique for transferring an already opened file descriptor from one process to another. In Unix systems, resources such as files, sockets, and pipes are represented by integer values called file descriptors within a process.

While each process typically maintains its own file descriptor table, Unix domain sockets' `SCM_RIGHTS` feature enables the transfer of file descriptors from one process to another, allowing both processes to access the same resource.

**Key benefits:**

- Resource sharing: Multiple processes can share the same file or socket
- Performance improvement: Transfer only the file descriptor without copying large data
- Privilege delegation: Privileged processes can safely pass resources to non-privileged processes
- Graceful service restart: Server processes can transfer connections to new processes when restarting without losing connections

**Technical mechanism:**

File descriptor passing is implemented using the ancillary data feature of Unix domain sockets. The sending process includes the file descriptor in a control message, and the kernel duplicates that file descriptor into the receiving process's file descriptor table. This provides the receiving process with a new file descriptor to the same resource.

## Overview

The `fdpassing` package provides a simple wrapper around the `SCM_RIGHTS` feature of Unix domain sockets to transfer file descriptors from one process to another. File descriptor passing is particularly useful when sharing opened files, sockets, pipes, etc. between processes.

This library provides the following key functions:

- Sending a single file descriptor (`Send` method)
- Receiving a single file descriptor (`Recv` method)

**Note**: The current implementation supports sending and receiving only one file descriptor at a time. To transfer multiple file descriptors, you need to call the Send/Recv methods multiple times.

## Usage

### Installation

```bash
go get github.com/devlights/fdpassing
```

### Basic Examples

#### Sender Process

```go
package main

import (
	"log"
	"net"
	"os"
	
	"github.com/devlights/fdpassing"
)

func main() {
	// Unix socket path
	socketPath := "/tmp/fd_socket"
	
	// Remove existing socket
	os.Remove(socketPath)
	
	// Create listener
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: socketPath, Net: "unix"})
	if err != nil {
		log.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()
	
	log.Printf("Waiting for connection: %s", socketPath)
	
	// Accept client connection
	conn, err := listener.AcceptUnix()
	if err != nil {
		log.Fatalf("Failed to accept connection: %v", err)
	}
	defer conn.Close()
	
	// Open file to be sent
	file, err := os.Open("/path/to/your/file.txt")
	if err != nil {
		log.Fatalf("Failed to open file: %v", err)
	}
	
	// Create fdpassing instance
	fdSender := fdpassing.NewFd(conn)
	
	// Send file descriptor
	err = fdSender.Send(int(file.Fd()))
	if err != nil {
		log.Fatalf("Failed to send FD: %v", err)
	}
	
	log.Println("File descriptor sent successfully")
}
```

#### Receiver Process

```go
package main

import (
	"io"
	"log"
	"net"
	"os"
	
	"github.com/devlights/fdpassing"
)

func main() {
	// Unix socket path
	socketPath := "/tmp/fd_socket"
	
	// Connect to server
	conn, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: socketPath, Net: "unix"})
	if err != nil {
		log.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()
	
	// Create fdpassing instance
	fdReceiver := fdpassing.NewFd(conn)
	
	// Receive file descriptor
	receivedFd, err := fdReceiver.Recv()
	if err != nil {
		log.Fatalf("Failed to receive FD: %v", err)
	}
	
	// Create file from received file descriptor
	file := os.NewFile(uintptr(receivedFd), "received-file")
	defer file.Close()
	
	// Read file content
	content, err := io.ReadAll(file)
	if err != nil {
		log.Fatalf("Failed to read file: %v", err)
	}
	
	log.Printf("Content of received file:\n%s", content)
}
```

## When This Library Is Effective

The `fdpassing` library is particularly useful in the following scenarios:

1. **Multi-process architectures**: When a parent process needs to pass opened files or sockets to child processes

2. **Privilege delegation**: When a process running with high privileges needs to pass file descriptors to a process with lower privileges

3. **Zero-copy communication**: When transferring large amounts of data between processes, improving performance by transferring only file descriptors without copying the data

4. **Hot reloading**: When migrating existing connections to a new process without restarting a service

5. **Sandboxed processes**: When allowing access only to specific files for processes running in restricted environments

6. **Unix socket communications**: When adding file descriptor passing capabilities to systems already using Unix domain sockets for IPC (Inter-Process Communication)

### Limitations

- This library works only on Unix-based systems (Linux, macOS, FreeBSD, etc.). Windows is not supported.
- The sending and receiving processes must be running on the same host machine. File descriptor passing across network boundaries is not supported.
- The current implementation supports sending and receiving only one file descriptor per operation.
- The caller must explicitly close received file descriptors after use (by calling the `Close()` method on the file created with `os.NewFile`).
