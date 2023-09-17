package server

import (
	"context"
	"errors"
	"io"
	"math/big"
	"net/http"
	"strconv"

	tz "github.com/ecadlabs/gotez/v2"
	"github.com/gorilla/mux"
)

var ErrUnknownNetwork = errors.New("unknown network")

type NetworkStatus struct {
	Balance *big.Int `json:"balance"`
	Count   int      `json:"count"`
}

type Lease struct {
	ID  uint64           `json:"id"`
	PKH tz.PublicKeyHash `json:"pkh"`
}

type Service interface {
	Pop(ctx context.Context, network string) (tz.PrivateKey, error)
	Status(ctx context.Context, network string) (*NetworkStatus, error)
	Lease(ctx context.Context, network string) (*Lease, error)
	Pub(ctx context.Context, network string, id uint64) (tz.PublicKey, error)
	Sign(ctx context.Context, network string, id uint64, r io.Reader) (tz.Signature, error)
}

type Server struct {
	Service Service
}

func serviceError(w http.ResponseWriter, err error) {
	var status int
	if errors.Is(err, ErrUnknownNetwork) {
		status = http.StatusNotFound
	} else {
		status = http.StatusInternalServerError
	}
	jsonError(w, err, status)
}

func (s *Server) popHandler(w http.ResponseWriter, r *http.Request) {
	net := mux.Vars(r)["net"]
	key, err := s.Service.Pop(r.Context(), net)
	if err != nil {
		serviceError(w, err)
		return
	}
	jsonResponse(w, 200, key)
}

func (s *Server) countHandler(w http.ResponseWriter, r *http.Request) {
	net := mux.Vars(r)["net"]
	status, err := s.Service.Status(r.Context(), net)
	if err != nil {
		serviceError(w, err)
		return
	}
	jsonResponse(w, 200, status)
}

func (s *Server) leaseHandler(w http.ResponseWriter, r *http.Request) {
	net := mux.Vars(r)["net"]
	lease, err := s.Service.Lease(r.Context(), net)
	if err != nil {
		serviceError(w, err)
		return
	}
	jsonResponse(w, 200, lease)
}

func (s *Server) pkHandler(w http.ResponseWriter, r *http.Request) {
	net := mux.Vars(r)["net"]
	id, _ := strconv.ParseUint(mux.Vars(r)["id"], 10, 64)
	pk, err := s.Service.Pub(r.Context(), net, id)
	if err != nil {
		serviceError(w, err)
		return
	}
	type response struct {
		PublicKey tz.PublicKey `json:"public_key"`
	}
	jsonResponse(w, 200, &response{PublicKey: pk})
}

func (s *Server) signHandler(w http.ResponseWriter, r *http.Request) {
	net := mux.Vars(r)["net"]
	id, _ := strconv.ParseUint(mux.Vars(r)["id"], 10, 64)
	sig, err := s.Service.Sign(r.Context(), net, id, r.Body)
	if err != nil {
		serviceError(w, err)
		return
	}
	type response struct {
		Signature tz.Signature `json:"signature"`
	}
	jsonResponse(w, 200, &response{Signature: sig})
}

func (s *Server) Router() *mux.Router {
	r := mux.NewRouter()
	r.Methods("POST").Path("/{net}").HandlerFunc(s.popHandler)
	r.Methods("GET").Path("/{net}").HandlerFunc(s.countHandler)
	r.Methods("POST").Path("/{net}/ephemeral").HandlerFunc(s.leaseHandler)
	r.Methods("GET").Path("/{net}/ephemeral/{id:[0-9]+}/keys/{key}").HandlerFunc(s.pkHandler)
	r.Methods("POST").Path("/{net}/ephemeral/{id:[0-9]+}/keys/{key}").HandlerFunc(s.signHandler)
	return r
}
