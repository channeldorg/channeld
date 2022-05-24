package channeld

import (
	"log"
	"sync"
	"testing"
	"time"

	"channeld.clewcat.com/channeld/pkg/channeldpb"
)

func TestConcurrentAccessChannels(t *testing.T) {
	InitLogsAndMetrics()
	InitChannels()
	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		for i := 0; i < 100; i++ {
			CreateChannel(channeldpb.ChannelType_SUBWORLD, nil)
			time.Sleep(1 * time.Millisecond)
		}
		wg.Done()
	}()

	// Read-Write ratio = 100:1
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			counter := 0
			for i := 0; i < 100; i++ {
				if GetChannel(ChannelId(i)) != nil {
					counter++
				}
				time.Sleep(1 * time.Millisecond)
			}
			log.Println(counter)
			wg.Done()
		}()
	}

	wg.Add(1)
	go func() {
		for i := 0; i < 100; i++ {
			allChannels.Range(func(k interface{}, v interface{}) bool {
				return true
			})
			time.Sleep(1 * time.Millisecond)
		}
		wg.Done()
	}()

	wg.Wait()
}
