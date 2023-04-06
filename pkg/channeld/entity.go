package channeld

import (
	"sync"

	"github.com/metaworking/channeld/pkg/channeldpb"
	"github.com/metaworking/channeld/pkg/common"
	"github.com/puzpuzpuz/xsync/v2"
	"go.uber.org/zap"
)

type EntityId uint32

// type EntityChannel struct {
// 	*Channel
// }

type EntityGroup struct {
	entityIds *xsync.MapOf[EntityId, interface{}] //[]EntityId
}

func newEntityGroup() *EntityGroup {
	return &EntityGroup{
		entityIds: xsync.NewTypedMapOf[EntityId, interface{}](UintIdHasher[EntityId]()),
	}
}

type EntityGroupController interface {
	SetGroup(t channeldpb.EntityGroupType, group *EntityGroup)
	AddToGroup(t channeldpb.EntityGroupType, junction EntityId, entitiesToAdd []EntityId) error
	RemoveFromGroup(t channeldpb.EntityGroupType, entitiesToRemove []EntityId) error
	GetHandoverEntities() []EntityId
}

type EntityChannelGroupController struct {
	rwLock        sync.RWMutex
	handoverGroup *EntityGroup
	lockGroup     *EntityGroup
}

func (ctl *EntityChannelGroupController) SetGroup(t channeldpb.EntityGroupType, group *EntityGroup) {
	ctl.rwLock.Lock()
	defer ctl.rwLock.Unlock()

	if t == channeldpb.EntityGroupType_HANDOVER {
		ctl.handoverGroup = group
	} else if t == channeldpb.EntityGroupType_LOCK {
		ctl.lockGroup = group
	}
}

func (ctl *EntityChannelGroupController) AddToGroup(t channeldpb.EntityGroupType, junction EntityId, entitiesToAdd []EntityId) error {
	if t == channeldpb.EntityGroupType_HANDOVER {
		if ctl.handoverGroup == nil {
			ctl.handoverGroup = newEntityGroup()
		}

		ctl.handoverGroup.entityIds.Store(junction, nil)

		for _, entityId := range entitiesToAdd {
			ctl.handoverGroup.entityIds.Store(entityId, nil)

			ch := GetChannel(common.ChannelId(entityId))
			if ch == nil {
				continue
			}

			if ch.entityController == nil {
				ch.Logger().Error("channel doesn't have the entity controller")
				continue
			}

			// All entity channels of the same group use the same handover group
			ch.entityController.SetGroup(t, ctl.handoverGroup)
		}
	}

	return nil
}

func (ctl *EntityChannelGroupController) RemoveFromGroup(t channeldpb.EntityGroupType, entitiesToRemove []EntityId) error {
	return nil
}

func (ctl *EntityChannelGroupController) GetHandoverEntities() []EntityId {
	if ctl.handoverGroup == nil {
		return make([]EntityId, 0)
	}

	ctl.rwLock.RLock()
	defer ctl.rwLock.RUnlock()

	arr := make([]EntityId, 0, ctl.handoverGroup.entityIds.Size())
	ctl.handoverGroup.entityIds.Range(func(key EntityId, _ interface{}) bool {
		arr = append(arr, key)
		return true
	})
	return arr
}

func (ch *Channel) GetHandoverEntities(notifyingEntityId EntityId) map[EntityId]common.Message {
	if ch.entityController == nil {
		ch.Logger().Error("channel doesn't have the entity controller")
		return nil
	}

	entityIds := ch.entityController.GetHandoverEntities()
	entities := make(map[EntityId]common.Message, len(entityIds))
	for _, entityId := range entityIds {
		entityChannel := GetChannel(common.ChannelId(entityId))
		if entityChannel == nil {
			entities[entityId] = nil
			continue
		}
		entities[entityId] = entityChannel.GetDataMessage()
	}

	return entities
}

func handleAddEntityGroup(ctx MessageContext) {
	addMsg, ok := ctx.Msg.(*channeldpb.AddEntityGroupMessage)
	if !ok {
		ctx.Connection.Logger().Error("message is not an AddEntityGroupMessage, will not be handled.")
		return
	}

	if addMsg.JunctionEntityId != ctx.ChannelId {
		ctx.Connection.Logger().Error("AddEntityGroupMessage should be handled in the junction entity channel",
			zap.Uint32("junctionEntityId", addMsg.JunctionEntityId), zap.Uint32("channelId", ctx.ChannelId))
		return
	}

	if ctx.Channel.entityController == nil {
		ctx.Channel.Logger().Error("channel doesn't have the entity controller")
		return
	}

	if ctx.Channel.entityController.AddToGroup(addMsg.Type, EntityId(addMsg.JunctionEntityId), CopyArray[uint32, EntityId](addMsg.EntitiesToAdd)) != nil {
		ctx.Channel.Logger().Error("failed to add entities to group",
			zap.Int32("groupType", int32(addMsg.Type)),
			zap.Uint32("junctionEntityId", addMsg.JunctionEntityId),
			zap.Uint32s("entitiesToAdd", addMsg.EntitiesToAdd),
		)
	}
}

func handleRemoveEntityGroup(ctx MessageContext) {

}
