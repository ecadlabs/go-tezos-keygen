package config

import (
	"encoding/hex"
	"io"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/ecadlabs/go-tezos-keygen/charger"
	tz "github.com/ecadlabs/gotez/v2"
	"github.com/ecadlabs/gotez/v2/crypt"
	"gopkg.in/yaml.v3"
)

type networkConfig struct {
	URL             string        `yaml:"url"`
	ChainID         *tz.ChainID   `yaml:"chain-id"`
	Seed            string        `yaml:"seed"`
	SeedFile        string        `yaml:"seed-file"`
	PrivateKey      string        `yaml:"private-key"`
	PrivateKeyFile  string        `yaml:"private-key-file"`
	MinBalance      *big.Int      `yaml:"min-balance"`
	Amount          *big.Int      `yaml:"amount"`
	OpsPerGroup     int           `yaml:"ops-per-group"`
	LeaseTime       time.Duration `yaml:"lease-time"`
	BufferLength    int           `yaml:"buffer-length"`
	BufferThreshold int           `yaml:"buffer-threshold"`
	Timeout         time.Duration `yaml:"rpc-timeout"`
}

type NetworkConfig struct {
	*networkConfig
	name       string
	seed       charger.Seed
	privateKey crypt.PrivateKey
}

func (n *NetworkConfig) GetURL() string                  { return n.URL }
func (n *NetworkConfig) GetChainID() *tz.ChainID         { return n.ChainID }
func (n *NetworkConfig) GetSeed() charger.Seed           { return n.seed }
func (n *NetworkConfig) GetPrivateKey() crypt.PrivateKey { return n.privateKey }
func (n *NetworkConfig) GetMinBalance() *big.Int         { return n.MinBalance }
func (n *NetworkConfig) GetAmount() *big.Int             { return n.Amount }
func (n *NetworkConfig) GetOpsPerGroup() int             { return n.OpsPerGroup }
func (n *NetworkConfig) GetLeaseTime() time.Duration     { return n.LeaseTime }
func (n *NetworkConfig) GetBucket() string               { return n.name }
func (n *NetworkConfig) GetBufferLength() int            { return n.BufferLength }
func (n *NetworkConfig) GetBufferThreshold() int         { return n.BufferThreshold }
func (n *NetworkConfig) GetTimeout() time.Duration       { return n.Timeout }

type Config map[string]*NetworkConfig

func inlineOrFile(inline, file string) ([]byte, error) {
	if inline != "" {
		return []byte(inline), nil
	}
	return os.ReadFile(file)
}

func New(rd io.Reader) (Config, error) {
	var raw map[string]*networkConfig
	if err := yaml.NewDecoder(rd).Decode(&raw); err != nil {
		return nil, err
	}
	out := make(Config, len(raw))
	for name, data := range raw {
		envPrefix := strings.ToUpper(name)
		var privData []byte
		if v := os.Getenv(envPrefix + "_PRIVATE_KEY"); v != "" {
			privData = []byte(v)
		} else {
			var err error
			privData, err = inlineOrFile(data.PrivateKey, data.PrivateKeyFile)
			if err != nil {
				return nil, err
			}
		}
		priv, err := crypt.ParsePrivateKey(privData)
		if err != nil {
			return nil, err
		}

		var seedData []byte
		if v := os.Getenv(envPrefix + "_SEED"); v != "" {
			seedData = []byte(v)
		} else {
			var err error
			seedData, err = inlineOrFile(data.Seed, data.SeedFile)
			if err != nil {
				return nil, err
			}
		}
		seed := make([]byte, hex.DecodedLen(len(seedData)))
		if _, err := hex.Decode(seed, seedData); err != nil {
			return nil, err
		}

		out[name] = &NetworkConfig{
			networkConfig: data,
			name:          name,
			seed:          seed,
			privateKey:    priv,
		}
	}
	return out, nil
}
