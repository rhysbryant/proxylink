package rulesengine

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
	"fmt"
	"net/http"

	"github.com/rhysbryant/proxylink/pkg/httputils"
)

const DefaultProviderName = "DIRECT"

type RequestWrapper struct {
	proxyProviders map[string]httputils.RequestProcessor
	rulesEngine    *RulesEngine
}

func NewRequestWrapper(rulesEngine *RulesEngine) *RequestWrapper {

	return &RequestWrapper{rulesEngine: rulesEngine, proxyProviders: map[string]httputils.RequestProcessor{}}
}

func (rw *RequestWrapper) AddProxyProvider(name string, provider httputils.RequestProcessor) {
	rw.proxyProviders[name] = provider
}

func (rw *RequestWrapper) ProcessRequest(r *http.Request, w http.ResponseWriter) error {

	result := rw.rulesEngine.FindMatch(r.URL, r.RemoteAddr)
	fmt.Println("Matched rule:", result)
	if result.Block {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return nil
	} else {
		//default to direct if no exit node specified
		var providerName = DefaultProviderName
		if result.Exit != nil {
			providerName = result.Exit.URL
		}

		provider, ok := rw.proxyProviders[providerName]
		if !ok {
			http.Error(w, "Proxy provider not found", http.StatusInternalServerError)
			return nil
		}

		return provider.ProcessRequest(r, w)
	}

}
