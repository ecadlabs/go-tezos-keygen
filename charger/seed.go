package charger

import (
	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/ecadlabs/gotez/v2/crypt"
	"github.com/ecadlabs/hdw"
	"github.com/ecadlabs/hdw/ecdsa"
)

type Seed []byte

func (s Seed) Derive(index uint64) (crypt.PrivateKey, error) {
	root, err := ecdsa.NewKeyFromSeed(s, secp256k1.S256())
	if err != nil {
		panic(err) // the curve is hardcoded
	}
	priv, err := root.DerivePath(hdw.Path{uint32(index >> 32), uint32(index & 0xffffffff)})
	if err != nil {
		return nil, err
	}
	return crypt.NewPrivateKeyFrom(priv.Naked())
}
