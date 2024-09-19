package channeld

import (
	"time"

	"github.com/metaworking/channeld/pkg/channeldpb"
	"github.com/puzpuzpuz/xsync/v2"
	"go.uber.org/zap"
	"golang.org/x/exp/maps"
	"google.golang.org/protobuf/types/known/anypb"
)

type connectionRecoverHandle struct {
	disconnTime time.Time
	// Once set, the channels will start to send ChannelDataRecoveryMessage to the new connection.
	// TODO: make it a chan?
	newConn           *Connection
	startRecoveryTime time.Time
	// channelSubs       []recoverableChannelSub
}

// type recoverableChannelSub struct {
// 	channelId  common.ChannelId
// 	subTime    ChannelTime
// 	subOptions *channeldpb.ChannelSubscriptionOptions
// }

// Key: PIT
var connectionRecoverHandles *xsync.MapOf[string, *connectionRecoverHandle]

func (h *connectionRecoverHandle) IsTimeOut() bool {
	return GlobalSettings.ServerConnRecoverTimeoutMs > 0 &&
		time.Since(h.disconnTime) > time.Millisecond*time.Duration(GlobalSettings.ServerConnRecoverTimeoutMs)
}

func setConnectionRecoverable(conn *Connection) {
	handle := &connectionRecoverHandle{
		disconnTime: time.Now(),
	}
	connectionRecoverHandles.Store(conn.pit, handle)
	conn.recoverHandle = handle
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
			if time.Since(value.startRecoveryTime) > time.Millisecond*5000 {
				value.newConn.Send(MessageContext{
					MsgType:   channeldpb.MessageType_RECOVERY_END,
					Msg:       &channeldpb.EndRecoveryMesssage{},
					ChannelId: uint32(GlobalChannelId),
				})
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
			anyData, err := anypb.New(ch.GetDataMessage())
			if err != nil {
				ch.Logger().Error("failed to marshal channel data message for recovery", zap.Error(err))
				// No need to recover other subscriptions if the channel data is not corrputed.
				break
			}
			value.connHandle.newConn.Send(MessageContext{
				MsgType: channeldpb.MessageType_RECOVERY_CHANNEL_DATA,
				Msg: &channeldpb.ChannelDataRecoveryMessage{
					ChannelId:   uint32(ch.id),
					ChannelType: ch.channelType,
					Metadata:    ch.metadata,
					OwnerConnId: uint32(ch.GetOwner().Id()),
					SubTime:     int64(value.oldSubTime),
					SubOptions:  value.oldSubOptions,
					Data:        anyData,
				},
				ChannelId: uint32(ch.id),
			})
			delete(ch.recoverableSubs, key)
		}
	}
}
