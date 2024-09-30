package channeld

import (
	"time"

	"github.com/metaworking/channeld/pkg/channeldpb"
	"github.com/puzpuzpuz/xsync/v2"
	"go.uber.org/zap"
	"golang.org/x/exp/maps"
	"google.golang.org/protobuf/types/known/anypb"
)

// Time to wait for all channels to send ChannelDataRecoveryMessage to the new connection, before sending EndRecoveryMesssage.
// This value should not be smaller than the tick interval of any recoverable channel.
const ChannelDataRecoveryTimeout = time.Millisecond * 1000

type connectionRecoverHandle struct {
	prevConnId  ConnectionId
	disconnTime time.Time
	// Once set, the channels will start to send ChannelDataRecoveryMessage to the new connection.
	// TODO: make it a chan?
	newConn           *Connection
	startRecoveryTime time.Time
}

// Key: PIT
var connectionRecoverHandles *xsync.MapOf[string, *connectionRecoverHandle]

func (h *connectionRecoverHandle) IsTimeOut() bool {
	return GlobalSettings.ServerConnRecoverTimeoutMs > 0 &&
		time.Since(h.disconnTime) > time.Millisecond*time.Duration(GlobalSettings.ServerConnRecoverTimeoutMs)
}

func (conn *Connection) makeRecoverable() {
	handle := &connectionRecoverHandle{
		prevConnId:  conn.Id(),
		disconnTime: time.Now(),
	}
	connectionRecoverHandles.Store(conn.pit, handle)
	conn.recoverHandle = handle
}

func (c *Connection) ShouldRecover() bool {
	return c.recoverHandle != nil
}

func (c *Connection) RecoverFromHandle(handle *connectionRecoverHandle) {

	// Update the connection with the previous connection id if the previous connection id is not used.
	_, prevIdExists := allConnections.LoadAndDelete(handle.prevConnId)
	if !prevIdExists {
		c.Logger().Info("recover the connection with the previous connection id", zap.Uint32("prevConnId", uint32(handle.prevConnId)))
		c.id = handle.prevConnId
		allConnections.Store(c.id, c)
	} else {
		c.Logger().Error("failed to recover the connection as the previous connection id is used")
		return
	}

	c.recoverHandle = handle
	c.recoverHandle.newConn = c
	c.recoverHandle.startRecoveryTime = time.Now()
}

func tickConnectionRecovery() {
	for {
		connectionRecoverHandles.Range(func(key string, value *connectionRecoverHandle) bool {
			if value.IsTimeOut() {
				connectionRecoverHandles.Delete(key)
				return true
			}

			if value.newConn == nil {
				return true
			}

			// Wait for all channels to send the ChannelDataRecoveryMessage
			if time.Since(value.startRecoveryTime) > ChannelDataRecoveryTimeout {
				value.newConn.Send(MessageContext{
					MsgType:   channeldpb.MessageType_RECOVERY_END,
					Msg:       &channeldpb.EndRecoveryMesssage{},
					ChannelId: uint32(GlobalChannelId),
				})
				value.newConn.recoverHandle = nil
				connectionRecoverHandles.Delete(key)
				return true
			}

			return true
		})

		time.Sleep(time.Millisecond * 1000)
	}
}

func (ch *Channel) tickRecoverableSubscriptions() {
	for key, value := range ch.recoverableSubs {
		if value.connHandle.IsTimeOut() {
			maps.Clear(ch.recoverableSubs)
			if GlobalSettings.GetChannelSettings(ch.channelType).RemoveChannelAfterOwnerRemoved {
				removeChannelAfterOwnerRemoved(ch)
			}
			break
		}

		if value.connHandle.newConn != nil {
			if value.isOwner {
				if ch.HasOwner() {
					ch.Logger().Warn("failed to restore the owner of the channel",
						zap.Uint32("newOwnerConnId", uint32(ch.GetOwner().Id())),
						zap.Uint32("oldOwnerConnId", uint32(value.connHandle.newConn.Id())))
				} else {
					ch.SetOwner(value.connHandle.newConn)
					ch.Logger().Info("restored the owner of the channel", zap.Uint32("ownerConnId", uint32(ch.GetOwner().Id())))
				}
			}
			// Always skip the first fan out
			value.oldSubOptions.SkipFirstFanOut = Pointer(true)
			value.connHandle.newConn.SubscribeToChannel(ch, value.oldSubOptions)
			// Set the old subOptions to the recover handle so that the connection can recover the subscription.
			// value.connHandle.channelSubs = append(value.connHandle.channelSubs,
			// 	recoverableChannelSub{
			// 		channelId:  ch.id,
			// 		subTime:    value.oldSubTime,
			// 		subOptions: value.oldSubOptions,
			// 	},
			// )
			channelData, err := anypb.New(ch.GetDataMessage())
			if err != nil {
				ch.Logger().Error("failed to marshal channel full data message for recovery", zap.Error(err))
				// No need to recover other subscriptions if the channel data is corrputed.
				break
			}
			recoveryMsg := &channeldpb.ChannelDataRecoveryMessage{
				ChannelId:   uint32(ch.id),
				ChannelType: ch.channelType,
				Metadata:    ch.metadata,
				SubTime:     value.oldSubTime,
				SubOptions:  value.oldSubOptions,
				ChannelData: channelData,
			}
			if ch.HasOwner() {
				recoveryMsg.OwnerConnId = uint32(ch.GetOwner().Id())
			}
			if ch.Data().Extension() != nil {
				recoveryData, err := anypb.New(ch.Data().Extension().GetRecoveryDataMessage())
				if err != nil {
					ch.Logger().Error("failed to marshal channel recovery data message for recovery", zap.Error(err))
					// No need to recover other subscriptions if the spawn data is corrputed.
					break
				}
				recoveryMsg.RecoveryData = recoveryData
			}
			value.connHandle.newConn.Send(MessageContext{
				MsgType:   channeldpb.MessageType_RECOVERY_CHANNEL_DATA,
				Msg:       recoveryMsg,
				ChannelId: uint32(ch.id),
			})
			delete(ch.recoverableSubs, key)

			if GlobalSettings.GetChannelSettings(ch.channelType).SendOwnerLostAndRecovered {
				// The mesage should be sent after EndRecoveryMesssage is sent.
				go func() {
					time.Sleep(ChannelDataRecoveryTimeout)
					ch.Broadcast(MessageContext{
						MsgType:   channeldpb.MessageType_CHANNEL_OWNER_RECOVERED,
						Msg:       &channeldpb.ChannelOwnerRecoveredMessage{},
						Broadcast: uint32(channeldpb.BroadcastType_ALL_BUT_OWNER),
						ChannelId: uint32(ch.id),
					})
				}()
			}
		}
	}
}
