package main

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
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/kardianos/service"
	"github.com/rhysbryant/proxylink/pkg/bridgeserver"
	"github.com/rhysbryant/proxylink/pkg/config"
	"github.com/rhysbryant/proxylink/pkg/httputils"
	"github.com/rhysbryant/proxylink/pkg/proxy"
	"github.com/rhysbryant/proxylink/pkg/requestlogging"
	"github.com/rhysbryant/proxylink/pkg/rulesengine"
	"golang.org/x/crypto/acme/autocert"
)

// Program structure for service
type program struct {
	server *http.Server
	config *config.Config
}

func (p *program) Start(s service.Service) error {
	// Start should not block. Do the actual work in a goroutine.
	go p.run()
	return nil
}

func (p *program) run() {
	// Start the HTTP server
	if p.config.TLS.LetsEncrypt {
		certManager := autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(p.config.TLS.Domain),
			Cache:      autocert.DirCache("certs"), // Directory for storing certificates
		}
		p.server.TLSConfig = certManager.TLSConfig()

		log.Printf("Starting proxy server with Let's Encrypt on %s\n", p.config.ListenAddr)
		log.Fatal(p.server.ListenAndServeTLS("", ""))
	} else {
		log.Printf("Starting proxy server in %s mode on %s\n", p.config.Mode, p.config.ListenAddr)
		if p.config.TLS.CertFile != "" && p.config.TLS.KeyFile != "" {
			log.Fatal(p.server.ListenAndServeTLS(p.config.TLS.CertFile, p.config.TLS.KeyFile))
		} else {
			log.Fatal(p.server.ListenAndServe())
		}
	}
}

func (p *program) Stop(s service.Service) error {
	// Gracefully shut down the server
	if err := p.server.Close(); err != nil {
		return fmt.Errorf("failed to stop server: %w", err)
	}
	log.Println("Server stopped")
	return nil
}

func main() {
	// Define CLI flags
	var mode string
	var nextProxyAddr string
	var certFile string
	var keyFile string
	var wsKey string
	var wsKeyBytes []byte
	var logLevel string
	var configFileName string
	var useLetsEncrypt bool
	var domain string
	var serviceFlag string

	flag.StringVar(&mode, "mode", "standalone", "Mode of operation: standalone, bridge, or exit")
	flag.StringVar(&nextProxyAddr, "next", "", "Address of the next proxy (required in bridge mode)")
	var listenAddr string
	flag.StringVar(&listenAddr, "listen", ":8080", "Address to listen on")
	flag.StringVar(&certFile, "tls-cert", "", "Path to TLS certificate file")
	flag.StringVar(&keyFile, "tls-key", "", "Path to TLS key file")
	flag.StringVar(&wsKey, "ws-key", "", "32-byte key for encrypting WebSocket traffic (optional)")
	flag.StringVar(&logLevel, "log-level", "error", "Logging level: debug, info, warn, error")
	flag.StringVar(&configFileName, "config", "config.yml", "Path to configuration file")
	flag.BoolVar(&useLetsEncrypt, "lets-encrypt", false, "Enable Let's Encrypt support")
	flag.StringVar(&domain, "domain", "", "Domain name for Let's Encrypt (required if --lets-encrypt is enabled)")
	flag.StringVar(&serviceFlag, "service", "", "Control the system service (install, uninstall, start, stop)")
	flag.Parse()

	// Load configuration
	cfg, err := config.LoadConfig(configFileName)
	if err != nil {
		log.Println("failed to load config:", err)
		cfg = &config.Config{}
		cfg.ListenAddr = listenAddr
		cfg.Mode = mode
		cfg.TLS.CertFile = certFile
		cfg.TLS.KeyFile = keyFile
		cfg.TLS.LetsEncrypt = useLetsEncrypt
		cfg.TLS.Domain = domain
	}

	if wsKey == "" {
		wsKey = cfg.Key
	}

	if wsKey != "" && len(wsKey) != 64 {
		log.Fatal("ws-key must be 32 bytes long", len(wsKey))
	} else if wsKey != "" {
		var err error
		wsKeyBytes, err = hex.DecodeString(wsKey)
		if err != nil {
			log.Fatal("failed to decode ws-key:", err)
		}
	}

	if useLetsEncrypt && domain == "" {
		log.Fatal("Domain name must be specified when using Let's Encrypt")
	}

	var rp httputils.RequestProcessor

	switch cfg.Mode {
	case "standalone":
		rp = proxy.NewDirectHTTPProxy()
	case "bridge":
		rp = proxy.NewWSBridgeProxyClient(nextProxyAddr, wsKeyBytes)
	case "exit":
		rp = bridgeserver.NewBridgeServer(wsKeyBytes)
	default:
		rp = proxy.NewDirectHTTPProxy()
	}

	if len(cfg.Rules) > 0 {
		rulesEng := rulesengine.NewRulesEngine(cfg.Rules)
		rw := rulesengine.NewRequestWrapper(rulesEng)

		rw.AddProxyProvider(rulesengine.DefaultProviderName, proxy.NewDirectHTTPProxy())

		for _, entry := range rulesEng.GetExitNodes() {
			wsKeyBytes, err = hex.DecodeString(entry.Key)
			if err != nil {
				log.Fatal("failed to decode ws-key:", err)
			}

			if len(wsKeyBytes) != 32 {
				log.Fatal("ws-key must be 32 bytes long", len(wsKeyBytes))
			}

			rw.AddProxyProvider(entry.URL, proxy.NewWSBridgeProxyClient(entry.URL, wsKeyBytes))
		}

		rp = rw
	}

	rp = requestlogging.NewRequestTrackingWrapper(rp)

	server := &http.Server{
		Addr: cfg.ListenAddr,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rp.ProcessRequest(r, w)
		}),
	}

	var serviceArgs []string
	if configFileName != "" {
		serviceArgs = append(serviceArgs, "-config", configFileName)
	}

	// Service setup
	svcConfig := &service.Config{
		Name:        "WebProxy",
		DisplayName: "Web Proxy Service",
		Description: "A web proxy service with tunneling and rules capabilities.",
		Arguments:   serviceArgs,
	}

	prg := &program{
		server: server,
		config: cfg,
	}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		log.Fatalf("Failed to create service: %v", err)
	}

	if serviceFlag != "" {
		// Handle service control commands
		err := service.Control(s, serviceFlag)
		if err != nil {
			log.Fatalf("Failed to %s service: %v", serviceFlag, err)
		}
		log.Printf("Service %s successfully", serviceFlag)
		return
	}

	// Run the service
	err = s.Run()
	if err != nil {
		log.Fatalf("Service failed: %v", err)
	}
}
