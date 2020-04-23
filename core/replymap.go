package core

import "sync"

// ReplyMap is the interface for a thread-safe map that will store channels
// used by echo requests to receive their replies
type ReplyMap interface {
	GetOrCreate(key uint16) chan *RoundTrip
	Get(key uint16) (chan *RoundTrip, bool)
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

func (m *replyMap) GetOrCreate(key uint16) chan *RoundTrip {
	m.rwm.RLock()
	defer m.rwm.RUnlock()
	ch, ok := m.allData[key]
	if !ok {
		ch = make(chan *RoundTrip, 1)
		m.allData[key] = ch
	}
	return ch
}

func (m *replyMap) Get(key uint16) (chan *RoundTrip, bool) {
	m.rwm.RLock()
	defer m.rwm.RUnlock()
	ch, ok := m.allData[key]
	return ch, ok
}

func (m *replyMap) Erase(key uint16) {
	m.rwm.Lock()
	defer m.rwm.Unlock()
	ch, ok := m.allData[key]
	if ok {
		close(ch)
		delete(m.allData, key)
	}
}
