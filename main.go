package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"crypto/rand"

	"github.com/ecadlabs/go-tezos-keygen/charger"
	"github.com/ecadlabs/go-tezos-keygen/config"
	"github.com/ecadlabs/go-tezos-keygen/keypool"
	"github.com/ecadlabs/go-tezos-keygen/server"
	"github.com/ecadlabs/go-tezos-keygen/server/middleware"
	"github.com/ecadlabs/go-tezos-keygen/service"
	"github.com/ecadlabs/go-tezos-keygen/utils"
	"github.com/ecadlabs/gotez/v2/client"
	log "github.com/sirupsen/logrus"
	bolt "go.etcd.io/bbolt"
)

func main() {
	var (
		networksFile string
		databaseFile string
		address      string
		level        string
		genSeed      bool
	)
	flag.StringVar(&networksFile, "n", "", "Networks configuration file")
	flag.StringVar(&databaseFile, "d", "", "Database")
	flag.StringVar(&address, "a", ":3000", "Address")
	flag.StringVar(&level, "l", "info", "Level")
	flag.BoolVar(&genSeed, "seed", false, "Generate seed and exit")
	flag.Parse()

	l, err := log.ParseLevel(level)
	if err != nil {
		log.Fatal(err)
	}
	log.SetLevel(l)

	if genSeed {
		var seed [64]byte
		if _, err := rand.Read(seed[:]); err != nil {
			log.Fatal(err)
		}
		fmt.Println(hex.EncodeToString(seed[:]))
		return
	}

	if networksFile == "" {
		networksFile = os.Getenv("KEYGEN_NETWORKS")
	}

	if databaseFile == "" {
		databaseFile = os.Getenv("KEYGEN_DB")
	}

	var rd io.Reader
	if x := os.Getenv("KEYGEN_NETWORKS_DATA"); x != "" {
		rd = bytes.NewReader([]byte(x))
	} else {
		fd, err := os.Open(networksFile)
		if err != nil {
			log.Fatal(err)
		}
		defer fd.Close()
		rd = bufio.NewReader(fd)
	}
	cfg, err := config.New(rd)
	if err != nil {
		log.Fatal(err)
	}

	log.Infof("Database file: %s", databaseFile)

	db, err := bolt.Open(databaseFile, 0600, nil)
	if err != nil {
		log.Fatal(err)
	}

	nets := make(map[string]*service.Network, len(cfg))
	for name, net := range cfg {
		client := &client.Client{
			URL:         net.GetURL(),
			DebugLogger: (*utils.DebugLogger)(log.StandardLogger()),
		}
		charger := charger.New(net, client)
		pool, err := keypool.New(db, net, charger)
		if err != nil {
			log.Fatal(err)
		}
		nets[name] = &service.Network{
			Pool:    pool,
			Charger: charger,
			Config:  net,
		}
	}

	service := service.Service{Networks: nets}
	server := server.Server{Service: &service}
	handler := server.Router()

	logger := middleware.Logging{}
	handler.Use(logger.Handler)

	srv := &http.Server{
		Handler: handler,
		Addr:    address,
	}

	errCh := make(chan error)
	go func() {
		log.Printf("HTTP server is listening for connections on %s", srv.Addr)
		errCh <- srv.ListenAndServe()
	}()

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-signalCh:
	case <-errCh:
		log.Fatal(err) // happened before shutdown
	}

	log.Info("Shutting down...")
	if err := srv.Shutdown(context.Background()); err != nil {
		log.Fatal(err)
	}
	if err := <-errCh; !errors.Is(err, http.ErrServerClosed) {
		log.Error(err)
	}
	for _, n := range nets {
		if err := n.Pool.Stop(context.Background()); err != nil {
			log.Error(err)
		}
	}
	if err := db.Close(); err != nil {
		log.Fatal(err)
	}
	log.Info("Bye")
}
