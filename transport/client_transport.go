package transport

import (
	"context"
	"fmt"
	// "errors"
	// "syscall"
)

type clientTransport struct {
	opts *ClientTransportOptions
}

var clientTransportMap = make(map[string]ClientTransport)

func init() {
	clientTransportMap["default"] = DefaultClientTransport
}

// RegisterClientTransport supports business custom registered ClientTransport
func RegisterClientTransport(name string, clientTransport ClientTransport) {
	if clientTransportMap == nil {
		clientTransportMap = make(map[string]ClientTransport)
	}
	clientTransportMap[name] = clientTransport
}

// Get the ServerTransport
func GetClientTransport(transport string) ClientTransport {
	if v, ok := clientTransportMap[transport]; ok {
		return v
	}

	return DefaultClientTransport
}

// The default ClientTransport
var DefaultClientTransport = New()

// Use the singleton pattern to create a ClientTransport
var New = func() ClientTransport {
	return &clientTransport{
		opts: &ClientTransportOptions{},
	}
}

func (c *clientTransport) Send(ctx context.Context, req []byte, opts ...ClientTransportOption) ([]byte, error) {
	for _, o := range opts {
		o(c.opts)
	}

	return c.SendTcpReq(ctx, req)
}

func (c *clientTransport) SendTcpReq(ctx context.Context, req []byte) ([]byte, error) {
	addr := c.opts.Target

	conn, err := c.opts.Pool.Get(ctx, c.opts.Network, addr)
	//	conn, err := net.DialTimeout("tcp", addr, c.opts.Timeout);
	if err != nil {
		// panic(err)
		fmt.Println("get pool err: ", err)
		return nil, err
	}

	defer conn.Close()

	sendNum := 0
	num := 0
	for sendNum < len(req) {
		num, err = conn.Write(req[sendNum:])
		if err != nil {
			// todo this have a error message
			panic(err)
		}
		sendNum += num

		if err = isDone(ctx); err != nil {
			fmt.Println("isdone err: ", err)
			return nil, err
		}
	}

	// parse frame
	wrapperConn := wrapConn(conn)
	frame, err := wrapperConn.framer.ReadFrame(conn)
	if err != nil {
		panic(err)
		// if errors.Is(err, syscall.ECONNRESET) {
		// 	fmt.Println("This is connection reset by peer error: ", err)
		// } else {
		// 	fmt.Println("other error: ", err)
		// }
		// return nil, err
	}

	return frame, nil
}

func isDone(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	return nil
}
