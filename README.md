# rpc

This is a simple rpc framework.

Architecture
___

	Router
		|-EndPoint
			|-Reader
			|-Writer
		|-Listern

TODO
___
- Compatibility: 'grpc' register server feature
- BUG: It looks like huge concurrent requests with small buffered reader/writer will cause crash.
- Feature: EndPoint Notify(client close Router and report error.)
- Feature: writer timeout using time.Tick instead of using time.After
- Feature: Support Reader/Writer timeout(Defer/Deadline)
- Feature: Support Large Message.
- Feature: Error
- Feature: Log
- Feature: Route Rule
- Test: Call timeout
- Performance: Reuse EndPoint
- Performance: Reuse Reader/Writer
- Performance: Task Queue(May not useful, gc trace show there is no gc operation at all.)
- Performance: Use lower level event module/shareReader/shareWriter
- Security: transport
- Protocol: Http Message
- Protocol: Http 2.0
- Compatibility: grpc api
- Compatibility: grpc stream

Benchmark
---
grpc:
stream	requests	avg time
1		147035		66.52
2		259814		75.79
3		386114		76.67
4		410275		96.42
6		472222		125.89
8		480160		165.33 (*) <- rpc(buffered) faster from here

