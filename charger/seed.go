package charger

import (
	"github.com/ecadlabs/gotez/v2/crypt"
	"github.com/ecadlabs/hdw"
	"github.com/ecadlabs/hdw/ed25519"
)

type Seed []byte

func (s Seed) Derive(index uint64) (crypt.PrivateKey, error) {
	root := ed25519.NewKeyFromSeed(s)
	priv, err := root.DerivePath(hdw.Path{uint32(index>>32) | hdw.Hard, uint32(index&0xffffffff) | hdw.Hard})
	if err != nil {
		return nil, err
	}
	return crypt.NewPrivateKeyFrom(priv.Naked())
}
