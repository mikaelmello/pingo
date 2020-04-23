package core

import "sync"

// ReplyMap is the interface for a thread-safe map that will store channels
// used by echo requests to receive their replies
type ReplyMap interface {
	Get(key uint16) chan *RoundTrip
	Erase(key uint16)
}

type replyMap struct {
	allData map[uint16](chan *RoundTrip)
	rwm     sync.RWMutex
}

func newReplyMap() ReplyMap {
	return &replyMap{
		allData: make(map[uint16](chan *RoundTrip)),
		rwm:     sync.RWMutex{},
	}
}

func (m *replyMap) Get(key uint16) chan *RoundTrip {
	m.rwm.RLock()
	defer m.rwm.RUnlock()
	ch, ok := m.allData[key]
	if !ok {
		m.allData[key] = make(chan *RoundTrip, 1)
	}
	return ch
}

func (m *replyMap) Erase(key uint16) {
	m.rwm.Lock()
	defer m.rwm.Unlock()
	_, ok := m.allData[key]
	if ok {
		delete(m.allData, key)
	}
}
