package channeld

import (
	"testing"

	"github.com/channeldorg/channeld/pkg/channeldpb"
	"github.com/channeldorg/channeld/pkg/common"
	"github.com/stretchr/testify/assert"
)

func TestEntityChannelGroupController(t *testing.T) {
	InitChannels()

	/* Case 1: When Character A moves across the spatial channel, its PlayerController and PlayerState should be handed over together.
	 */
	charA := EntityId(1)
	pcA := EntityId(2)
	psA := EntityId(3)
	chA := createChannelWithId(common.ChannelId(charA), channeldpb.ChannelType_ENTITY, nil)
	chA.entityController.AddToGroup(channeldpb.EntityGroupType_HANDOVER, []EntityId{charA, pcA, psA})
	handoverEntities := chA.entityController.GetHandoverEntities()
	assert.Equal(t, 3, len(handoverEntities))
	assert.Contains(t, handoverEntities, charA)
	assert.Contains(t, handoverEntities, pcA)
	assert.Contains(t, handoverEntities, psA)

	/* Case 2: When A is attacked by Character B in another spatial server, A and its PlayerController and PlayerState
	 * should be handed over to B's spatial server and then locked from handover.
	 */
	charB := EntityId(4)
	pcB := EntityId(5)
	psB := EntityId(6)
	chB := createChannelWithId(common.ChannelId(charB), channeldpb.ChannelType_ENTITY, nil)
	chB.entityController.AddToGroup(channeldpb.EntityGroupType_HANDOVER, []EntityId{charB, pcB, psB})

	// Triggers cross-server attack
	chB.entityController.AddToGroup(channeldpb.EntityGroupType_LOCK, []EntityId{charA, charB})
	handoverEntities = chA.entityController.GetHandoverEntities()
	assert.Equal(t, 0, len(handoverEntities), "Character A is attacked and locked by Character B, should not handover")

	/* Case 3: When A leaves the combat, A and its PlayerController and PlayerState should be unlocked from handover, and then
	 * be handed over to the spatial channel corresponding to its current location.
	 */
	chA.entityController.RemoveFromGroup(channeldpb.EntityGroupType_LOCK, []EntityId{charA})
	handoverEntities = chA.entityController.GetHandoverEntities()
	assert.Equal(t, 3, len(handoverEntities), "Character A left the combat, should be able to handover")

	handoverEntities = chB.entityController.GetHandoverEntities()
	assert.Equal(t, 0, len(handoverEntities), "Character B is still in combat, should be locked from handover")

	/* Case 4: When a vehicle moves across the spatial channel, its passengers should be handed over together.
	 * If A gets down from the vehicle, A should be handed over to the spatial channel corresponding to its current location.
	 */
	vehicle := EntityId(7)
	chV := createChannelWithId(common.ChannelId(vehicle), channeldpb.ChannelType_ENTITY, nil)

	charC := EntityId(8)
	pcC := EntityId(9)
	psC := EntityId(10)
	chC := createChannelWithId(common.ChannelId(charC), channeldpb.ChannelType_ENTITY, nil)
	chC.entityController.AddToGroup(channeldpb.EntityGroupType_HANDOVER, []EntityId{charC, pcC, psC})

	// Character C gets into the vehicle
	chV.entityController.AddToGroup(channeldpb.EntityGroupType_HANDOVER, []EntityId{vehicle, charC})
	chC.entityController.AddToGroup(channeldpb.EntityGroupType_LOCK, []EntityId{charC})
	// Character A gets into the vehicle
	chV.entityController.AddToGroup(channeldpb.EntityGroupType_HANDOVER, []EntityId{vehicle, charA})
	chA.entityController.AddToGroup(channeldpb.EntityGroupType_LOCK, []EntityId{charA})
	// Character C moves aross the spatial channel first, but the vehicle hasn't yet.
	handoverEntities = chC.entityController.GetHandoverEntities()
	assert.Equal(t, 0, len(handoverEntities), "Character C should be locked from handover")

	// The vehicle moves across the spatial channel
	handoverEntities = chV.entityController.GetHandoverEntities()
	assert.Contains(t, handoverEntities, vehicle)
	assert.Contains(t, handoverEntities, charA, "Character A should be handed over with the vehicle")
	assert.Contains(t, handoverEntities, charC, "Character C should be handed over with the vehicle")

	// Character A gets down from the vehicle
	chV.entityController.RemoveFromGroup(channeldpb.EntityGroupType_HANDOVER, []EntityId{charA})
	chA.entityController.RemoveFromGroup(channeldpb.EntityGroupType_LOCK, []EntityId{charA})
	// IMPORTANT: after getting off the vehicle, Character A add its PlayerController and PlayerState back to the handover group again
	chA.entityController.AddToGroup(channeldpb.EntityGroupType_HANDOVER, []EntityId{charA, pcA, psA})
	handoverEntities = chA.entityController.GetHandoverEntities()
	assert.Equal(t, 3, len(handoverEntities), "Character A got off the vehicle, should be able to handover again")

	/* Case 5: When A is in a vehicle and attacked by Character B in another spatial server, A should be pulled off the vehicle and handed over to B's spatial server;
	 * the vehicle and all its passengers can continue to move across the spatial channel be handed over together.
	 */

	// Character A gets into the vehicle again
	chV.entityController.AddToGroup(channeldpb.EntityGroupType_HANDOVER, []EntityId{vehicle, charA})
	// Triggers cross-server attack
	chB.entityController.AddToGroup(channeldpb.EntityGroupType_LOCK, []EntityId{charA, charB})
	// The spatial server of the vehicle should remove Character A from the vehicle's handover group
	chV.entityController.RemoveFromGroup(channeldpb.EntityGroupType_HANDOVER, []EntityId{charA})
	handoverEntities = chA.entityController.GetHandoverEntities()
	assert.Equal(t, 0, len(handoverEntities), "Character A is attacked and locked by Character B, should not handover")

	// The vehicle moves across the spatial channel
	handoverEntities = chV.entityController.GetHandoverEntities()
	assert.Contains(t, handoverEntities, vehicle)
	assert.NotContains(t, handoverEntities, charA, "Character A should NOT be handed over with the vehicle")
	assert.Contains(t, handoverEntities, charC, "Character C should be handed over with the vehicle")
}
