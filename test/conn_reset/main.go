package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	// "os/signal"
	"syscall"
	"time"
)

func server() {
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal(err)
	}
    defer listener.Close()

    for {
        conn, err := listener.Accept()
        if err != nil {
            log.Fatal("server", err)
            os.Exit(1)
        }
        go handleConn(conn)
    }

	

	// ch := make(chan os.Signal, 1)
	// signal.Notify(ch, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGSEGV)
	// <-ch

	// listener.Close()
}

func handleConn(conn net.Conn) {
// 读取2字节 没问题
	// data := make([]byte, 2)
	// 读取1字节，实际发了两字节就把conn关闭了，就会报错
	data := make([]byte, 1)
	if _, err := conn.Read(data); err != nil {
		log.Fatal("server", err)
	}


	// 连接关闭了，但是没有写东西，客户端读数据，就会报错了
	// 如果在连接关闭之前写入数据，客户端有数据可以读，就不会报错
	// 也就是说是server端问题，不是客户端的
	// if _, err := conn.Write([]byte("a")); err != nil {
	// 	log.Fatal("server", err)
	// }
	conn.Close()
}

func client() {
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		log.Fatal("client", err)
	}

	if _, err := conn.Write([]byte("ab")); err != nil {
		log.Printf("client: %v", err)
	}

	time.Sleep(1 * time.Second) // wait for close on the server side

	data := make([]byte, 1)
	if _, err := conn.Read(data); err != nil {
		log.Printf("client: %v", err)
		if errors.Is(err, syscall.ECONNRESET) {
			log.Print("This is connection reset by peer error")
		}
	} else {
		fmt.Printf("%s\n", data)
	}
}

func main() {
	fmt.Println(len([]byte("ab")))
	go server()

	time.Sleep(3 * time.Second) // wait for server to run

	client()
}
