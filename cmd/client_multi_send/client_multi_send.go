package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

type ConnInfo struct {
	conn     *net.UDPConn
	addr     *net.UDPAddr
	lastTime time.Time
}

var (
	remoteAddrCfg = flag.String("remoteAddr", "0.0.0.0:12345", "remote server address")
	localAddrCfg  = flag.String("localAddr", "0.0.0.0:54321", "local address")
	pktLen        = flag.Int("pktLen", 256, "send packet length")
	sendLock      sync.Mutex
	globalConnMap = make(map[string]ConnInfo)
	localAddr     *net.UDPAddr
	localConn     *net.UDPConn
)

const (
	agingTime    = 5 * time.Second
	keepAliveMsg = "keepalive"
	recvBufLen   = 64 * 1024
)

func main() {
	flag.Parse()

	createLocalConn()

	connToAllServer()

	for _, connInfo := range globalConnMap {
		defer connInfo.conn.Close()
	}
	defer localConn.Close()

	go keepAlive()
	go sendMsg()
	recvMsg()
}

func createLocalConn() {
	localAddr, err := net.ResolveUDPAddr("udp", *localAddrCfg)
	if err != nil {
		fmt.Printf("Error resolving local address [%s]: %v\n", *localAddrCfg, err)
		os.Exit(1)
	}

	localConn, err = net.ListenUDP("udp", localAddr)
	if err != nil {
		fmt.Printf("Error listening on local address [%s]: %v\n", *localAddrCfg, err)
		os.Exit(1)
	}
}

func connToAllServer() {
	addrStrs := strings.Split(*remoteAddrCfg, ",")
	for _, addrStr := range addrStrs {
		if _, exists := globalConnMap[addrStr]; exists {
			continue
		}

		connToServer(addrStr)
	}
}

func connToServer(addrStr string) {
	udpAddr, err := net.ResolveUDPAddr("udp", addrStr)
	if err != nil {
		fmt.Println("Error resolving UDP address:", err)
		os.Exit(1)
	}

	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		fmt.Println("Error connecting:", err)
		os.Exit(1)
	}

	globalConnMap[addrStr] = ConnInfo{conn, udpAddr, time.Now()}
}

func keepAlive() {
	for {
		for addrStr, info := range globalConnMap {
			sendLock.Lock()
			_, err := localConn.WriteToUDP([]byte(keepAliveMsg), info.addr)
			if err != nil {
				fmt.Println("Error sending:", err)
			}
			sendLock.Unlock()
			time.Sleep(5 * time.Second)

			curTime := time.Now()
			dur := curTime.Sub(info.lastTime)
			if dur > agingTime {
				info.conn.Close()
				fmt.Printf("Reconnect to server [%s] ... \n", addrStr)
				connToServer(addrStr)
			}
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
		for _, info := range globalConnMap {
			sendLock.Lock()
			_, err := localConn.WriteToUDP([]byte(msg), info.addr)
			if err != nil {
				fmt.Println("Error sending:", err)
			}
			sendLock.Unlock()
		}
		fmt.Printf("SEND %s\n", string(msg[0]))
		seq++
		time.Sleep(1 * time.Second)
	}
}

func recvMsg() {
	for {
		for _, info := range globalConnMap {
			recvBuf := make([]byte, recvBufLen)
			n, addr, err := localConn.ReadFromUDP(recvBuf)
			if err != nil {
				fmt.Printf("ReadFromUDP failed: %d, %v\n", n, err)
				continue
			}
			info.lastTime = time.Now()

			fmt.Printf("RECV from [%v]: %s\n", addr, recvBuf)
		}
	}
}
