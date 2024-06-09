package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"time"
)

var (
	listenAddr = flag.String("addr", "0.0.0.0:12345", "listen address")
	connTrack  = make(map[string]time.Time)
)

const (
	recvBufLen   = 2048
	agingTime    = 10 * time.Second
	keepAliveMsg = "keepalive"
)

func main() {
	flag.Parse()

	udpAddr, err := net.ResolveUDPAddr("udp", *listenAddr)
	if err != nil {
		fmt.Println("Error resolving UDP address:", err)
		os.Exit(1)
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		fmt.Println("Error listening:", err)
		os.Exit(1)
	}
	defer conn.Close()

	fmt.Printf("UDP server listening on %s\n", *listenAddr)

	for {
		recvBuf := make([]byte, recvBufLen)
		_, clientAddr, err := conn.ReadFromUDP(recvBuf)
		if err != nil {
			fmt.Println("Error reading:", err)
			continue
		}
		fmt.Printf("RECV from [%s]: %s\n", clientAddr, recvBuf)

		connTrack[clientAddr.String()] = time.Now()
		updateConnTrack()

		sendToOtherClients(conn, clientAddr, recvBuf)
	}
}

func updateConnTrack() {
	curTime := time.Now()

	needDel := make([]string, 0)
	for key, val := range connTrack {
		dur := curTime.Sub(val)
		if dur > agingTime {
			needDel = append(needDel, key)
		}
	}

	for _, key := range needDel {
		fmt.Printf("[%s] is aged.\n", key)
		delete(connTrack, key)
	}
}

func isKeepAlive(buf []byte) bool {
	for i := 0; i < len(keepAliveMsg); i++ {
		if buf[i] != keepAliveMsg[i] {
			return false
		}
	}

	return true
}

func sendToOtherClients(conn *net.UDPConn, clientAddr *net.UDPAddr, buf []byte) {
	if isKeepAlive(buf) {
		return
	}

	msg := []byte(fmt.Sprintf("From [%s]: %s", clientAddr, buf))

	for key := range connTrack {
		if key == clientAddr.String() {
			continue
		}

		udpAddr, err := net.ResolveUDPAddr("udp", key)
		if err != nil {
			fmt.Println("Error resolving UDP address:", err)
			continue
		}

		_, err = conn.WriteToUDP(msg, udpAddr)
		if err != nil {
			fmt.Println("Error sending:", err)
		}
	}
}
