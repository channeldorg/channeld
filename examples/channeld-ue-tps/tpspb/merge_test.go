package tpspb

import (
	"testing"

	"channeld.clewcat.com/channeld/pkg/channeld"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/encoding/protojson"
)

// How to run: go test -bench Merge -benchmem -benchtime=10s -timeout=30s -count=1

var srcDataJson = []byte(`{"gameState":{"spectatorClassName":"/Script/Engine.SpectatorPawn","gameModeClassName":"/Game/Blueprints/BP_TestRepGameMode.BP_TestRepGameMode_C","bReplicatedHasBegunPlay":true},"actorStates":{"1":{"localRole":3,"remoteRole":1,"bHidden":true},"1048598":{"owningConnId":4,"bReplicateMovement":true,"localRole":3,"remoteRole":2,"owner":{"netGUID":1048590,"bunchBitsNum":0},"instigator":{"netGUID":1048598,"bunchBitsNum":0},"replicatedMovement":{"location":{"y":500,"z":118.274994}}},"1048580":{"owningConnId":3,"localRole":3,"remoteRole":1,"owner":{"netGUID":1048578,"bunchBitsNum":0},"bHidden":true},"1048586":{"owningConnId":3,"bReplicateMovement":true,"localRole":3,"remoteRole":2,"owner":{"netGUID":1048578,"bunchBitsNum":0},"instigator":{"netGUID":1048586,"bunchBitsNum":0},"replicatedMovement":{"linearVelocity":{"x":70.356514,"y":43.1561699,"z":-592.405457},"location":{"x":1180.83948,"y":767.166809,"z":118.149994},"rotation":{"y":32.5804596}}},"1048592":{"owningConnId":4,"localRole":3,"remoteRole":1,"owner":{"netGUID":1048590,"bunchBitsNum":0},"bHidden":true}},"pawnStates":{"1048598":{"playerState":{"netGUID":1048592,"bunchBitsNum":0},"controller":{"netGUID":1048590,"bunchBitsNum":0}},"1048586":{"playerState":{"netGUID":1048580,"bunchBitsNum":0},"controller":{"netGUID":1048578,"bunchBitsNum":0}}},"characterStates":{"1048586":{"basedMovement":{"movementBase":{"owner":{"netGUID":1048595,"context":[{"netGUID":1048595,"pathName":"Floor","outerGUID":3}],"netGUIDBunch":"JwGA","bunchBitsNum":0},"compName":"StaticMeshComponent0"},"bServerHasBaseComponent":true},"movementMode":1,"animRootMotionTranslationScale":1},"1048598":{"basedMovement":{"movementBase":{"owner":{"netGUID":1048595,"context":[{"netGUID":1048595,"pathName":"Floor","outerGUID":3}],"netGUIDBunch":"JwGA","bunchBitsNum":24},"compName":"StaticMeshComponent0"},"bServerHasBaseComponent":true},"movementMode":1,"animRootMotionTranslationScale":1}},"playerStates":{"1048592":{"playerId":257,"playerName":"IndieST-M16-AC36D147"},"1048580":{"playerId":256,"playerName":"IndieST-M16-C1CE32E0"}},"testGameState":{}}`)

var dstDataJson1 = []byte(`{"gameState":{"spectatorClassName":"/Script/Engine.SpectatorPawn","gameModeClassName":"/Game/Blueprints/BP_TestRepGameMode.BP_TestRepGameMode_C","bReplicatedHasBegunPlay":true},"actorStates":{"1048580":{"owningConnId":3,"localRole":3,"remoteRole":1,"owner":{"netGUID":1048578,"bunchBitsNum":0},"bHidden":true},"1048586":{"owningConnId":3,"bReplicateMovement":true,"localRole":3,"remoteRole":2,"owner":{"netGUID":1048578,"bunchBitsNum":0},"instigator":{"netGUID":1048586,"bunchBitsNum":0},"replicatedMovement":{"location":{"y":500,"z":118.274994}}},"1":{"localRole":3,"remoteRole":1,"bHidden":true}},"pawnStates":{"1048586":{"playerState":{"netGUID":1048580,"bunchBitsNum":0},"controller":{"netGUID":1048578,"bunchBitsNum":0}}},"characterStates":{"1048586":{"basedMovement":{"movementBase":{"owner":{"netGUID":1048595,"context":[{"netGUID":1048595,"pathName":"Floor","outerGUID":3}],"netGUIDBunch":"JwGA","bunchBitsNum":24},"compName":"StaticMeshComponent0"},"bServerHasBaseComponent":true},"movementMode":1,"animRootMotionTranslationScale":1}},"playerStates":{"1048580":{"playerId":256,"playerName":"IndieST-M16-C1CE32E0"}},"testGameState":{}}`)

var dstDataJson2 = []byte(`{"actorStates":{"1048598":{"replicatedMovement":{"linearVelocity":{"x":599.873718,"y":-11.7270174,"z":564.423462},"location":{"x":1484.74963,"y":897.948486,"z":139.410782},"rotation":{"y":-3.56381297}}}},"characterStates":{"1048598":{"basedMovement":{"movementBase":{"owner":{"netGUID":0}},"bServerHasBaseComponent":false},"movementMode":3}}}`)

var srcData = &TestRepChannelData{}
var dstData1 = &TestRepChannelData{}
var dstData2 = &TestRepChannelData{}

func init() {
	channeld.InitLogs()

	srcData.Init()

	protojson.Unmarshal(srcDataJson, srcData)
	protojson.Unmarshal(dstDataJson1, dstData1)
	protojson.Unmarshal(dstDataJson2, dstData2)
}

func TestJsonUnmarshal(t *testing.T) {
	assert.NotNil(t, srcData.ActorStates)
	assert.NotNil(t, srcData.CharacterStates)
	assert.NotNil(t, srcData.ActorStates[1048580])

	assert.NotNil(t, dstData1.ActorStates)

	assert.NotNil(t, dstData2.ActorStates)
}

// create a benchmark for the merge function
func BenchmarkMerge1(b *testing.B) {
	b.ResetTimer()
	// run the merge function b.N times
	for i := 0; i < b.N; i++ {
		srcData.Merge(dstData1, nil, nil)
	}

	// BenchmarkMerge1-20      13945999               823.2 ns/op           313 B/op         36 allocs/op
}

func BenchmarkMerge2(b *testing.B) {
	b.ResetTimer()
	// run the merge function b.N times
	for i := 0; i < b.N; i++ {
		srcData.Merge(dstData2, nil, nil)
	}

	// BenchmarkMerge2-20      48221277               253.5 ns/op            40 B/op         10 allocs/op
}

func BenchmarkMarshal1(b *testing.B) {
	b.ResetTimer()
	// run the merge function b.N times
	for i := 0; i < b.N; i++ {
		protojson.Marshal(dstData1)
	}

	// BenchmarkMarshal1-20              626674             19006 ns/op           11330 B/op        210 allocs/op
}

func BenchmarkMarshal2(b *testing.B) {
	b.ResetTimer()
	// run the merge function b.N times
	for i := 0; i < b.N; i++ {
		protojson.Marshal(dstData2)
	}

	// BenchmarkMarshal2-20             1561024              7698 ns/op            4054 B/op         90 allocs/op
}
