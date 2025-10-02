package requestlogging

/*
 Copyright (c) 2025 Rhys Bryant

 This program is free software: you can redistribute it and/or modify
 it under the terms of the GNU General Public License as published by
 the Free Software Foundation, either version 3 of the License, or
 (at your option) any later version.

 This program is distributed in the hope that it will be useful,
 but WITHOUT ANY WARRANTY; without even the implied warranty of
 MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 GNU General Public License for more details.

 You should have received a copy of the GNU General Public License
 along with this program. If not, see <https://www.gnu.org/licenses/>.
*/
import (
	"log/slog"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rhysbryant/proxylink/pkg/httputils"
)

type RequestTrackingWrapper struct {
	next                  httputils.RequestProcessor
	inProgressConnections atomic.Int64
	hostnameCache         sync.Map // Cache for reverse DNS lookups
}

type cachedHostname struct {
	hostname   string
	expiration time.Time
}

func NewRequestTrackingWrapper(next httputils.RequestProcessor) *RequestTrackingWrapper {
	rtw := &RequestTrackingWrapper{next: next}

	// Periodically log in-progress connections
	go func() {
		ticker := time.NewTicker(time.Second * 30)
		defer ticker.Stop()
		for {
			<-ticker.C
			slog.Debug("In-progress connections update", "count", rtw.inProgressConnections.Load())
		}
	}()

	// Periodically clean up expired cache entries
	go func() {
		ticker := time.NewTicker(time.Minute * 5) // Run cleanup every 5 minutes
		defer ticker.Stop()
		for {
			<-ticker.C
			rtw.cleanExpiredCacheEntries()
		}
	}()

	return rtw
}

func (rtw *RequestTrackingWrapper) cleanExpiredCacheEntries() {
	now := time.Now()
	rtw.hostnameCache.Range(func(key, value interface{}) bool {
		entry := value.(cachedHostname)
		if now.After(entry.expiration) {
			rtw.hostnameCache.Delete(key)
		}
		return true // Continue iterating
	})
}

func (rtw *RequestTrackingWrapper) getHostname(remoteAddr string) string {
	// Check if the hostname is cached
	if cached, ok := rtw.hostnameCache.Load(remoteAddr); ok {
		entry := cached.(cachedHostname)

		return entry.hostname
	}

	// Perform reverse DNS lookup
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return remoteAddr
	}
	names, err := net.LookupAddr(host)
	if err != nil || len(names) == 0 {
		return remoteAddr // Return the original address if lookup fails
	}

	// Cache the result with a TTL of 1 hour
	hostname := names[0]
	rtw.hostnameCache.Store(remoteAddr, cachedHostname{
		hostname:   hostname,
		expiration: time.Now().Add(time.Hour),
	})

	return hostname
}

func (rtw *RequestTrackingWrapper) ProcessRequest(r *http.Request, w http.ResponseWriter) error {

	rtw.inProgressConnections.Add(1)

	// Perform reverse lookup and get the hostname
	hostname := rtw.getHostname(r.RemoteAddr)

	logEntryContext := slog.With(
		"destination", r.URL.Hostname(), "destinationPort", r.URL.Port(),
		"method", r.Method, "from", r.RemoteAddr, "fromHost", hostname)

	start := time.Now()
	err := rtw.next.ProcessRequest(r, w)
	logEntryContext = logEntryContext.With("duration", time.Since(start).Milliseconds())

	rtw.inProgressConnections.Add(-1)

	if err != nil {
		logEntryContext.Info("request error", "error", err)
	} else {
		logEntryContext.Info("request processed")
	}

	return err
}
