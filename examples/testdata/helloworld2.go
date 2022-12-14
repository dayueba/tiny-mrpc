package testdata

import (
	"context"
	"fmt"
)

type Service2 struct {

}

type AddReply2 struct {
	Result int32 `msgpack:"result"`
}

func (s *Service2) Add(ctx context.Context, req *AddRequest) (*AddReply2, error) {
	fmt.Println("have req")
	rsp := &AddReply2{
		Result: req.A + req.B,
	}

	return rsp, nil
}

func (s *Service2) Count(ctx context.Context, req *HelloRequest) (*CountResponse, error) {
	rsp := &CountResponse{
		Count: 100,
	}
	return rsp, nil
}
