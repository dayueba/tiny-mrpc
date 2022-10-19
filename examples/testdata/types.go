package testdata

type Response struct {
	Result int `mapstructure:"result" msgpack:"result"`
}

type Request struct {	
	A int `msgpack:"a"`
	B int `msgpack:"b"`
}

type CountResponse struct {
	Count int64 `mapstructure:"count" msgpack:"count"`
}

type HelloRequest struct {
	Msg string
}

type HelloReply struct {
	Msg string
}

type AddRequest struct {
	A int32 `msgpack:"a"`
	B int32 `msgpack:"b"`
}

type AddReply struct {
	Result int32 `msgpack:"result"`
}
