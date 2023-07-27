package tracker

import (
	"sync"
	"sync/atomic"
)

type SwarmMutex struct {
	mu     sync.Mutex
	locked int32
}

type Swarm struct {
	infoHash       string
	peers          []*PeerContext
	completedPeers map[string]struct{}
	mutex          SwarmMutex
}

func (sm *SwarmMutex) Lock() {
	sm.mu.Lock()
	atomic.StoreInt32(&sm.locked, 1)
}

func (sm *SwarmMutex) Unlock() {
	atomic.StoreInt32(&sm.locked, 0)
	sm.mu.Unlock()
}

func (sm *SwarmMutex) IsLocked() bool {
	return atomic.LoadInt32(&sm.locked) == 1
}

func NewSwarm(infoHash string) *Swarm {
	return &Swarm{
		infoHash:       infoHash,
		peers:          make([]*PeerContext, 0),
		completedPeers: make(map[string]struct{}),
		mutex:          SwarmMutex{},
	}
}

func (s *Swarm) GetInfoHash() string {
	return s.infoHash
}

func (s *Swarm) GetPeersCount() int {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return len(s.peers)
}

func (s *Swarm) GetCompletedCount() int {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return len(s.completedPeers)
}

func (s *Swarm) GetPeers() []*PeerContext {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	peersCopy := make([]*PeerContext, len(s.peers))
	copy(peersCopy, s.peers)

	return peersCopy
}

func (s *Swarm) AddPeer(peer *PeerContext, completed bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.peers = append(s.peers, peer)
	if completed {
		s.completedPeers[peer.Id] = struct{}{}
	}
}

func (s *Swarm) RemovePeer(peer *PeerContext) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	delete(s.completedPeers, peer.Id)
	for i, p := range s.peers {
		if p == peer {
			s.peers = append(s.peers[:i], s.peers[i+1:]...)
			break
		}
	}
}

func (s *Swarm) SetCompleted(peer *PeerContext) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.completedPeers[peer.Id] = struct{}{}
}
