package inspector

import (
	"sync"
	"time"

	"github.com/channeldorg/channeld/pkg/channeld"
	"github.com/channeldorg/channeld/pkg/channeldpb"
	"github.com/channeldorg/channeld/pkg/common"
	"go.uber.org/zap"
)

var (
	inspectorConnections     sync.Map // map[*channeld.Connection]*InspectorConnection
	inspectorConnectionsLock sync.RWMutex
)

// InitInspector initializes Inspector event listeners
func InitInspector() {
	// Listen to channel events
	channeld.Event_ChannelCreated.Listen(onChannelCreated)
	channeld.Event_ChannelRemoved.Listen(onChannelRemoved)

	// Start periodic full sync for subscribed connections
	go startPeriodicFullSync()
}

// onChannelCreated handles channel creation event
func onChannelCreated(ch *channeld.Channel) {
	info := GetChannelInfo(ch.Id())
	if info == nil {
		return
	}

	// Use DebugChannelUpdatedEvent for now (will be changed to DebugChannelAddedEvent after proto generation)
	event := &channeldpb.DebugChannelUpdatedEvent{
		ChannelId:       uint32(info.ChannelID),
		ChannelType:     info.ChannelType,
		OwnerConnId:     uint32(info.OwnerConnID),
		SubscriberCount: uint32(info.SubscriberCount),
	}

	broadcastToInspectors(channeldpb.MessageType_DEBUG_CHANNEL_UPDATED, event)
}

// onChannelRemoved handles channel removal event
func onChannelRemoved(channelID common.ChannelId) {
	// Use DebugChannelUpdatedEvent for now (will be changed to DebugChannelRemovedEvent after proto generation)
	event := &channeldpb.DebugChannelUpdatedEvent{
		ChannelId: uint32(channelID),
	}

	broadcastToInspectors(channeldpb.MessageType_DEBUG_CHANNEL_UPDATED, event)
}

// broadcastToInspectors broadcasts an event to all subscribed Inspector connections
func broadcastToInspectors(msgType channeldpb.MessageType, msg common.Message) {
	inspectorConnections.Range(func(key, value interface{}) bool {
		conn := key.(*channeld.Connection)
		inspectorConn := value.(*InspectorConnection)

		if conn.IsClosing() || !inspectorConn.subscribedToList {
			return true
		}

		ctx := channeld.MessageContext{
			MsgType:    msgType,
			Msg:        msg,
			ChannelId:  0, // Events use GLOBAL channel
			Connection: conn,
		}
		conn.Send(ctx)
		return true
	})
}

// startPeriodicFullSync starts a goroutine that periodically sends full snapshots
func startPeriodicFullSync() {
	ticker := time.NewTicker(5 * time.Minute) // Default 5 minutes
	defer ticker.Stop()

	for range ticker.C {
		inspectorConnections.Range(func(key, value interface{}) bool {
			conn := key.(*channeld.Connection)
			inspectorConn := value.(*InspectorConnection)

			if conn.IsClosing() || !inspectorConn.subscribedToList || !inspectorConn.fullSyncEnabled {
				return true
			}

			// Send full snapshot
			allChannels := GetAllChannels()
			pbChannels := make([]*channeldpb.DebugChannelInfo, len(allChannels))
			for i, ch := range allChannels {
				pbChannels[i] = &channeldpb.DebugChannelInfo{
					ChannelId:       uint32(ch.ChannelID),
					ChannelType:     ch.ChannelType,
					OwnerConnId:     uint32(ch.OwnerConnID),
					SubscriberCount: uint32(ch.SubscriberCount),
					Metadata:        ch.Metadata,
					CreatedTime:     ch.CreatedTime,
				}
			}

			event := &channeldpb.DebugChannelListSnapshotEvent{
				Channels:     pbChannels,
				Total:        uint32(len(allChannels)),
				SnapshotTime: time.Now().UnixMilli(),
			}

			ctx := channeld.MessageContext{
				MsgType:    channeldpb.MessageType_DEBUG_CHANNEL_LIST_SNAPSHOT,
				Msg:        event,
				ChannelId:  0,
				Connection: conn,
			}
			conn.Send(ctx)
			return true
		})
	}
}

// RegisterInspectorConnection registers an Inspector connection
func RegisterInspectorConnection(conn *channeld.Connection) {
	inspectorConn := GetInspectorConnection(conn)
	inspectorConnections.Store(conn, inspectorConn)
	conn.Logger().Info("Inspector connection registered")
}

// UnregisterInspectorConnection unregisters an Inspector connection
func UnregisterInspectorConnection(conn *channeld.Connection) {
	inspectorConnections.Delete(conn)
	conn.Logger().Info("Inspector connection unregistered")
}

// HandleDebugSubscribeChannelList handles subscription request
func HandleDebugSubscribeChannelList(ctx channeld.MessageContext) {
	req, ok := ctx.Msg.(*channeldpb.DebugSubscribeChannelListRequest)
	if !ok {
		ctx.Connection.Logger().Error("message is not DebugSubscribeChannelListRequest")
		return
	}

	conn, ok := ctx.Connection.(*channeld.Connection)
	if !ok {
		return
	}

	value, exists := inspectorConnections.Load(conn)
	if !exists {
		RegisterInspectorConnection(conn)
		value, _ = inspectorConnections.Load(conn)
	}

	inspectorConn := value.(*InspectorConnection)
	inspectorConn.subscribedToList = true
	inspectorConn.fullSyncEnabled = req.EnableFullSync
	if req.FullSyncIntervalSec > 0 {
		inspectorConn.fullSyncInterval = int(req.FullSyncIntervalSec)
	}

	conn.Logger().Info("Inspector subscribed to channel list updates",
		zap.Bool("fullSyncEnabled", inspectorConn.fullSyncEnabled),
		zap.Int("fullSyncInterval", inspectorConn.fullSyncInterval),
	)
}

// HandleDebugUnsubscribeChannelList handles unsubscription request
func HandleDebugUnsubscribeChannelList(ctx channeld.MessageContext) {
	conn, ok := ctx.Connection.(*channeld.Connection)
	if !ok {
		return
	}

	value, exists := inspectorConnections.Load(conn)
	if !exists {
		return
	}

	inspectorConn := value.(*InspectorConnection)
	inspectorConn.subscribedToList = false

	conn.Logger().Info("Inspector unsubscribed from channel list updates")
}
