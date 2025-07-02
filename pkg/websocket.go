package barertc

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"git.kirsle.net/apps/barertc/pkg/config"
	"git.kirsle.net/apps/barertc/pkg/log"
	"git.kirsle.net/apps/barertc/pkg/util"
	"nhooyr.io/websocket"
)

// WebSocket handles the /ws websocket connection endpoint.
func (s *Server) WebSocket() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := util.IPAddress(r)
		log.Info("WebSocket connection from %s - %s", ip, r.Header.Get("User-Agent"))
		log.Debug("Headers: %+v", r.Header)
		c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			CompressionMode: websocket.CompressionDisabled,
		})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Impossible d'accepter la connexion websocket : %s", err)
			return
		}
		defer c.Close(websocket.StatusInternalError, "erreur interne du serveur")

		log.Debug("WebSocket: %s has connected", ip)
		c.SetReadLimit(config.Current.WebSocketReadLimit)

		// CloseRead starts a goroutine that will read from the connection
		// until it is closed.
		// ctx := c.CloseRead(r.Context())
		ctx, cancel := context.WithCancel(r.Context())

		sub := s.NewWebSocketSubscriber(ctx, c, cancel)

		s.AddSubscriber(sub)
		defer s.DeleteSubscriber(sub)

		go sub.ReadLoop(s)
		pinger := time.NewTicker(PingInterval)
		for {
			select {
			case msg := <-sub.messages:
				err = writeTimeout(ctx, time.Second*time.Duration(config.Current.WebSocketSendTimeout), c, msg)
				if err != nil {
					return
				}
			case <-pinger.C:
				sub.SendPing()
			case <-ctx.Done():
				pinger.Stop()
				return
			}
		}

	})
}

func writeTimeout(ctx context.Context, timeout time.Duration, c *websocket.Conn, msg []byte) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return c.Write(ctx, websocket.MessageText, msg)
}
