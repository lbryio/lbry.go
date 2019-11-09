package stop

import (
	"testing"
	"time"
)

func TestStopParentChild(t *testing.T) {
	stopper := New()
	parent := stopper.Child()
	child := New(parent)
	ticker := time.NewTicker(1 * time.Second)
	go func() {
		time.Sleep(3 * time.Second)
		stopper.Stop()
	}()
loop:
	for {
		select {
		case <-ticker.C:
			continue
		case <-child.Ch():
			break loop
		}
	}
	//Will run forever if stop is not propagated
}

var tracker int
var trackerExpected int

func TestWaitParentChild(t *testing.T) {
	parent := New()
	child1 := parent.Child()
	child2 := parent.Child()
	child1.Add(1)
	go runChild("child1", child1, 1*time.Second)
	child2.Add(1)
	go runChild("child2", child2, 2*time.Second)
	time.Sleep(5 * time.Second)
	parent.Stop()
	parent.Wait()
	if tracker != trackerExpected {
		t.Errorf("Stopper is not waiting for children to finish, expected %d, got %d", trackerExpected, tracker)
	}
}

func runChild(name string, child *Group, duration time.Duration) {
	defer child.Done()
	ticker := time.NewTicker(duration)
loop:
	for {
		select {
		case <-ticker.C:
			trackerExpected++
			println(name + " start tick++")
			time.Sleep(2 * duration)
			tracker++
			println(name + " end tick++")
			continue loop
		case <-child.Ch():
			break loop
		}
	}
}
