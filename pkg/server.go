package barertc

import (
	"io"
	"net/http"
	"sync"
	"time"

	"git.kirsle.net/apps/barertc/pkg/config"
	"git.kirsle.net/apps/barertc/pkg/log"
	"git.kirsle.net/apps/barertc/pkg/models"
)

// Server is the primary back-end server struct for BareRTC, see main.go
type Server struct {
	// Timestamp when the server started.
	upSince time.Time

	// HTTP router.
	mux *http.ServeMux

	// Max number of messages we'll buffer for a subscriber
	subscriberMessageBuffer int

	// Connected WebSocket subscribers.
	subscribersMu sync.RWMutex
	subscribers   map[*Subscriber]struct{}

	// Cached filehandles for channel logging.
	logfh map[string]io.WriteCloser
}

// NewServer initializes the Server.
func NewServer() *Server {
	return &Server{
		subscriberMessageBuffer: 32,
		subscribers:             make(map[*Subscriber]struct{}),
	}
}

// Setup the server: configure HTTP routes, etc.
func (s *Server) Setup() error {
	// Enable the SQLite database for DM history?
	if config.Current.DirectMessageHistory.Enabled {
		if err := models.Initialize(config.Current.DirectMessageHistory.SQLiteDatabase); err != nil {
			log.Error("Error initializing SQLite database: %s", err)
		}
	}

	var mux = http.NewServeMux()

	mux.Handle("/", IndexPage())
	mux.Handle("/about", AboutPage())
	mux.Handle("/logout", LogoutPage())
	mux.Handle("/ws", s.WebSocket())
	mux.Handle("/poll", s.PollingAPI())
	mux.Handle("/api/statistics", s.Statistics())
	mux.Handle("/api/blocklist", s.BlockList())
	mux.Handle("/api/block/now", s.BlockNow())
	mux.Handle("/api/disconnect/now", s.DisconnectNow())
	mux.Handle("/api/authenticate", s.Authenticate())
	mux.Handle("/api/amend-jwt", s.AmendJWT())
	mux.Handle("/api/shutdown", s.ShutdownAPI())
	mux.Handle("/api/profile", s.UserProfile())
	mux.Handle("/api/message/history", s.MessageHistory())
	mux.Handle("/api/message/usernames", s.MessageUsernameHistory())
	mux.Handle("/api/message/clear", s.ClearMessages())
	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("dist/assets"))))
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("dist/static"))))

	s.mux = mux

	return nil
}

// ListenAndServe starts the web server.
func (s *Server) ListenAndServe(address string) error {
	// Run the polling user idle kicker.
	s.upSince = time.Now()
	go s.KickIdlePollUsers()
	go s.sendWhoListAfterReady()
	return http.ListenAndServe(address, s.mux)
}

// Send first WhoList update 15 seconds after the server reboots. This is in case a lot of chatters
// are online during a reboot: we avoid broadcasting Presence for 30 seconds and WhoList for 15 to
// reduce chatter and avoid kicking clients offline repeatedly for filling up their message buffer
// as everyone rejoins the chat all at once.
func (s *Server) sendWhoListAfterReady() {
	time.Sleep(16 * time.Second)
	log.Info("Up 15 seconds, sending WhoList to any online chatters")
	s.SendWhoList()
}
