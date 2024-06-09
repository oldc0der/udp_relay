package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net"
	"os"
	"strconv"
	"sync"
	"time"
)

var (
	serverAddr = flag.String("addr", "0.0.0.0:12345", "remote server address")
	sendLock   sync.Mutex
)

const (
	recvBufLen   = 2048
	keepAliveMsg = "keepalive"
)

func main() {
	flag.Parse()

	udpAddr, err := net.ResolveUDPAddr("udp", *serverAddr)
	if err != nil {
		fmt.Println("Error resolving UDP address:", err)
		os.Exit(1)
	}

	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		fmt.Println("Error connecting:", err)
		os.Exit(1)
	}
	defer conn.Close()

	go keepAlive(conn)
	go sendMsg(conn)
	recvMsg(conn)
}

func keepAlive(conn *net.UDPConn) {
	for {
		sendLock.Lock()
		_, err := conn.Write([]byte(keepAliveMsg))
		if err != nil {
			fmt.Println("Error sending:", err)
		}
		sendLock.Unlock()
		time.Sleep(5 * time.Second)
	}
}

func sendMsg(conn *net.UDPConn) {
	i := rand.Intn(100)
	for {
		msg := strconv.Itoa(i)
		sendLock.Lock()
		_, err := conn.Write([]byte(msg))
		if err != nil {
			fmt.Println("Error sending:", err)
		}
		sendLock.Unlock()
		fmt.Printf("SEND %s\n", msg)
		i++
		time.Sleep(1 * time.Second)
	}
}

func recvMsg(conn *net.UDPConn) {
	for {
		recvBuf := make([]byte, recvBufLen)
		_, _, err := conn.ReadFromUDP(recvBuf)
		if err != nil {
			fmt.Println("Error reading:", err)
			continue
		}

		fmt.Printf("RECV: %s\n", recvBuf)
	}
}
