// Copyright (C) Tao Ma(tao.ma.1984@gmail.com), All rights reserved.
// https://github.com/Tao-Ma/rpc/

package rpc

import (
	"github.com/golang/protobuf/proto"
	"math/rand"
	"net"
	"testing"
	"time"
)

func (r *ResourceReq) RpcGetId() uint64 {
	return r.GetId()
}

func (r *ResourceReq) RpcSetId(id uint64) {
	r.Id = proto.Uint64(id)
}

func (r *ResourceReq) RpcIsRequest() bool {
	return true
}

func (r *ResourceReq) RpcIsResponse() bool {
	return false
}

func (r *ResourceResp) RpcGetId() uint64 {
	return r.GetId()
}

func (r *ResourceResp) RpcSetId(id uint64) {
	r.Id = proto.Uint64(id)
}

func (r *ResourceResp) RpcIsRequest() bool {
	return false
}

func (r *ResourceResp) RpcIsResponse() bool {
	return true
}

func ServiceProcessConn(r *Router, c net.Conn) bool {
	return false
}

func ServiceProcessPayload(r *Router, name string, p Payload) bool {
	resp := NewResourceResp()
	req := p.(*ResourceReq)
	resp.Id = proto.Uint64(req.GetId())

	r.Write(name, resp)
	return true
}

func ClientProcessReponse(p Payload, arg rpc_arg, err error) {
	done := arg.(chan bool)
	done <- true
}

func ClientProcessReponseIgnore(p Payload, arg rpc_arg, err error) {
}

func TestRouter(t *testing.T) {
	r, err := NewRouter(nil, ServiceProcessPayload)
	if err != nil {
		t.FailNow()
	}

	hf := NewMsgHeaderFactory(NewProtobufFactory())

	name := "scheduler"
	network := "tcp"
	address := "localhost:10000"

	r.Run()

	if err := r.ListenAndServe("client", network, address, hf, ServiceProcessConn); err != nil {
		t.Log(err)
		t.FailNow()
	}
	if err := r.Dial(name, network, address, hf); err != nil {
		t.Log(err)
		t.FailNow()
	}

	req := NewResourceReq()
	//	if resp := r.CallWait(name, req, 0); resp == nil {
	//t.Log("CallWait timeout")
	//t.FailNow()
	//	}

	done := make(chan bool)
	r.Call("scheduler", req, ClientProcessReponse, done)
	<-done

	r.DelEndPoint("scheduler")

	r.DelListener("client")

	r.Stop()
}

func TestReadWriter(t *testing.T) {
	s, c := net.Pipe()

	ch_c_w := make(chan Payload, 1024)
	ch_s_w := make(chan Payload, 1024)
	ch_d := make(chan Payload, 1024)

	hf := NewMsgHeaderFactory(NewProtobufFactory())

	ep_c := NewEndPoint("c", c, ch_c_w, ch_d, nil, hf, nil)
	ep_s := NewEndPoint("s", s, ch_s_w, ch_s_w, nil, hf, nil)

	ep_c.Run()
	ep_s.Run()

	req := NewResourceReq()
	req.Id = proto.Uint64(1)
	ep_c.write(req)
	<-ch_d
}

func BenchmarkPipeReadWriter(b *testing.B) {
	s, c := net.Pipe()

	ch_c_w := make(chan Payload, 1024)
	ch_s_w := make(chan Payload, 1024)
	ch_d := make(chan Payload, 1024)

	hf := NewMsgHeaderFactory(NewProtobufFactory())

	ep_c := NewEndPoint("c", c, ch_c_w, ch_d, nil, hf, nil)
	ep_s := NewEndPoint("s", s, ch_s_w, ch_s_w, nil, hf, nil)

	ep_c.Run()
	ep_s.Run()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		req := NewResourceReq()
		req.Id = proto.Uint64(1)
		for pb.Next() {
			ch_c_w <- req
			<-ch_d
		}
	})

}

func BenchmarkTCPReadWriter(b *testing.B) {
	network := "tcp"
	address := "localhost:10008"

	l, err := net.Listen(network, address)
	if err != nil {
		b.FailNow()
	}
	c, err := net.Dial(network, address)
	if err != nil {
		b.FailNow()
	}
	s, err := l.Accept()
	if err != nil {
		b.FailNow()
	}

	ch_c_w := make(chan Payload, 1024)
	ch_s_w := make(chan Payload, 1024)
	ch_d := make(chan Payload, 1024)

	hf := NewMsgHeaderFactory(NewProtobufFactory())

	ep_c := NewEndPoint("c", c, ch_c_w, ch_d, nil, hf, nil)
	ep_s := NewEndPoint("s", s, ch_s_w, ch_s_w, nil, hf, nil)

	ep_c.Run()
	ep_s.Run()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		req := NewResourceReq()
		req.Id = proto.Uint64(1)
		for pb.Next() {
			ch_c_w <- req
			<-ch_d
		}
	})

}

func BenchmarkPipeSeperateRouter(b *testing.B) {
	server_r, err := NewRouter(nil, ServiceProcessPayload)
	if err != nil {
		b.FailNow()
	}
	client_r, err := NewRouter(nil, ServiceProcessPayload)
	if err != nil {
		b.FailNow()
	}

	hf := NewMsgHeaderFactory(NewProtobufFactory())

	server_r.Run()
	client_r.Run()
	<-time.Tick(1 * time.Millisecond)

	name := "scheduler"
	n := 128
	for i := 0; i < n; i++ {
		c, s := net.Pipe()
		ep_c := client_r.newRouterEndPoint(name+string(i), c, hf)
		ep_s := server_r.newRouterEndPoint("client"+string(n), s, hf)
		client_r.AddEndPoint(ep_c)
		server_r.AddEndPoint(ep_s)
	}

	<-time.Tick(1 * time.Millisecond)
	testSeperateRouter(b, server_r, client_r, n)
}

func BenchmarkPipeShareRouter(b *testing.B) {
	r, err := NewRouter(nil, ServiceProcessPayload)
	if err != nil {
		b.FailNow()
	}

	hf := NewMsgHeaderFactory(NewProtobufFactory())

	r.Run()
	<-time.Tick(1 * time.Millisecond)

	name := "scheduler"
	n := 128
	for i := 0; i < n; i++ {
		c, s := net.Pipe()
		ep_c := r.newRouterEndPoint(name+string(i), c, hf)
		ep_s := r.newRouterEndPoint("client"+string(n), s, hf)
		r.AddEndPoint(ep_c)
		r.AddEndPoint(ep_s)
	}

	<-time.Tick(1 * time.Millisecond)
	testShareRouter(b, r, n)
}

func BenchmarkTCPSeperateRouter(b *testing.B) {
	server_r, err := NewRouter(nil, ServiceProcessPayload)
	if err != nil {
		b.FailNow()
	}
	client_r, err := NewRouter(nil, ServiceProcessPayload)
	if err != nil {
		b.FailNow()
	}

	hf := NewMsgHeaderFactory(NewProtobufFactory())

	server_r.Run()
	client_r.Run()
	<-time.Tick(1 * time.Millisecond)

	network := "tcp"
	address := "localhost:10001"
	if err := server_r.ListenAndServe("client", network, address, hf, ServiceProcessConn); err != nil {
		b.Log(err)
		b.FailNow()
	}

	name := "scheduler"
	n := 128
	for i := 0; i < n; i++ {
		if err := client_r.Dial(name+string(i), network, address, hf); err != nil {
			b.Log(err)
			b.FailNow()
		}
	}

	<-time.Tick(1 * time.Millisecond)
	testSeperateRouter(b, server_r, client_r, n)
}

func BenchmarkTCPShareRouter(b *testing.B) {
	r, err := NewRouter(nil, ServiceProcessPayload)
	if err != nil {
		b.FailNow()
	}

	hf := NewMsgHeaderFactory(NewProtobufFactory())

	r.Run()
	<-time.Tick(1 * time.Millisecond)

	network := "tcp"
	address := "localhost:10001"
	if err := r.ListenAndServe("client", network, address, hf, ServiceProcessConn); err != nil {
		b.Log(err)
		b.FailNow()
	}

	name := "scheduler"
	n := 128
	for i := 0; i < n; i++ {
		if err := r.Dial(name+string(i), network, address, hf); err != nil {
			b.Log(err)
			b.FailNow()
		}
	}

	<-time.Tick(1 * time.Millisecond)
	testShareRouter(b, r, n)
}

func testShareRouter(b *testing.B, r *Router, n int) {
	testSeperateRouter(b, r, r, n)
}

func testSeperateRouter(b *testing.B, server_r *Router, client_r *Router, n int) {
	name := "scheduler"
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := NewResourceReq()
			i := rand.Intn(n)
			//r.Call("scheduler", req, ClientProcessReponseIgnore, nil)
			if resp := client_r.CallWait(name+string(i), req, 0); resp == nil {
				b.Log("CallWait timeout")
				b.FailNow()
			}
		}
	})

	for i := 0; i < n; i++ {
		client_r.DelEndPoint(name + string(i))
		server_r.DelListener("client" + string(i))
	}

	client_r.Stop()
	server_r.Stop()
}

func BenchmarkTCPReadWrite(b *testing.B) {
	network := "tcp"
	address := "localhost:10009"

	l, err := net.Listen(network, address)
	if err != nil {
		b.FailNow()
	}
	c, err := net.Dial(network, address)
	if err != nil {
		b.FailNow()
	}
	s, err := l.Accept()
	if err != nil {
		b.FailNow()
	}

	ch_r := make(chan uint64, 1024)
	ch_w := make(chan uint64, 1024)
	ch_d := make(chan uint64, 1024)

	ch_c_w := make(chan uint64, 1024)
	ch_s_w := make(chan uint64, 1024)
	go func(s net.Conn, ch_r chan uint64) {
		for {
			bi1 := make([]byte, 16)
			s.Read(bi1)
			bi2 := make([]byte, 3)
			s.Read(bi2)
			ch_r <- 3
		}
	}(s, ch_r)
	go func(s net.Conn, ch_s_w chan uint64) {
		for {
			<-ch_s_w
			bo := make([]byte, 19)
			s.Write(bo)
		}
	}(s, ch_s_w)
	go func(c net.Conn, ch_r chan uint64) {
		for {
			bi1 := make([]byte, 16)
			c.Read(bi1)
			bi2 := make([]byte, 3)
			c.Read(bi2)
			ch_r <- 5
		}
	}(c, ch_r)
	go func(c net.Conn, ch_c_w chan uint64) {
		for {
			<-ch_c_w
			bo := make([]byte, 19)
			c.Write(bo)
		}
	}(c, ch_c_w)
	go func(ch_r chan uint64, ch_w chan uint64, ch_d chan uint64, ch_c_w chan uint64, ch_s_w chan uint64) {
		for {
			select {
			case id := <-ch_r:
				if id == 3 {
					ch_s_w <- 4
				} else if id == 5 {
					ch_d <- 6
				}
			case <-ch_w:
				ch_c_w <- 2
			}
		}
	}(ch_r, ch_w, ch_d, ch_c_w, ch_s_w)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ch_w <- 1
			<-ch_d
		}
	})
}

func BenchmarkPipeReadWrite(b *testing.B) {
	s, c := net.Pipe()

	ch_r := make(chan uint64, 1024)
	ch_w := make(chan uint64, 1024)
	ch_d := make(chan uint64, 1024)

	ch_c_w := make(chan uint64, 1024)
	ch_s_w := make(chan uint64, 1024)
	go func(s net.Conn, ch_r chan uint64) {
		for {
			bi1 := make([]byte, 16)
			s.Read(bi1)
			bi2 := make([]byte, 3)
			s.Read(bi2)
			ch_r <- 3
		}
	}(s, ch_r)
	go func(s net.Conn, ch_s_w chan uint64) {
		for {
			<-ch_s_w
			bo := make([]byte, 19)
			s.Write(bo)
		}
	}(s, ch_s_w)
	go func(c net.Conn, ch_r chan uint64) {
		for {
			bi1 := make([]byte, 16)
			c.Read(bi1)
			bi2 := make([]byte, 3)
			c.Read(bi2)
			ch_r <- 5
		}
	}(c, ch_r)
	go func(c net.Conn, ch_c_w chan uint64) {
		for {
			<-ch_c_w
			bo := make([]byte, 19)
			c.Write(bo)
		}
	}(c, ch_c_w)
	go func(ch_r chan uint64, ch_w chan uint64, ch_d chan uint64, ch_c_w chan uint64, ch_s_w chan uint64) {
		for {
			select {
			case id := <-ch_r:
				if id == 3 {
					ch_s_w <- 4
				} else if id == 5 {
					ch_d <- 6
				}
			case <-ch_w:
				ch_c_w <- 2
			}
		}
	}(ch_r, ch_w, ch_d, ch_c_w, ch_s_w)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ch_w <- 1
			<-ch_d
		}
	})
}

func BenchmarkChan(b *testing.B) {
	ch := make(chan uint64, 10240)

	go func(ch chan uint64) {
		for {
			select {
			case ch <- 1:
			case <-ch:
			}
		}
	}(ch)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ch <- 1
			<-ch
		}
	})
}
