package main

import (
	"encoding/hex"
	"fmt"
	"github.com/spf13/cobra"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"sync"
)

const NAME = "tcpblower"

var Version = "%UNKNOWN%"
var BuildTime = "%UNKNOWN%"

var rootCmd = &cobra.Command{
	Use:     NAME,
	Version: fmt.Sprintf("%s(built at %s)", Version, BuildTime),
	Short:   fmt.Sprintf("%s is a compact testing tool that facilitates the transfer of data between various ports.\n\n", NAME),
	Long: fmt.Sprintf(`%s can forward data between different ports and display it in a hex table format, making it useful for debugging embedded devices. It also supports multiple architectures.
`, NAME),
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}
var _port, _peerPort int

func main() {
	rootCmd.PersistentFlags().IntVarP(&_port, "port", "p", 34050, "port to listen")
	rootCmd.PersistentFlags().IntVarP(&_peerPort, "peer-port", "P", 34051, "peer port to connect")
	rootCmd.Run = run
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) {
	if _port < 0 || _port > 65535 || _peerPort < 0 || _peerPort > 65535 {
		log.Fatal("port must be in range [0, 65535]")
	}
	portA := fmt.Sprintf(":%d", _port)
	portB := fmt.Sprintf(":%d", _peerPort)

	// 启动 A 端口监听器
	go listenPort(portA, portB)

	// 启动 B 端口监听器
	go listenPort(portB, portA)

	// 阻塞主线程，保持程序运行
	select {}
}

type Connections struct {
}

// 使用 sync.Map 维护连接
var conns sync.Map

func listenPort(port string, peerPort string) {
	// 监听指定端口
	l, err := net.Listen("tcp", port)
	if err != nil {
		log.Println("err: error listening:", err.Error())
		return
	}
	defer l.Close()
	log.Println("listening on", port)

	for {
		// 接收连接请求
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("error accepting:", err.Error())
			continue
		}

		// 在 sync.Map 中保存连接
		conns.Store(conn, true)
		log.Println("new connection from", conn.RemoteAddr().String())

		// 启动新协程处理连接
		go handleConn(conn, peerPort, &conns)
	}
}

func handleConn(conn net.Conn, peerPort string, conns *sync.Map) {
	defer func() {
		// 当连接断开时，从 sync.Map 中移除
		conn.Close()
		conns.Delete(conn)
		log.Println("connection from", conn.RemoteAddr().String(), "closed")
	}()

	for {
		// 读取消息
		buf := make([]byte, 1024)
		n, err := conn.Read(buf)
		if err != nil {
			if err != io.EOF {
				log.Println("Error reading from", conn.RemoteAddr().String(), ":", err.Error())
			}
			break
		}
		// 判断是否为心跳包
		heartBeat := judgeHeartBeat(buf[:n])
		if heartBeat {
			port := conn.LocalAddr().String()
			sendToAll(buf[:n], port, conns)
			return
		}
		// 将消息发送给另一个端口的所有设备
		sendToAll(buf[:n], peerPort, conns)
	}
}

func samePort(portA string, portB string) bool {
	if portA == portB {
		return true
	}
	partsOfA := strings.Split(portA, ":")
	partsOfB := strings.Split(portB, ":")
	if len(partsOfA) == 2 && len(partsOfB) == 2 && partsOfA[1] == partsOfB[1] {
		return true
	}
	return false
}

func judgeHeartBeat(msg []byte) bool {
	return len(msg) == 7
}

func sendToAll(msg []byte, port string, conns *sync.Map) {
	heartbeat := judgeHeartBeat(msg)
	// 遍历 sync.Map 中所有连接，找出连接到指定端口的设备
	conns.Range(func(k, v interface{}) bool {
		conn := k.(net.Conn)
		address := conn.RemoteAddr().String()
		if samePort(conn.LocalAddr().String(), port) {
			logTemplate := "send to %s. data = \n%s\n"
			if heartbeat {
				logTemplate = "send heartbeat to %s. data = \n%s\n"
			}
			log.Printf(logTemplate, address, hex.Dump(msg))
			_, err := conn.Write(msg)
			if err != nil {
				fmt.Println("Error sending to", conn.RemoteAddr().String(), ":", err.Error())
			}
		}
		return true
	})
}
