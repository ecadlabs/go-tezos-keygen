package service

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/ecadlabs/go-tezos-keygen/charger"
	"github.com/ecadlabs/go-tezos-keygen/keypool"
	"github.com/ecadlabs/go-tezos-keygen/server"
	tz "github.com/ecadlabs/gotez/v2"
	"github.com/ecadlabs/gotez/v2/client"
	"github.com/ecadlabs/gotez/v2/encoding"
	log "github.com/sirupsen/logrus"
)

type NetworkConfig interface {
	GetSeed() charger.Seed
	GetLeaseTime() time.Duration
}

type Network struct {
	Pool    *keypool.Pool
	Charger *charger.Charger
	Config  NetworkConfig
}

type Service struct {
	Networks map[string]*Network
}

func logError(err error) {
	log.Error(err)
	log.Debugf("%#v", err)

	var e *encoding.Error
	if errors.As(err, &e) {
		log.Debug(e.Path)
	} else {
		var e *client.Error
		if errors.As(err, &e) {
			log.Debug(spew.Sdump(e.Body))
		}
	}
}

func (s *Service) Pop(ctx context.Context, network string) (tz.PrivateKey, error) {
	net, ok := s.Networks[network]
	if !ok {
		return nil, server.ErrUnknownNetwork
	}
	index, err := net.Pool.Get(ctx)
	if err != nil {
		logError(err)
		return nil, err
	}
	priv, err := net.Config.GetSeed().Derive(index)
	if err != nil {
		return nil, err
	}
	return priv.ToProtocol(), nil
}

func (s *Service) Status(ctx context.Context, network string) (*server.NetworkStatus, error) {
	net, ok := s.Networks[network]
	if !ok {
		return nil, server.ErrUnknownNetwork
	}
	balance, err := net.Charger.GetFunds(ctx)
	if err != nil {
		logError(err)
		return nil, err
	}
	cnt, err := net.Pool.Count()
	if err != nil {
		return nil, err
	}
	return &server.NetworkStatus{
		Count:   cnt,
		Balance: balance,
	}, nil
}

func (s *Service) Lease(ctx context.Context, network string) (*server.Lease, error) {
	net, ok := s.Networks[network]
	if !ok {
		return nil, server.ErrUnknownNetwork
	}
	index, err := net.Pool.Lease(ctx, time.Now().Add(net.Config.GetLeaseTime()))
	if err != nil {
		logError(err)
		return nil, err
	}
	priv, err := net.Config.GetSeed().Derive(index)
	if err != nil {
		return nil, err
	}
	return &server.Lease{
		ID:  index,
		PKH: priv.Public().Hash(),
	}, nil
}

func (s *Service) Pub(ctx context.Context, network string, id uint64) (tz.PublicKey, error) {
	net, ok := s.Networks[network]
	if !ok {
		return nil, server.ErrUnknownNetwork
	}
	priv, err := net.Config.GetSeed().Derive(id)
	if err != nil {
		return nil, err
	}
	return priv.Public().ToProtocol(), nil
}

func (s *Service) Sign(ctx context.Context, network string, id uint64, r io.Reader) (tz.Signature, error) {
	net, ok := s.Networks[network]
	if !ok {
		return nil, server.ErrUnknownNetwork
	}
	priv, err := net.Config.GetSeed().Derive(id)
	if err != nil {
		return nil, err
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	sig, err := priv.Sign(data)
	if err != nil {
		return nil, err
	}
	return sig.ToProtocol(), nil
}
