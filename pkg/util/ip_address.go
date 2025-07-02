package util

import (
	"net/http"
	"strings"

	"git.kirsle.net/apps/barertc/pkg/config"
)

/*
IPAddress returns the best guess at the user's IP address, as a string for logging.
*/
func IPAddress(r *http.Request) string {
	if config.Current.UseXForwardedFor {
		if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
			return realIP
		}
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			return strings.SplitN(xff, " ", 1)[0]
		}
	}
	return r.RemoteAddr
}
