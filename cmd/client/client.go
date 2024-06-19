package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net"
	"os"
	"sync"
	"time"
)

var (
	serverAddr   = flag.String("addr", "0.0.0.0:12345", "remote server address")
	pktLen       = flag.Int("pktLen", 256, "send packet length")
	sendLock     sync.Mutex
	globalConn   *net.UDPConn
	lastRecvTime time.Time
)

const (
	agingTime    = 5 * time.Second
	keepAliveMsg = "keepalive"
	recvBufLen   = 64 * 1024
)

func main() {
	flag.Parse()
	connToServer()

	defer globalConn.Close()

	go keepAlive()
	go sendMsg()
	recvMsg()
}

func connToServer() {
	udpAddr, err := net.ResolveUDPAddr("udp", *serverAddr)
	if err != nil {
		fmt.Println("Error resolving UDP address:", err)
		os.Exit(1)
	}

	globalConn, err = net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		fmt.Println("Error connecting:", err)
		os.Exit(1)
	}
}

func keepAlive() {
	for {
		sendLock.Lock()
		_, err := globalConn.Write([]byte(keepAliveMsg))
		if err != nil {
			fmt.Println("Error sending:", err)
		}
		sendLock.Unlock()
		time.Sleep(5 * time.Second)

		curTime := time.Now()
		dur := curTime.Sub(lastRecvTime)
		if dur > agingTime {
			globalConn.Close()
			fmt.Println("Reconnect to server ... ")
			connToServer()
		}
	}
}

func sendMsg() {
	seq := rand.Intn(26)
	msg := make([]byte, *pktLen)
	for i := 0; i < len(msg); i++ {
		msg[i] = 'A' + byte(seq)
	}
	for {
		sendLock.Lock()
		_, err := globalConn.Write([]byte(msg))
		if err != nil {
			fmt.Println("Error sending:", err)
		}
		sendLock.Unlock()
		fmt.Printf("SEND %s\n", string(msg[0]))
		seq++
		time.Sleep(1 * time.Second)
	}
}

func recvMsg() {
	for {
		recvBuf := make([]byte, recvBufLen)
		n, _, err := globalConn.ReadFromUDP(recvBuf)
		if err != nil {
			fmt.Printf("ReadFromUDP failed: %d, %v\n", n, err)
			continue
		}
		lastRecvTime = time.Now()

		fmt.Printf("RECV: %s\n", recvBuf)
	}
}
