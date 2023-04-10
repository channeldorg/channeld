package channeld

import (
	"fmt"
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

func (g *EntityGroup) Add(groupToAdd *EntityGroup) {
	if groupToAdd == nil {
		return
	}

	groupToAdd.entityIds.Range(func(key EntityId, value interface{}) bool {
		g.entityIds.Store(key, value)
		return true
	})
}

func toArray[T UintId](g *EntityGroup) []T {
	arr := []T{}
	g.entityIds.Range(func(key EntityId, value interface{}) bool {
		arr = append(arr, T(key))
		return true
	})
	return arr
}

type EntityGroupController interface {
	cascadeGroup(t channeldpb.EntityGroupType, group *EntityGroup)
	AddToGroup(t channeldpb.EntityGroupType, entitiesToAdd []EntityId) error
	RemoveFromGroup(t channeldpb.EntityGroupType, entitiesToRemove []EntityId) error
	GetHandoverEntities() []EntityId
}

// FlatEntityGroupController is a simple implementation of EntityGroupController which has
// only one layer of handover and lock group. Adding an entities in a group to another group
// will overwrite the previous group and the overwrite is not revertible.
type FlatEntityGroupController struct {
	entityId      EntityId
	rwLock        sync.RWMutex
	handoverGroup *EntityGroup
	lockGroup     *EntityGroup
}

func (ctl *FlatEntityGroupController) cascadeGroup(t channeldpb.EntityGroupType, group *EntityGroup) {
	// Current entity is already locked, won't cascade.
	if ctl.lockGroup != nil && ctl.lockGroup.entityIds.Size() > 0 {
		return
	}

	ctl.rwLock.Lock()
	defer ctl.rwLock.Unlock()

	if t == channeldpb.EntityGroupType_HANDOVER {
		// Add the entities in current group to the new group
		group.Add(ctl.handoverGroup)

		// Set current group to the new group
		ctl.handoverGroup = group
	} else if t == channeldpb.EntityGroupType_LOCK {
		// LOCK has higher priority than HANDOVER, so the cascade brings the entities in the handover group into the lock group
		group.Add(ctl.handoverGroup)
		group.Add(ctl.lockGroup)

		// Set current group to the new group
		ctl.lockGroup = group
	}
}

func (ctl *FlatEntityGroupController) AddToGroup(t channeldpb.EntityGroupType, entitiesToAdd []EntityId) error {
	if t == channeldpb.EntityGroupType_HANDOVER {
		if ctl.handoverGroup == nil {
			ctl.handoverGroup = newEntityGroup()
		}

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

			// All entity channels of the same group share the same handover group instance
			ch.entityController.cascadeGroup(t, ctl.handoverGroup)
		}

		rootLogger.Debug("updated handover group", zap.Uint32("entityId", uint32(ctl.entityId)),
			zap.Any("handoverGroup", toArray[EntityId](ctl.handoverGroup)))

	} else if t == channeldpb.EntityGroupType_LOCK {
		if ctl.lockGroup == nil {
			ctl.lockGroup = newEntityGroup()
		}

		for _, entityId := range entitiesToAdd {
			ctl.lockGroup.entityIds.Store(entityId, nil)

			ch := GetChannel(common.ChannelId(entityId))
			if ch == nil {
				continue
			}

			if ch.entityController == nil {
				ch.Logger().Error("channel doesn't have the entity controller")
				continue
			}

			// All entity channels of the same group share the same lock group instance
			ch.entityController.cascadeGroup(t, ctl.lockGroup)
		}

		rootLogger.Debug("updated lock group", zap.Uint32("entityId", uint32(ctl.entityId)),
			zap.Any("lockGroup", toArray[EntityId](ctl.lockGroup)))
	}

	return nil
}

func (ctl *FlatEntityGroupController) RemoveFromGroup(t channeldpb.EntityGroupType, entitiesToRemove []EntityId) error {
	if t == channeldpb.EntityGroupType_HANDOVER {
		if ctl.handoverGroup != nil {
			for _, entityId := range entitiesToRemove {
				ctl.handoverGroup.entityIds.Delete(entityId)
				// Reset the removed entity's entity channel's handover group
				entityCh := GetChannel(common.ChannelId(entityId))
				if entityCh != nil {
					entityCh.entityController.(*FlatEntityGroupController).handoverGroup = newEntityGroup()
				}
			}
		} else {
			return fmt.Errorf("handover group is nil, entityId: %d", ctl.entityId)
		}
	} else if t == channeldpb.EntityGroupType_LOCK {
		if ctl.lockGroup != nil {
			for _, entityId := range entitiesToRemove {
				ctl.lockGroup.entityIds.Delete(entityId)
				// Reset the removed entity's entity channel's lock group
				entityCh := GetChannel(common.ChannelId(entityId))
				if entityCh != nil {
					entityCh.entityController.(*FlatEntityGroupController).lockGroup = newEntityGroup()
				}
			}
		} else {
			return fmt.Errorf("lock group is nil, entityId: %d", ctl.entityId)
		}
	}

	return nil
}

func (ctl *FlatEntityGroupController) GetHandoverEntities() []EntityId {
	// If AddToGroup is never called, return the entity itself
	if ctl.handoverGroup == nil {
		return []EntityId{ctl.entityId}
	}

	ctl.rwLock.RLock()
	defer ctl.rwLock.RUnlock()

	arr := make([]EntityId, 0, ctl.handoverGroup.entityIds.Size())
	locked := false
	ctl.handoverGroup.entityIds.Range(func(key EntityId, _ interface{}) bool {
		if ctl.lockGroup != nil {
			// If any entity in the handover group is locked, the handover should not happen.
			if _, locked = ctl.lockGroup.entityIds.Load(key); locked {
				return false
			}
		}
		arr = append(arr, key)
		return true
	})

	if locked {
		return []EntityId{}
	}

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
	if ctx.Connection != ctx.Channel.ownerConnection {
		ctx.Connection.Logger().Error("only the owner connection of the entity channel can handle the message.")
		return
	}

	addMsg, ok := ctx.Msg.(*channeldpb.AddEntityGroupMessage)
	if !ok {
		ctx.Connection.Logger().Error("message is not an AddEntityGroupMessage, will not be handled.")
		return
	}

	if ctx.Channel.entityController == nil {
		ctx.Channel.Logger().Error("channel doesn't have the entity controller")
		return
	}

	if ctx.Channel.entityController.AddToGroup(addMsg.Type, CopyArray[uint32, EntityId](addMsg.EntitiesToAdd)) != nil {
		ctx.Channel.Logger().Error("failed to add entities to group",
			zap.Int32("groupType", int32(addMsg.Type)),
			zap.Uint32s("entitiesToAdd", addMsg.EntitiesToAdd),
		)
	}
}

func handleRemoveEntityGroup(ctx MessageContext) {
	if ctx.Connection != ctx.Channel.ownerConnection {
		ctx.Connection.Logger().Error("only the owner connection of the entity channel can handle the message.")
		return
	}

	removeMsg, ok := ctx.Msg.(*channeldpb.RemoveEntityGroupMessage)
	if !ok {
		ctx.Connection.Logger().Error("message is not an RemoveEntityGroupMessage, will not be handled.")
		return
	}

	if ctx.Channel.entityController == nil {
		ctx.Channel.Logger().Error("channel doesn't have the entity controller")
		return
	}

	if ctx.Channel.entityController.RemoveFromGroup(removeMsg.Type, CopyArray[uint32, EntityId](removeMsg.EntitiesToRemove)) != nil {
		ctx.Channel.Logger().Error("failed to remove entities from group",
			zap.Int32("groupType", int32(removeMsg.Type)),
			zap.Uint32s("entitiesToRemove", removeMsg.EntitiesToRemove),
		)
	}
}
