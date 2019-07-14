package main

// oauth2
// github.com/rdeusser/oauth2-proxy

import (
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"go.uber.org/zap"

	"github.com/rdeusser/oauth2-proxy/handlers"
	"github.com/rdeusser/oauth2-proxy/pkg/cfg"
	"github.com/rdeusser/oauth2-proxy/pkg/timelog"
	tran "github.com/rdeusser/oauth2-proxy/pkg/transciever"
)

// version and semver get overwritten by build with
// go build -i -v -ldflags="-X main.version=$(git describe --always --long) -X main.semver=v$(git semver get)"
var (
	version   = "undefined"
	builddt   = "undefined"
	host      = "undefined"
	semver    = "undefined"
	branch    = "undefined"
	staticDir = "/static/"
	logger    = cfg.Cfg.Logger
	fastlog   = cfg.Cfg.FastLogger
)

// fwdToZapWriter allows us to use the zap.Logger as our http.Server ErrorLog
// see https://stackoverflow.com/questions/52294334/net-http-set-custom-logger
type fwdToZapWriter struct {
	logger *zap.Logger
}

func (fw *fwdToZapWriter) Write(p []byte) (n int, err error) {
	fw.logger.Error(string(p))
	return len(p), nil
}

func main() {
	var listen = cfg.Cfg.Listen + ":" + strconv.Itoa(cfg.Cfg.Port)
	logger.Infow("starting "+cfg.Branding.CcName,
		// "semver":    semver,
		"version", version,
		"buildtime", builddt,
		"buildhost", host,
		"branch", branch,
		"semver", semver,
		"listen", listen,
		"oauth.provider", cfg.GenOAuth.Provider)

	muxR := mux.NewRouter()

	authH := http.HandlerFunc(handlers.ValidateRequestHandler)
	muxR.HandleFunc("/validate", timelog.TimeLog(authH))
	muxR.HandleFunc("/_external-auth-{id}", timelog.TimeLog(authH))

	loginH := http.HandlerFunc(handlers.LoginHandler)
	muxR.HandleFunc("/login", timelog.TimeLog(loginH))

	logoutH := http.HandlerFunc(handlers.LogoutHandler)
	muxR.HandleFunc("/logout", timelog.TimeLog(logoutH))

	callH := http.HandlerFunc(handlers.CallbackHandler)
	muxR.HandleFunc("/auth", timelog.TimeLog(callH))

	healthH := http.HandlerFunc(handlers.HealthcheckHandler)
	muxR.HandleFunc("/healthcheck", timelog.TimeLog(healthH))

	if logger.Desugar().Core().Enabled(zap.DebugLevel) {
		path, err := filepath.Abs(staticDir)
		if err != nil {
			logger.Errorf("couldn't find static assets at %s", path)
		}
		logger.Debugf("serving static files from %s", path)
	}
	// https://golangcode.com/serve-static-assets-using-the-mux-router/
	muxR.PathPrefix(staticDir).Handler(http.StripPrefix(staticDir, http.FileServer(http.Dir("."+staticDir))))

	if cfg.Cfg.WebApp {
		logger.Info("enabling websocket")
		tran.ExplicitInit()
		muxR.Handle("/ws", tran.WS)
	}

	// socketio := tran.NewServer()
	// muxR.Handle("/socket.io/", cors.AllowAll(socketio))
	// http.Handle("/socket.io/", tran.Server)

	srv := &http.Server{
		Handler: muxR,
		Addr:    listen,
		// Good practice: enforce timeouts for servers you create!
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
		ErrorLog:     log.New(&fwdToZapWriter{fastlog}, "", 0),
	}

	log.Fatal(srv.ListenAndServe())

}
