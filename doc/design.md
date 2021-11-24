## 网游的架构演进
(TODO)

## 和其它类似技术的对比
(TODO)

BigWorld, Skynet, Photon, SpatialOS

## 将网络层作为独立的服务 vs. 整合进开发框架
整合进开发框架的缺点：
- 固定的开发语言
- 往往对游戏的业务模型存在一些假设而难以通用。例如，MOBA类型往往使用帧同步的框架
- 往往难以扩容，或是将游戏业务层的扩容和网络层的扩容耦合在了一起。例如，从1个游戏房间扩容到100个，意味着需要增加99个公共端点(endpoint)

独立的服务的优点：
- 开发语言无关。只要实现了通信协议即可，或者直接使用SDK进行调用
- 对各种游戏类型更通用。
- 容灾性更好。业务层的代码问题导致的进程崩溃，不会影响网络服务。通过负载均衡方案可以热实现灾难恢复。
- 客户端保持连接，体验更平滑。从大厅到房间，或从地图到另一个地图，客户端不需要重新建立连接，而且数据能够持续地同步到客户端。

## 为什么不用Redis来实现类似的功能
- Redis的编码协议RESP基于字符串，而游戏业务中的大部分数据不是字符串
- 复杂的数据结构需要自己用C扩展，且传输仍基于RESP
- 只能基于TCP，不适用于移动设备的网络场景
- 通讯方式基于请求-返回，不支持异步，而游戏中的大部分业务不能阻塞线程，需要用异步的方式实现
- PUB/SUB模式下更是会阻塞住其它命令请求
- PUB/SUB的基准测试低于单核1000 rps。channeld的目标是数倍。

## 竞态和权限问题：
1. 连接列表可能被各个连接和频道goroutine写，需要加锁；
2. 每个频道跑在不同的goroutine上；频道之间是隔离的，除了：
- 创建和删除频道；设置频道状态和所属连接。只能在主频道上处理。任何一个连接收取消息时都需要查询频道，所以需要对总频道列表加读写锁
- 频道的订阅回调消息会导致跨频道数据操作，需要对订阅列表加锁
3. channeld不假设客户端或服务端连接拥有不同的权限。这样是为了实现例如转发服务器这样没有游戏服务器的应用。
如果要控制客户端的访问权限，请使用连接的有限状态机来过滤消息。
例如：客户端在未验证时只能发送验证消息；在验证后只能发送用户自定义的消息（类型100以上）。通过这种方式，channeld就不会处理客户端发送的订阅和退订等消息。


## Binary protocol:
[TAG] [Packet [ChannelID | BroadcastType | StubID | MessageType | MessageBody]
1. The tag consists of 4 bytes. The first byte must be 67 which is the ASCII of 'C' character. The 2-4 bytes are the "dynamic" size of the packet, which means if the size is less than 65536(2^16), the second byte is 72('H' in ASCII), otherwise the byte is used for the size; if the size is less than 256(2^8), the third byte is 78('L' in ASCII), otherwise the byte is used for the size; the fourth and last byte is always used for the size. So, if the packet size is less than 256, which is most of the case, the TAG bytes are: [67 72 78 SIZE]
2. The following are the marshalled bytes of the Packet (see the message definition in [channeld.proto](../proto/channeld.proto)). The header of a packet includes an uint32 ChannelID, an enum BroadcastType, an uint32 StubId, and an uint32 MessageType. Because it utilizes [Protobuf's encoding](https://developers.google.com/protocol-buffers/docs/encoding), in most cases the header only has 4 bytes (see the BenchmarkProtobufPacket in [message_test.go](../pkg/channeld/message_test.go))
3. The message body is the marshalled bytes of the actual message that channeld will proceed or forward.

## Goroutines
### IO
- Client listner.Accept()
- Server listner.Accept()
### Connection
- (Per connection) Connection.Receive()
- (Per connection) Connection.Flush()
### Channel
- (Per channel) Channel.Tick()

## How channel data updates are fanned out
U = sends channel data update message to channeld

F = channeld sends accumulated update message to a subscribed connection

Server Connection: (C0) ------U1----U2--------U3---------

Client Connection1:(C1) ----F1---F2---F3---F4---F5---F6--

Client Connection2:(C2)   --F7--------F8--------F9-------

Fan out interval of C1 = 50ms, C2 = 100ms
Time of U1 = 60ms, U2 = 120ms, U3 = 240ms, F1/F7 = 50ms, F2 = 100ms, F3/F8 = 150ms...

F1 = nil(no send), F2 = U1, F3 = U2, F4 = nil, F5 = U3, F6 = nil
F7 = the whole data (as the client connection2 just subscribed), F8 = U1+U2, F9 = U3

Assuming there are n connections, and each U has m properties in average. There are several ways to implement the fan-out:

A. Each connection contains its own fan-out message
- Space complexity: O(n)*O(m)
- Time complexity: O(n)*O(m)

B. Store all the U in the channel. Sort the connections by the lastFanOutTime, and then send the accumulated update message to each connection.
- Space complexity: O(1)*O(m)
- Time complexity: O(nlog(n)) + O(n)*O(m)
