channeld is a light-weight and efficient **messaging gateway** server designed for **distributed game servers** (typically MMO) 
and other backend applications that require real-time, subscription-based user interaction with high concurrency (e.g. an instance messenger).

## Features:
* Supports binary protocol via TCP/UDP and gRPC via HTTPS2
* Load-balancing backend servers with auto-scaling
* FSM-based message filtering
* Channel-based data subscription and storage of any type defined with Protocol Buffer
* Area of interest management based on channel and data subscriptions

## Binary protocol:
[-----------------------------------------------------------------------------------
CHNL(4B) | ChannelID(4B) | StubID(4B) | MessageType(4B) | BodySize(4B) | MessageBody
------------------------------------------------------------------------------------]
1. The first 4 bytes is the indentifier of a valid channeld packet ("CHNL" short for channel).
2. The second 4 bytes is an uint32 indicates which channel the message is sent to. 0 means the message should be handled in the global channel (e.g. a CreateChannel message).
3. The third 4 bytes is an uint32 indentifier as the RPC stub. 0 means the packet is not a RPC message.
4. Followed by the 4 bytes as an uint32 indicates the [message type](proto/message_types.proto).
5. Followed by the 4 bytes as the uint32 size of the remaining message body.
6. The remaining bytes is the message body, which is the marshalled data of a protobuf message.

## Performance

## Examples:
* Chat room
* Relay server
* Seamless world travelling
* Dynamic region load-balancing

## 设计原则：
1. 每个频道跑在不同的goroutine上；频道之间是线程隔离的，除了：
- 创建和删除频道；设置频道状态和所属连接。只能在主频道上处理。不需要对总频道列表加锁
- 频道的订阅回调消息会导致跨频道数据操作，需要对订阅列表加锁
 
2. channeld不假设客户端或服务端连接拥有不同的权限。这样是为了实现例如转发服务器这样没有游戏服务器的应用。
如果要控制客户端的访问权限，请使用连接的有限状态机来过滤消息。
例如：客户端在未验证时只能发送验证消息；在验证后只能发送用户自定义的消息（类型100以上）。通过这种方式，channeld就不会处理客户端发送的订阅和退订等消息。