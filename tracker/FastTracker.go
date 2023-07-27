package tracker

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/AlchemillaHQ/owtt/utils"
	"github.com/gofiber/contrib/websocket"
)

type FastTracker struct {
	Swarms           map[string]*Swarm
	Peers            map[string]*PeerContext
	MaxOffers        int
	AnnounceInterval int
	FastTrackerMutex sync.Mutex
}

type PeerContext struct {
	WS               *websocket.Conn
	WSWriteMutex     sync.Mutex
	Id               string
	InfoHashes       map[string]struct{}
	PeerContextMutex sync.Mutex
}

type Offer struct {
	Type string      `json:"type"`
	Sdp  interface{} `json:"sdp"`
}

type OfferItem struct {
	Offer   Offer       `json:"offer"`
	OfferID interface{} `json:"offer_id"`
}

type Announce struct {
	Numwant    int         `json:"numwant"`
	Event      *string     `json:"event"`
	Uploaded   int         `json:"uploaded"`
	Downloaded int         `json:"downloaded"`
	Left       interface{} `json:"left"`
	Action     string      `json:"action"`
	InfoHash   string      `json:"info_hash"`
	PeerID     string      `json:"peer_id"`
	Answer     interface{} `json:"answer"`
	Offers     []OfferItem `json:"offers"`
	ToPeerId   string      `json:"to_peer_id"`
}

func NewFastTracker(maxOffers, announceInterval int) *FastTracker {
	return &FastTracker{
		Swarms:           make(map[string]*Swarm),
		Peers:            make(map[string]*PeerContext),
		MaxOffers:        maxOffers,
		AnnounceInterval: announceInterval,
		FastTrackerMutex: sync.Mutex{},
	}
}

func (pc *PeerContext) writeWebSocketMessage(messageType int, data []byte) error {
	pc.WSWriteMutex.Lock()
	defer pc.WSWriteMutex.Unlock()

	if pc.WS == nil {
		return fmt.Errorf("WebSocket connection is nil")
	}

	return pc.WS.WriteMessage(messageType, data)
}

func (pc *PeerContext) SendMessage(data map[string]interface{}) {
	if pc.WS != nil {
		jsonData, err := json.Marshal(data)
		if err != nil {
			return
		}

		err = pc.writeWebSocketMessage(websocket.TextMessage, jsonData)

		if err != nil {
			return
		}
	}
}

func (ft *FastTracker) GetSwarms() map[string]*Swarm {
	return ft.Swarms
}

func (ft *FastTracker) GetSwarm(infoHash string) *Swarm {
	ft.FastTrackerMutex.Lock()
	defer ft.FastTrackerMutex.Unlock()

	if swarm, exists := ft.Swarms[infoHash]; exists {
		return swarm
	}

	return nil
}

func (ft *FastTracker) ProcessMessage(data string, peer *PeerContext) {
	var parsedData Announce

	err := json.Unmarshal([]byte(data), &parsedData)

	if err != nil {
		utils.LogError("announce: invalid JSON data")
		return
	}

	action := parsedData.Action
	event := parsedData.Event

	if action == "announce" {
		if event == nil {
			answer := parsedData.Answer
			if answer == nil {
				ft.ProcessAnnounce(parsedData, peer, false)
			} else {
				ft.ProcessAnswer(data, peer)
			}
		} else {
			if *event == "started" {
				ft.ProcessAnnounce(parsedData, peer, false)
			} else if *event == "stopped" {
				ft.ProcessStop(data, peer)
			} else if *event == "completed" {
				ft.ProcessAnnounce(parsedData, peer, true)
			} else {
				utils.LogError("announce: unknown event")
			}
		}
	} else if action == "scrape" {
		ft.ProcessScrape(data, peer)
	} else {
		utils.LogError("announce: unknown action")
	}

}

func (ft *FastTracker) SendOffersToPeers(data Announce, peers []*PeerContext, peer *PeerContext, infoHash string) {
	if len(peers) <= 1 {
		return
	}

	numwant := data.Numwant

	countPeersToSend := len(peers) - 1
	countOffersToSend := utils.Min(countPeersToSend, utils.Min(len(data.Offers), utils.Min(ft.MaxOffers, numwant)))

	if countOffersToSend == countPeersToSend {
		for i, toPeer := range peers {
			if toPeer != peer && i < len(data.Offers) {
				offerItem := data.Offers[i]
				ft.SendOffer(offerItem.Offer, peer.Id, toPeer, infoHash, offerItem.OfferID)
			}
		}
	} else {
		source := rand.NewSource(time.Now().UnixNano())
		random := rand.New(source)

		peerIndex := random.Intn(len(peers))

		for i := 0; i < countOffersToSend; i++ {
			toPeer := peers[peerIndex]

			if toPeer == peer {
				i--
			} else if i < len(data.Offers) {
				offerItem := data.Offers[i]
				ft.SendOffer(offerItem.Offer, peer.Id, toPeer, infoHash, offerItem.OfferID)
			}

			peerIndex++
			if peerIndex == len(peers) {
				peerIndex = 0
			}
		}
	}
}

func (ft *FastTracker) SendOffer(offer Offer, fromPeerID string, toPeer *PeerContext, infoHash string, offerID interface{}) {
	if offer.Sdp == nil || offerID == nil {
		panic("announce: wrong offer item format")
	}

	responseData := map[string]interface{}{
		"action":    "announce",
		"info_hash": infoHash,
		"offer_id":  offerID,
		"peer_id":   fromPeerID,
		"offer": map[string]interface{}{
			"type": "offer",
			"sdp":  offer.Sdp,
		},
	}

	toPeer.SendMessage(responseData)
}

func (ft *FastTracker) ProcessAnnounce(data Announce, peer *PeerContext, completed bool) {
	peerId := data.PeerID
	infoHash := data.InfoHash

	var swarm *Swarm

	if peer.Id == "" {
		peer.PeerContextMutex.Lock()
		defer peer.PeerContextMutex.Unlock()

		ft.FastTrackerMutex.Lock()
		peer.Id = peerId
		if existingPeer, exists := ft.Peers[peerId]; exists {
			ft.DisconnectPeer(existingPeer)
		}
		ft.FastTrackerMutex.Unlock()

		ft.Peers[peerId] = peer
	} else if peer.Id != peerId {
		ft.FastTrackerMutex.Lock()
		if existingPeer, exists := ft.Peers[peerId]; exists {
			ft.DisconnectPeer(existingPeer)
		}
		ft.FastTrackerMutex.Unlock()
	} else {
		peer.PeerContextMutex.Lock()
		defer peer.PeerContextMutex.Unlock()

		if _, exists := peer.InfoHashes[infoHash]; exists {
			if existingSwarm, exists := ft.Swarms[infoHash]; exists {
				swarm = existingSwarm
			}
		}
	}

	left := data.Left

	if left == nil || left == 0 {
		completed = true
	}

	isPeerCompleted := completed

	if swarm == nil {
		swarm = ft.AddPeerToSwarm(peer, infoHash, isPeerCompleted)
	} else if swarm != nil {
		if isPeerCompleted {
			swarm.SetCompleted(peer)
		}
	} else {
		panic("announce: illegal info_hash field")
	}

	responseData := map[string]interface{}{
		"action":     "announce",
		"interval":   ft.AnnounceInterval,
		"info_hash":  infoHash,
		"complete":   swarm.GetCompletedCount(),
		"incomplete": swarm.GetPeersCount() - swarm.GetCompletedCount(),
	}

	peer.SendMessage(responseData)
	ft.SendOffersToPeers(data, swarm.GetPeers(), peer, infoHash)
}

func (ft *FastTracker) ProcessAnswer(data string, peer *PeerContext) {
	parsedData, err := utils.ParseRawJSON(data)

	if err != nil {
		return
	}

	if _, ok := parsedData["to_peer_id"].(string); !ok {
		fmt.Println("answer: to_peer_id is not in the swarm")
	}

	toPeerId := parsedData["to_peer_id"].(string)

	toPeer, ok := ft.Peers[toPeerId]

	if !ok {
		fmt.Println("answer: to_peer_id is not in the swarm")
		return
	}

	parsedData["peer_id"] = peer.Id
	delete(parsedData, "to_peer_id")

	toPeer.SendMessage(parsedData)
}

func (ft *FastTracker) ProcessStop(data string, peer *PeerContext) {
	parsedData, err := utils.ParseRawJSON(data)

	if err != nil {
		fmt.Println("stop: invalid JSON data")
		return
	}

	infoHashRaw, ok := parsedData["info_hash"]
	if !ok {
		fmt.Println("stop: info_hash is not in the swarm")
	}

	if _, ok := parsedData["info_hash"].(string); !ok {
		fmt.Println("stop: info_hash is not in the swarm")
	}

	infoHash, ok := infoHashRaw.(string)

	if !ok {
		fmt.Println("stop: info_hash is not in the swarm")
		return
	}

	ft.FastTrackerMutex.Lock()
	swarm, ok := ft.Swarms[infoHash]
	ft.FastTrackerMutex.Unlock()

	if !ok {
		fmt.Println("stop: info_hash is not in the swarm")
		return
	}

	ft.FastTrackerMutex.Lock()
	swarm.RemovePeer(peer)
	ft.FastTrackerMutex.Unlock()

	peer.PeerContextMutex.Lock()
	delete(peer.InfoHashes, infoHash)
	peer.PeerContextMutex.Unlock()

	if swarm.GetPeersCount() == 0 {
		ft.FastTrackerMutex.Lock()
		delete(ft.Swarms, infoHash)
		ft.FastTrackerMutex.Unlock()
	}
}

func (ft *FastTracker) ProcessScrape(data string, peer *PeerContext) {
	files := make(map[string]map[string]int)
	parsedData, err := utils.ParseRawJSON(data)

	if err != nil {
		fmt.Println("scrape: invalid JSON data")
		return
	}

	if _, exists := parsedData["info_hash"]; !exists {
		for _, swarm := range ft.Swarms {
			infoHash := swarm.GetInfoHash()
			files[infoHash] = map[string]int{
				"complete":   swarm.GetCompletedCount(),
				"incomplete": swarm.GetPeersCount() - swarm.GetCompletedCount(),
				"downloaded": swarm.GetCompletedCount(),
			}
		}
	} else if infoHashes, isArray := parsedData["info_hash"].([]interface{}); isArray {
		for _, infoHashRaw := range infoHashes {
			if singleInfoHash, isString := infoHashRaw.(string); isString {
				if swarm, exists := ft.Swarms[singleInfoHash]; exists {
					files[singleInfoHash] = map[string]int{
						"complete":   swarm.GetCompletedCount(),
						"incomplete": swarm.GetPeersCount() - swarm.GetCompletedCount(),
						"downloaded": swarm.GetCompletedCount(),
					}
				} else {
					files[singleInfoHash] = map[string]int{
						"complete":   0,
						"incomplete": 0,
						"downloaded": 0,
					}
				}
			}
		}
	} else if infoHash, isString := parsedData["info_hash"].(string); isString {
		if swarm, exists := ft.Swarms[infoHash]; exists {
			files[infoHash] = map[string]int{
				"complete":   swarm.GetCompletedCount(),
				"incomplete": swarm.GetPeersCount() - swarm.GetCompletedCount(),
				"downloaded": swarm.GetCompletedCount(),
			}
		} else {
			files[infoHash] = map[string]int{
				"complete":   0,
				"incomplete": 0,
				"downloaded": 0,
			}
		}
	}

	responseData := map[string]interface{}{
		"action": "scrape",
		"files":  files,
	}

	peer.SendMessage(responseData)
}

func (ft *FastTracker) DisconnectPeer(peer *PeerContext) {
	if peer.Id == "" {
		return
	}

	for infoHash := range peer.InfoHashes {
		ft.FastTrackerMutex.Lock()
		if swarm, exists := ft.Swarms[infoHash]; exists {
			swarm.RemovePeer(peer)
			if swarm.GetPeersCount() == 0 {
				delete(ft.Swarms, infoHash)
			}
		}
		ft.FastTrackerMutex.Unlock()
	}

	peer.InfoHashes = make(map[string]struct{})
	delete(ft.Peers, peer.Id)
}

func (ft *FastTracker) AddPeerToSwarm(peer *PeerContext, infoHash string, completed bool) *Swarm {
	ft.FastTrackerMutex.Lock()
	defer ft.FastTrackerMutex.Unlock()

	if _, exists := ft.Swarms[infoHash]; !exists {
		swarm := NewSwarm(infoHash)
		ft.Swarms[infoHash] = swarm
	}

	ft.Swarms[infoHash].AddPeer(peer, completed)

	peer.InfoHashes[infoHash] = struct{}{}
	return ft.Swarms[infoHash]
}
