# What
## 概念
channeld主要有三个基本概念：Connection 连接，Channel 频道，ChannelData 频道数据。
### Connection
每一个建立到channeld服务的外部连接，都是一个Connection。主要有两类：客户端和服务端。在某些应用场景下，如中转服务，只有客户端连接，没有后端的专用服务器。

开发者可以为每类连接配置一个有限状态机，指定某种状态下的消息类型白名单和黑名单。这是channeld提供的基本访问控制机制。

### Channel
Channel可以理解为一个兴趣组，聚合了多个连接的订阅。channeld预制的频道类型包括：
- 全局频道。系统在启动后就会自动创建一个唯一的全局频道。所有非频道相关的消息，如：验证，创建或删除频道，都会在全局频道处理。也可用于全局广播
- 私有频道。每个连接可以把自己公开的数据放到这个频道，以供其它连接订阅。如：玩家的等级和基本装备信息。也可以通过这个频道进行一对一聊天
- 子世界频道。每个子世界是一个独立的、隔离的空间。子世界中的订阅者可以互相观察。适用于游戏房间或空间上隔离的游戏场景
- 空间频道。如果游戏服务器需要将玩家分布到不同的子空间来进行负载均衡，并且在不同子空间之间可以无缝移动和交互，就需要用到空间频道。开发者需要实现空间位置到频道ID的映射逻辑

开发者可以通过修改[channeld.proto](../proto/channeld.proto)来扩展频道类型。

每个频道都有一个所有者。它一般是创建频道的那个连接。在权威服务器(Server authoritative)的架构中，频道所有者往往是后端的游戏服务器。它们控制着客户端到频道的订阅，消息的广播等。

### ChannelData
频道数据是订阅的核心，也就是兴趣数据。频道数据的修改，会通过[扇出 Fan-out](https://en.wikipedia.org/wiki/Fan-out_(software))的形式发送给所有订阅的连接。

每个连接可以设置自己扇出的最小间隔时间。通过这种方式，开发者可以控制现客户端对不同的兴趣数据的订阅频率。如：组队和聊天数据的同步频率较低，玩家位置的同步频率较高。

# Why
## 网游的架构演进
(TODO)

## 和其它类似技术的对比
(TODO)

BigWorld, Skynet, Photon, SpatialOS

## 将网络层作为独立的服务 vs. 整合进开发框架
整合进开发框架的缺点：
- 固定的开发语言
- 往往对游戏的业务模型存在一些假设而难以通用。例如，MOBA类型往往使用房间+帧同步的框架；而MMORPG使用分布式场景+状态同步的框架
- 往往难以扩容，或是将游戏业务层的扩容和网络层的扩容耦合在了一起。例如，从1个游戏房间扩容到100个，意味着需要增加99个公共端点(endpoint)

独立的服务的优点：
- 开发语言无关。只要实现了通信协议即可，或者直接使用SDK进行调用
- 对各种游戏类型更通用。
- 容灾性更好。业务层的代码问题导致的进程崩溃，不会影响网络服务。通过负载均衡方案可以热实现灾难恢复。
- 客户端保持连接，体验更平滑。从大厅到房间，或从地图到另一个地图，客户端不需要重新建立连接，而且数据能够持续地同步到客户端。

## 为什么不用Redis来实现网络层
- Redis的编码协议RESP基于字符串，而游戏业务中的大部分数据不是字符串
- 复杂的数据结构需要自己用C扩展，且传输仍基于RESP
- 只能基于TCP，不适用于移动设备的网络场景
- 通讯方式基于请求-返回模式，不支持异步，而游戏中的大部分业务不能阻塞线程，需要用异步的方式实现
- PUB/SUB模式下更是会阻塞住其它命令请求
- PUB/SUB的基准测试低于单核1000 rps。channeld的目标是至少高一个数量级。

# How
## Binary protocol:
[TAG] [[MessagePack0 [ChannelID | BroadcastType | StubID | MessageType | MessageBody] | MessagePack1 | MessagePack2 ...]
1. A packet consists of a TAG, and a serial of MessagePacks (see the definition in [channeld.proto](../proto/channeld.proto))
2. The tag has 4 bytes. The first byte must be 67 which is the ASCII of 'C' character. The 2-4 bytes are the "dynamic" size of the packet, which means if the size is less than 65536(2^16), the second byte is 72('H' in ASCII), otherwise the byte is used for the size; if the size is less than 256(2^8), the third byte is 78('L' in ASCII), otherwise the byte is used for the size; the fourth and last byte is always used for the size. So, if the packet size is less than 256, which is most of the case, the TAG bytes are: [67 72 78 SIZE]
3. Each MessagePack consists of a header and a body. The header includes an uint32 ChannelID, an enum BroadcastType, an uint32 StubId, and an uint32 MessageType. Because it utilizes [Protobuf's encoding](https://developers.google.com/protocol-buffers/docs/encoding), in most cases the header only has 4 bytes (see *BenchmarkProtobufMessageBase* in [message_test.go](../pkg/channeld/message_test.go))
4. The message body is the marshalled bytes of the actual message that channeld will proceed or forward.

## 竞态和权限问题：
1. 连接列表可能被各个连接和频道goroutine写，需要加锁；
2. 每个频道跑在不同的goroutine上；频道之间是隔离的，除了：
- 创建和删除频道；设置频道状态和所属连接。只能在主频道上处理。任何一个连接收取消息时都需要查询频道，所以需要对总频道列表加读写锁
- 频道的订阅回调消息会导致跨频道数据操作，需要对订阅列表加锁
3. channeld不假设客户端或服务端连接拥有不同的权限。这样是为了实现例如转发服务器这样没有游戏服务器的应用。
如果要控制客户端的访问权限，请使用连接的有限状态机来过滤消息。
例如：客户端在未验证时只能发送验证消息；在验证后只能发送用户自定义的消息（类型100以上）。通过这种方式，channeld就不会处理客户端发送的订阅和退订等消息。

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
