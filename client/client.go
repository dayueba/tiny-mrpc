package client

import (
	"context"
	"errors"
	"strconv"

	"sync/atomic"
	"syscall"

	"mrpc/codec"
	"mrpc/interceptor"
	"mrpc/pool/connpool"
	"mrpc/protocol"
	"mrpc/transport"

	// "mrpc/utils"
	"github.com/mitchellh/mapstructure"
)

type Client interface {
	// Invoke 这个方法表示向下游服务发起调用
	Invoke(ctx context.Context, req, rsp interface{}, path string, opts ...Option) error
}

var DefaultClient = New()

var New = func() *defaultClient {
	return &defaultClient{
		opts: &Options{},
	}
}

type defaultClient struct {
	opts  *Options
	msgId int32
}

func (c *defaultClient) Call(ctx context.Context, method string, params interface{}, rsp interface{},
	opts ...Option) error {

	callOpts := make([]Option, 0, len(opts))
	callOpts = append(callOpts, opts...)

	atomic.AddInt32(&c.msgId, 1)
	req := &protocol.Request{
		Method: method,
		Type:   "call",
		Params: params,
		MsgId:  strconv.Itoa(int(c.msgId)),
	}

	err := c.Invoke(ctx, req, rsp, method, callOpts...)
	if err != nil {
		panic(err)
	}

	return nil
}

func (c *defaultClient) Invoke(ctx context.Context, req, rsp interface{}, path string, opts ...Option) error {
	for _, o := range opts {
		o(c.opts)
	}

	return interceptor.ClientIntercept(ctx, req, rsp, c.opts.interceptors, c.invoke)
}

func (c *defaultClient) invoke(ctx context.Context, req, rsp interface{}) error {
	// 对请求体序列化
	serialization := codec.DefaultSerialization
	r := req.(*protocol.Request)
	arr := make([]interface{}, 0)
	arr = append(arr, r.MsgId)
	arr = append(arr, r.Type)
	arr = append(arr, r.Method)
	arr = append(arr, r.Params)
	payload, err := serialization.Marshal(arr)

	if err != nil {
		panic("序列化失败")
	}

	// 添加包头
	clientCodec := codec.DefaultCodec
	reqbody, err := clientCodec.Encode(payload)
	if err != nil {
		panic(err)
	}

	// 发送请求
	clientTransport := c.NewClientTransport()
	clientTransportOpts := []transport.ClientTransportOption{
		transport.WithServiceName(c.opts.serviceName),
		transport.WithClientTarget(c.opts.target),
		transport.WithClientNetwork("tcp"),
		transport.WithClientPool(connpool.GetPool("default")),
		transport.WithTimeout(c.opts.timeout),
	}
	frame, err := clientTransport.Send(ctx, reqbody, clientTransportOpts...)
	if err != nil {
		if errors.Is(err, syscall.ECONNRESET) {
			panic(err)
		}
		return err
	}

	// 对 server 回包进行解包
	rspbuf, err := clientCodec.Decode(frame)
	if err != nil {
		return err
	}

	respp := make([]interface{}, 0)
	err = serialization.Unmarshal(rspbuf, &respp)
	if err != nil {
		return err
	}

	if respp[1].(string) == "error" {
		e := protocol.RpcError{}
		err = mapstructure.Decode(respp[len(respp)-1], &e)
		if err != nil {
			return err
		}
		return e
	}

	// 转结构体
	return mapstructure.Decode(respp[len(respp)-1], &rsp)
}

func (c *defaultClient) NewClientTransport() transport.ClientTransport {
	return transport.DefaultClientTransport
}
