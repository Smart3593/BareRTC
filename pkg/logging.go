package barertc

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"git.kirsle.net/apps/barertc/pkg/config"
	"git.kirsle.net/apps/barertc/pkg/log"
	"git.kirsle.net/apps/barertc/pkg/messages"
)

// IsLoggingUsername checks whether the app is currently configured to log a user's DMs.
func IsLoggingUsername(sub *Subscriber) bool {
	if !config.Current.Logging.Enabled || sub == nil {
		return false
	}

	// Has a cached setting and writer.
	if sub.log {
		return true
	}

	// Check the server config.
	for _, username := range config.Current.Logging.Usernames {
		if username == sub.Username {
			sub.log = true
		}
	}

	return sub.log
}

// IsLoggingChannel checks whether the app is currently logging a public channel.
func IsLoggingChannel(channel string) bool {
	if !config.Current.Logging.Enabled {
		return false
	}

	for _, value := range config.Current.Logging.Channels {
		if value == channel {
			return true
		}
	}
	return false
}

// LogMessage appends to a user's conversation log.
func LogMessage(sub *Subscriber, otherUsername, senderUsername string, msg messages.Message) {
	if sub == nil || !sub.log {
		return
	}

	// Create or get the filehandle.
	fh, err := initLogFile(sub, "@"+sub.Username, otherUsername)
	if err != nil {
		log.Error("LogMessage(%s): %s", sub.Username, err)
		return
	}

	fh.Write(
		[]byte(fmt.Sprintf(
			"%s [%s] %s\n",
			time.Now().Format(time.RFC3339),
			senderUsername,
			msg.Message,
		)),
	)
}

// LogChannel appends to a channel's conversation log.
func LogChannel(s *Server, channel string, username string, msg messages.Message) {
	fh, err := initLogFile(s, channel)
	if err != nil {
		log.Error("LogChannel(%s): %s", channel, err)
	}

	fh.Write(
		[]byte(fmt.Sprintf(
			"%s [%s] %s\n",
			time.Now().Format(time.RFC3339),
			username,
			msg.Message,
		)),
	)
}

// Tear down log files for subscribers.
func (s *Subscriber) teardownLogs() {
	if s.logfh == nil {
		return
	}

	for username, fh := range s.logfh {
		log.Error("TeardownLogs(%s/%s)", s.Username, username)
		fh.Close()
	}
}

// Initialize a logging directory.
func initLogFile(sub LogCacheable, components ...string) (io.WriteCloser, error) {
	// Initialize the logfh cache?
	var logfh = sub.GetLogFilehandleCache()

	var (
		suffix = components[len(components)-1]
		middle = components[:len(components)-1]
		paths  = append([]string{
			config.Current.Logging.Directory,
		}, middle...,
		)
		filename = strings.Join(
			append(paths, suffix+".txt"),
			"/",
		)
	)

	// Already have this conversation log open?
	if fh, ok := logfh[suffix]; ok {
		return fh, nil
	}

	log.Warn("Initialize log directory: path=%+v suffix=%s", paths, suffix)
	if err := os.MkdirAll(strings.Join(paths, "/"), 0755); err != nil {
		return nil, err
	}

	fh, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	logfh[suffix] = fh
	return logfh[suffix], nil
}

// Interface for objects that hold log filehandle caches.
type LogCacheable interface {
	GetLogFilehandleCache() map[string]io.WriteCloser
}

// Implementations of LogCacheable.
func (sub *Subscriber) GetLogFilehandleCache() map[string]io.WriteCloser {
	if sub.logfh == nil {
		sub.logfh = map[string]io.WriteCloser{}
	}
	return sub.logfh
}
func (s *Server) GetLogFilehandleCache() map[string]io.WriteCloser {
	if s.logfh == nil {
		s.logfh = map[string]io.WriteCloser{}
	}
	return s.logfh
}
