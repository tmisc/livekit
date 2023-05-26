package pacer

import (
	"sync"

	"github.com/gammazero/deque"
	"github.com/livekit/livekit-server/pkg/sfu/sendsidebwe"
	"github.com/livekit/protocol/logger"
)

type NoQueue struct {
	*Base

	logger logger.Logger

	lock      sync.RWMutex
	packets   deque.Deque[Packet]
	wake      chan struct{}
	isStopped bool
}

func NewNoQueue(logger logger.Logger, sendSideBWE *sendsidebwe.SendSideBWE) *NoQueue {
	n := &NoQueue{
		Base:   NewBase(logger, sendSideBWE),
		logger: logger,
		wake:   make(chan struct{}, 1),
	}
	n.packets.SetMinCapacity(9)

	go n.sendWorker()
	return n
}

func (n *NoQueue) Stop() {
	n.lock.Lock()
	if n.isStopped {
		n.lock.Unlock()
		return
	}

	close(n.wake)
	n.isStopped = true
	n.lock.Unlock()
}

func (n *NoQueue) Enqueue(p Packet) {
	n.lock.Lock()
	n.packets.PushBack(p)

	notify := false
	if n.packets.Len() == 1 && !n.isStopped {
		notify = true
	}
	n.lock.Unlock()

	if !notify {
		return
	}

	select {
	case n.wake <- struct{}{}:
	default:
	}
}

func (n *NoQueue) sendWorker() {
	for {
		<-n.wake
		for {
			n.lock.Lock()
			if n.isStopped {
				return
			}

			if n.packets.Len() == 0 {
				n.lock.Unlock()
				break
			}
			p := n.packets.PopFront()
			n.lock.Unlock()

			n.Base.SendPacket(&p)
		}
	}
}

// ------------------------------------------------
