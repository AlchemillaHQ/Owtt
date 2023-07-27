package tracker

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/AlchemillaHQ/owtt/utils"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cache"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/jpillora/ipfilter"
)

type WebSocketContext struct {
	Conn     *websocket.Conn
	UserData interface{}
}

type WebtorrentTracker struct {
	Port                   int
	Host                   string
	KeyFile                string
	CertFile               string
	Tracker                *FastTracker
	Filter                 *ipfilter.IPFilter
	WebSocketCount         int
	WebSocketMap           map[*websocket.Conn]*WebSocketContext
	WebTorrentTrackerMutex sync.Mutex
	ReverseProxy           bool
	RateLimit              int
	RateLimitExpiry        int
}

func NewWebtorrentTracker(port int, host, keyFile, certFile string, maxOffers int, announceInterval int, reverseProxy bool, rateLimit int, rateLimitExpiry int) *WebtorrentTracker {
	return &WebtorrentTracker{
		Port:            port,
		Host:            host,
		KeyFile:         keyFile,
		CertFile:        certFile,
		WebSocketMap:    make(map[*websocket.Conn]*WebSocketContext),
		Tracker:         NewFastTracker(maxOffers, announceInterval),
		ReverseProxy:    reverseProxy,
		RateLimit:       rateLimit,
		RateLimitExpiry: rateLimitExpiry,
	}
}

func (wt *WebtorrentTracker) Run() error {

	fiberConfig := fiber.Config{
		DisableStartupMessage: true,
		Network:               "tcp",
	}

	if wt.ReverseProxy {
		fiberConfig.ProxyHeader = fiber.HeaderXForwardedFor
	}

	fiberApp := fiber.New(fiberConfig)
	fiberApp.Use(limiter.New(limiter.Config{
		Expiration: time.Duration(wt.RateLimitExpiry) * time.Second,
		Max:        wt.RateLimit,
	}))

	fiberApp.Get("/stats.json", func(c *fiber.Ctx) error {
		return StatsHandler(c, wt)
	})

	fiberApp.Use(cache.New(cache.Config{
		Next: func(c *fiber.Ctx) bool {
			return strings.Contains(c.Route().Path, "/")
		},
	}))

	fiberApp.Get("/", websocket.New(wt.handleWebSocketMessages))

	if !utils.FileExists(wt.KeyFile) || !utils.FileExists(wt.CertFile) {
		utils.LogDebug("Starting tracker without TLS")
		fiberApp.Listen(fmt.Sprintf("%s:%d", wt.Host, wt.Port))
	} else {
		fiberApp.ListenTLS(fmt.Sprintf("%s:%d", wt.Host, wt.Port), wt.CertFile, wt.KeyFile)
	}

	return nil
}

func (wt *WebtorrentTracker) handleWebSocketMessages(c *websocket.Conn) {
	const maxPayloadLength = 64 * 1024

	for {
		_, message, err := c.ReadMessage()

		if err != nil {
			utils.LogDebug("Websocket closed for peer " + c.RemoteAddr().String())

			wt.WebTorrentTrackerMutex.Lock()
			if _, exists := wt.WebSocketMap[c]; exists {
				wt.WebSocketCount--
				delete(wt.WebSocketMap, c)
			}

			wt.WebTorrentTrackerMutex.Unlock()

			break
		}

		if len(message) > maxPayloadLength {
			utils.LogDebug("Message from peer " + c.RemoteAddr().String() + " exceeds maximum length")
			continue
		}

		if !json.Valid(message) {
			utils.LogDebug("Invalid JSON received from peer " + c.RemoteAddr().String())
			continue
		}

		wt.WebTorrentTrackerMutex.Lock()
		if _, exists := wt.WebSocketMap[c]; !exists {
			wt.WebSocketCount++

			peer := &PeerContext{
				WS:         c,
				InfoHashes: make(map[string]struct{}),
			}

			context := &WebSocketContext{
				Conn:     c,
				UserData: peer,
			}

			wt.WebSocketMap[c] = context
		}

		wt.WebTorrentTrackerMutex.Unlock()

		wt.WebTorrentTrackerMutex.Lock()
		context := wt.WebSocketMap[c]
		wt.WebTorrentTrackerMutex.Unlock()

		mStr := string(message)
		wt.Tracker.ProcessMessage(mStr, context.UserData.(*PeerContext))
	}
}
