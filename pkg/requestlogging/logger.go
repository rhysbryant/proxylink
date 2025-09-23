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
	"net/http"
	"sync/atomic"
	"time"

	"github.com/rhysbryant/proxylink/pkg/httputils"
)

type RequestTrackingWrapper struct {
	next                  httputils.RequestProcessor
	inProgressConnections atomic.Int64
}

func NewRequestTrackingWrapper(next httputils.RequestProcessor) *RequestTrackingWrapper {
	rtw := &RequestTrackingWrapper{next: next}
	go func() {
		ticker := time.NewTicker(time.Second * 30)
		for {
			<-ticker.C
			slog.Debug("In-progress connections update", "count", rtw.inProgressConnections.Load())
		}
	}()
	return rtw
}

func (rtw *RequestTrackingWrapper) ProcessRequest(r *http.Request, w http.ResponseWriter) error {

	rtw.inProgressConnections.Add(1)

	logEntryContext := slog.With("target", r.URL.String(),
		"method", r.Method, "from", r.RemoteAddr)

	logEntryContext.Info("Processing request")

	start := time.Now()
	err := rtw.next.ProcessRequest(r, w)
	logEntryContext = logEntryContext.With("duration", time.Since(start).String())

	rtw.inProgressConnections.Add(-1)

	if err != nil {
		logEntryContext.Info("Error processing request", "error", err)
	} else {
		logEntryContext.Info("Request processed successfully")
	}

	return err
}
