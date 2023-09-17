package charger

import (
	"context"
	"math/big"

	tz "github.com/ecadlabs/gotez/v2"
	"github.com/ecadlabs/gotez/v2/client"
	"github.com/ecadlabs/gotez/v2/crypt"
	"github.com/ecadlabs/gotez/v2/protocol/core"
	"github.com/ecadlabs/gotez/v2/protocol/latest"
	"github.com/ecadlabs/gotez/v2/teztool"
	log "github.com/sirupsen/logrus"
)

type Config interface {
	GetChainID() *tz.ChainID
	GetSeed() Seed
	GetPrivateKey() crypt.PrivateKey
	GetMinBalance() *big.Int
	GetAmount() *big.Int
	GetOpsPerGroup() int
}

type Charger struct {
	client *client.Client
	cfg    Config
}

func New(cfg Config, client *client.Client) *Charger {
	return &Charger{
		client: client,
		cfg:    cfg,
	}
}

func (c *Charger) ChargeKeys(ctx context.Context, keys []uint64) error {
	amount, err := tz.NewBigUint(c.cfg.GetAmount())
	if err != nil {
		return err
	}

	tezTool := teztool.TezTool{
		Client:  c.client,
		ChainID: c.cfg.GetChainID(),
		Signer:  teztool.NewLocalSigner(c.cfg.GetPrivateKey()),
	}

	for len(keys) != 0 {
		var ops []latest.OperationContents
		for len(ops) < c.cfg.GetOpsPerGroup() && len(keys) != 0 {
			keyIndex := keys[0]
			keys = keys[1:]

			priv, err := c.cfg.GetSeed().Derive(keyIndex)
			if err != nil {
				return err
			}
			dest := priv.Public().Hash()
			log.WithField("pkh", dest).Info("Funding")
			tx := latest.Transaction{
				ManagerOperation: latest.ManagerOperation{
					Source: c.cfg.GetPrivateKey().Public().Hash(),
				},
				Amount:      amount,
				Destination: core.ImplicitContract{PublicKeyHash: dest},
			}
			ops = append(ops, &tx)
		}
		hash, err := tezTool.SignAndInject(ctx, ops, teztool.FillCounter, teztool.FillFee, teztool.FillGasLimit, teztool.FillStorageLimit)
		if err != nil {
			log.Error(err)
			return err
		}
		log.WithField("hash", hash).Info("Injected")
	}
	return nil
}

func (c *Charger) IsDrained(ctx context.Context, key uint64) (bool, error) {
	priv, err := c.cfg.GetSeed().Derive(key)
	if err != nil {
		return false, err
	}
	address := priv.Public().Hash()
	balance, err := c.getBalance(ctx, address)
	if err != nil {
		log.Error(err)
		return false, err
	}
	return balance.Cmp(c.cfg.GetMinBalance()) < 0, nil
}

func (c *Charger) GetFunds(ctx context.Context) (*big.Int, error) {
	address := c.cfg.GetPrivateKey().Public().Hash()
	return c.getBalance(ctx, address)
}

func (c *Charger) getBalance(ctx context.Context, address tz.PublicKeyHash) (*big.Int, error) {
	value, err := c.client.ContractBalance(ctx, &client.ContractRequest{
		Chain: c.cfg.GetChainID().String(),
		Block: "head",
		ID:    core.ImplicitContract{PublicKeyHash: address},
	})
	if err != nil {
		return nil, err
	}
	return value.Int(), nil
}
