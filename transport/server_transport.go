package transport

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"time"

	"mrpc/codec"
	"mrpc/utils"
)

type serverTransport struct {
	opts *ServerTransportOptions
}

var serverTransportMap = make(map[string]ServerTransport)

func init() {
	serverTransportMap["default"] = DefaultServerTransport
}

// RegisterServerTransport supports business custom registered ServerTransport
func RegisterServerTransport(name string, serverTransport ServerTransport) {
	if serverTransportMap == nil {
		serverTransportMap = make(map[string]ServerTransport)
	}
	serverTransportMap[name] = serverTransport
}

// Get the ServerTransport
func GetServerTransport(transport string) ServerTransport {
	if v, ok := serverTransportMap[transport]; ok {
		return v
	}

	return DefaultServerTransport
}

var DefaultServerTransport = NewServerTransport()

var NewServerTransport = func() ServerTransport {
	return &serverTransport{
		opts: &ServerTransportOptions{},
	}
}

func (s *serverTransport) ListenAndServe(ctx context.Context, opts ...ServerTransportOption) error {
	for _, o := range opts {
		o(s.opts)
	}

	lis, err := net.Listen("tcp", s.opts.Address)
	if err != nil {
		return err
	}
	addr, err := utils.Extract(s.opts.Address, lis)
	if err != nil {
		return err
	}
	fmt.Printf("server listening on %s\n", addr)

	go func() {
		if err = s.serve(ctx, lis); err != nil {
			log.Fatalf("transport serve error, %v", err)
		}
	}()

	return nil
}

func (s *serverTransport) serve(ctx context.Context, lis net.Listener) error {
	var tempDelay time.Duration

	tl, ok := lis.(*net.TCPListener)
	if !ok {
		panic("")
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		conn, err := tl.AcceptTCP()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				time.Sleep(tempDelay)
				continue
			}
			panic(err)
			// return err
		}

		if err = conn.SetKeepAlive(true); err != nil {
			return err
		}

		if s.opts.KeepAlivePeriod != 0 {
			conn.SetKeepAlivePeriod(s.opts.KeepAlivePeriod)
		}

		go func() {
			if err := s.handleConn(ctx, wrapConn(conn)); err != nil {
				fmt.Printf("mrpc handle tcp conn error, %v", err)
			}
		}()
	}
}

func (s *serverTransport) handleConn(ctx context.Context, conn *connWrapper) error {
	defer conn.Close()
	// fmt.Printf("have req\n")

	for {
		// check upstream ctx is done
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		frame, err := s.read(ctx, conn)
		if err == io.EOF {
			// read compeleted
			return nil
		}

		if err != nil {
			fmt.Println("-------------", err)
			return err
		}

		rsp, err := s.handle(ctx, frame)
		if err != nil {
			fmt.Printf("s.handle err is not nil, %v\n", err)
		}

		if err = s.write(ctx, conn, rsp); err != nil {
			fmt.Println("write: ", err)
			return err
		}
	}

}

func (s *serverTransport) read(ctx context.Context, conn *connWrapper) ([]byte, error) {
	frame, err := conn.framer.ReadFrame(conn)

	if err != nil {
		return nil, err
	}

	return frame, nil
}

func (s *serverTransport) handle(ctx context.Context, frame []byte) ([]byte, error) {
	// parse reqbuf into req interface {}
	serverCodec := codec.DefaultCodec

	reqbuf, err := serverCodec.Decode(frame)
	if err != nil {
		fmt.Printf("server Decode error: %v", err)
		return nil, err
	}

	rspbuf, err := s.opts.Handler.Handle(ctx, reqbuf)
	if err != nil {
		// todo: handle error
		fmt.Printf("server Handle error: %v", err)
	}

	rspbody, err := serverCodec.Encode(rspbuf)
	if err != nil {
		fmt.Printf("server Encode error, response: %v, err: %v", rspbuf, err)
		return nil, err
	}

	return rspbody, nil
}

func (s *serverTransport) write(ctx context.Context, conn net.Conn, rsp []byte) error {
	if _, err := conn.Write(rsp); err != nil {
		fmt.Printf("conn Write err: %v", err)
	}

	return nil
}

type connWrapper struct {
	net.Conn
	framer Framer
}

func wrapConn(rawConn net.Conn) *connWrapper {
	return &connWrapper{
		Conn:   rawConn,
		framer: NewFramer(),
	}
}
