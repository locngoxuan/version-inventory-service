package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"

	"github.com/julienschmidt/httprouter"
	"github.com/urfave/cli/v2"
)

var version = "unknown"

func main() {
	initializeLogger()
	app := &cli.App{
		Name:            "vis",
		Usage:           "version inventory system",
		Copyright:       "Loc Ngo <xuanloc0511@gmail.com>",
		Description:     "",
		Version:         version,
		UsageText:       "vis [global options]",
		HideHelpCommand: true,
		Action:          run,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "tls.enabled",
				Usage:   "enable https schema",
				Value:   false,
				EnvVars: []string{"VIS_TLS_ENABLED"},
			},
			&cli.IntFlag{
				Name:    "port.http",
				Usage:   "specify listener port of http",
				Value:   80,
				EnvVars: []string{"VIS_PORT_HTTP"},
			},
			&cli.IntFlag{
				Name:    "port.https",
				Usage:   "specify listener port of https",
				Value:   443,
				EnvVars: []string{"VIS_PORT_HTTPS"},
			},
			&cli.StringFlag{
				Name:    "tls.cert",
				Usage:   "specify location of cert file",
				Value:   "ssl/server.crt",
				EnvVars: []string{"VIS_TLS_CERT"},
			},
			&cli.StringFlag{
				Name:    "tls.key",
				Usage:   "specify location of key file",
				Value:   "ssl/server.key",
				EnvVars: []string{"VIS_TLS_KEY"},
			},
			&cli.StringFlag{
				Name:    "db.driver",
				Usage:   "specify database driver",
				Value:   "sqlite",
				EnvVars: []string{"VIS_DB_DRIVER"},
			},
			&cli.StringFlag{
				Name:    "db.dsn",
				Usage:   "specify database data source",
				Value:   "data/vis.sqlite",
				EnvVars: []string{"VIS_DB_DSN"},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		logger.Fatal(err)
	}

}

func run(ctx *cli.Context) (err error) {
	err = initializeDatabase(ctx.Context, ctx.String("db.driver"), ctx.String("db.dsn"))
	if err != nil {
		logger.Fatalw("failed to init database", "err", err)
	}
	defer func() {
		_ = closeDb()
	}()

	r := httprouter.New()
	r.GET("/api/v1/version", getVersion)
	r.POST("/api/v1/version", updateVersion)
	r.DELETE("/api/v1/version", rollbackVersion)
	httpServer := &http.Server{
		Addr:    fmt.Sprintf("0.0.0.0:%d", ctx.Int("port.http")),
		Handler: r,
	}
	if ctx.Bool("tls.enabled") {
		httpServer.Addr = fmt.Sprintf("0.0.0.0:%d", ctx.Int("port.https"))
		cert := ctx.String("tls.cert")
		key := ctx.String("tls.key")

		//Achieving a Perfect SSL with go: https://blog.bracebin.com/achieving-perfect-ssl-labs-score-with-go
		httpServer.TLSConfig = &tls.Config{
			MinVersion:               tls.VersionTLS12,
			CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
			PreferServerCipherSuites: true,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
				tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_RSA_WITH_AES_256_CBC_SHA,
			},
		}
		httpServer.TLSNextProto = make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0)
		logger.Infow("tls files configuration", "tls_cert", cert, "tls_key", key)
		go func() {
			e := http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", ctx.Int("port.http")), http.HandlerFunc(redirect))
			if e != nil {
				if e == http.ErrServerClosed {
					return
				}
				logger.Fatalw("failed to setup http listener for redirecting to https", "err", e)
			}
		}()
		err = httpServer.ListenAndServeTLS(cert, key)
	} else {
		err = httpServer.ListenAndServe()
	}
	if err != nil {
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	}
	return nil
}

func redirect(w http.ResponseWriter, req *http.Request) {
	target := "https://" + req.Host + req.URL.Path
	if len(req.URL.RawQuery) > 0 {
		target += "?" + req.URL.RawQuery
	}
	http.Redirect(w, req, target, http.StatusTemporaryRedirect)
}
