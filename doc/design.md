
## 设计原则：
1. 每个频道跑在不同的goroutine上；频道之间是线程隔离的，除了：
- 创建和删除频道；设置频道状态和所属连接。只能在主频道上处理。不需要对总频道列表加锁
- 频道的订阅回调消息会导致跨频道数据操作，需要对订阅列表加锁
 
2. channeld不假设客户端或服务端连接拥有不同的权限。这样是为了实现例如转发服务器这样没有游戏服务器的应用。
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



## How channel data updates are fanned out{#fan-out}
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
Space complexity: O(n)*O(m)
Time complexity: O(n)*O(m)

B. Store all the U in the channel. Sort the connections by the lastFanOutTime, and then send the accumulated update message to each connection.
Space complexity: O(1)*O(m)
Time complexity: O(nlog(n)) + O(n)*O(m)
