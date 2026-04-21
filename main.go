package main

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

func handleTCP(wg *sync.WaitGroup, conn net.Conn) {
	defer wg.Done()
	defer conn.Close()

	// 连接到IPv4目标（5555）
	ipv4Conn, err := net.Dial("tcp", "127.0.0.1:5555") // 转发到5555
	if err != nil {
		fmt.Println("IPv4连接错误:", err)
		return
	}
	defer ipv4Conn.Close()

	// 启动两个goroutine进行双向数据转发
	var wg2 sync.WaitGroup
	wg2.Add(2)

	go func() {
		defer wg2.Done()
		io.Copy(ipv4Conn, conn)
	}()

	go func() {
		defer wg2.Done()
		io.Copy(conn, ipv4Conn)
	}()

	wg2.Wait()
}

func handleUDP() {
	// 创建IPv6 UDP监听（6666）
	ipv6Addr, err := net.ResolveUDPAddr("udp6", "[::]:6666") // 监听6666
	if err != nil {
		fmt.Println("解析IPv6地址错误:", err)
		return
	}
	ipv6Conn, err := net.ListenUDP("udp6", ipv6Addr)
	if err != nil {
		fmt.Println("UDP监听错误:", err)
		return
	}
	defer ipv6Conn.Close()

	// 创建IPv4 UDP地址（5555）
	ipv4Addr, err := net.ResolveUDPAddr("udp4", "127.0.0.1:5555") // 转发到5555
	if err != nil {
		fmt.Println("解析IPv4地址错误:", err)
		return
	}

	buffer := make([]byte, 4096)
	for {
		n, clientAddr, err := ipv6Conn.ReadFromUDP(buffer)
		if err != nil {
			fmt.Println("读取错误:", err)
			continue
		}

		// 转发到IPv4
		_, err = ipv6Conn.WriteToUDP(buffer[:n], ipv4Addr)
		if err != nil {
			fmt.Println("转发错误:", err)
			continue
		}

		// 从IPv4接收响应并转发回客户端
		ipv6Conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		n, _, err = ipv6Conn.ReadFromUDP(buffer)
		if err == nil {
			ipv6Conn.WriteToUDP(buffer[:n], clientAddr)
		} else if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			// 超时处理
			continue
		} else {
			fmt.Println("读取响应时错误:", err)
		}
	}
}

func main() {
	fmt.Println("启动IPv6到IPv4端口转发服务...")

	var wg sync.WaitGroup

	// TCP转发
	wg.Add(1)
	go func() {
		defer wg.Done()
		ipv6Listener, err := net.Listen("tcp6", ":6666") // 监听6666
		if err != nil {
			fmt.Println("TCP监听错误:", err)
			return
		}
		defer ipv6Listener.Close()

		for {
			conn, err := ipv6Listener.Accept()
			if err != nil {
				fmt.Println("接受连接错误:", err)
				continue
			}

			go handleTCP(&wg, conn)
		}
	}()

	// UDP转发
	go handleUDP()

	fmt.Println("TCP和UDP转发服务已启动！按Enter键退出...")
	fmt.Scanln()
	wg.Wait() // 等待所有TCP处理完
}
