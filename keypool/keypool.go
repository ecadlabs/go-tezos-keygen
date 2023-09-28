package keypool

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"
	bolt "go.etcd.io/bbolt"
)

type Charger interface {
	ChargeKeys(ctx context.Context, keys []uint64) error
	IsDrained(ctx context.Context, key uint64) (bool, error)
	Hash(key uint64) string
}

type Config interface {
	GetBucket() string
	GetBufferLength() int
	GetBufferThreshold() int
	GetTimeout() time.Duration
}

var (
	poolBucket  = []byte("keys")
	leaseBucket = []byte("lease")
)

type Pool struct {
	db      *bolt.DB
	charger Charger
	config  Config

	lease chan opLease
	get   chan opGet

	timeout *time.Timer
	stop    chan struct{}
	done    chan struct{}
}

type opLease struct {
	deadline time.Time
	key      chan<- uint64
	errCh    chan<- error
}

type opGet struct {
	key   chan<- uint64
	errCh chan<- error
}

type lease struct {
	KeyIndex uint64
	Deadline time.Time
}

func New(db *bolt.DB, config Config, charger Charger) (*Pool, error) {
	timeout := time.NewTimer(0)
	if !timeout.Stop() {
		select {
		case <-timeout.C:
		default:
		}
	}
	p := &Pool{
		config:  config,
		db:      db,
		charger: charger,
		lease:   make(chan opLease),
		get:     make(chan opGet),
		timeout: timeout,
		done:    make(chan struct{}),
		stop:    make(chan struct{}),
	}

	err := p.db.Update(func(tx *bolt.Tx) error {
		root, err := tx.CreateBucketIfNotExists([]byte(p.config.GetBucket()))
		if err != nil {
			return err
		}
		if _, err := root.CreateBucketIfNotExists(poolBucket); err != nil {
			return err
		}
		if _, err := root.CreateBucketIfNotExists(leaseBucket); err != nil {
			return err
		}
		return p.schedule(tx)
	})
	if err != nil {
		return nil, err
	}

	go p.loop()
	return p, nil
}

func (p *Pool) Get(ctx context.Context) (uint64, error) {
	key := make(chan uint64, 1)
	errCh := make(chan error, 1)
	select {
	case p.get <- opGet{key: key, errCh: errCh}:
	case <-ctx.Done():
		return 0, ctx.Err()
	}
	select {
	case idx := <-key:
		return idx, nil
	case err := <-errCh:
		return 0, err
	case <-ctx.Done():
		return 0, ctx.Err()
	}
}

func (p *Pool) Lease(ctx context.Context, deadline time.Time) (uint64, error) {
	key := make(chan uint64, 1)
	errCh := make(chan error, 1)
	select {
	case p.lease <- opLease{key: key, errCh: errCh, deadline: deadline}:
	case <-ctx.Done():
		return 0, ctx.Err()
	}
	select {
	case idx := <-key:
		return idx, nil
	case err := <-errCh:
		return 0, err
	case <-ctx.Done():
		return 0, ctx.Err()
	}
}

func (p *Pool) Count() (int, error) {
	var cnt int
	err := p.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(p.config.GetBucket())).Bucket(poolBucket)
		cnt = b.Stats().KeyN
		return nil
	})
	return cnt, err
}

func (p *Pool) Stop(ctx context.Context) error {
	select {
	case p.stop <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	}
	select {
	case <-p.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *Pool) loop() {
	for {
		select {
		case req := <-p.get:
			var keyIndex uint64
			err := p.db.Update(func(tx *bolt.Tx) error {
				b := bucket{tx.Bucket([]byte(p.config.GetBucket())).Bucket(poolBucket)}
				if err := p.fill(tx); err != nil {
					return err
				}
				// pop first
				c := b.Cursor()
				var k uint64
				if err := c.First(&k, &keyIndex); err != nil {
					if err == errEOF {
						panic("empty bucket") // shouldn't happen
					}
					return err
				}
				return c.Delete()
			})
			if err != nil {
				req.errCh <- err
				break
			}
			req.key <- keyIndex

		case req := <-p.lease:
			var keyIndex uint64
			err := p.db.Update(func(tx *bolt.Tx) error {
				root := tx.Bucket([]byte(p.config.GetBucket()))
				poolBkt := bucket{root.Bucket(poolBucket)}
				if err := p.fill(tx); err != nil {
					return err
				}
				// pop first
				c := poolBkt.Cursor()
				var k uint64
				if err := c.First(&k, &keyIndex); err != nil {
					if err == errEOF {
						panic("empty bucket") // shouldn't happen
					}
					return err
				}
				if err := c.Delete(); err != nil {
					return err
				}

				leaseBkt := bucket{root.Bucket(leaseBucket)}
				rec := lease{
					KeyIndex: keyIndex,
					Deadline: req.deadline,
				}
				if err := leaseBkt.Put(&k, &rec); err != nil {
					return err
				}
				return p.schedule(tx)
			})
			if err != nil {
				req.errCh <- err
				break
			}
			req.key <- keyIndex

		case now := <-p.timeout.C:
			err := p.db.Update(func(tx *bolt.Tx) error {
				root := tx.Bucket([]byte(p.config.GetBucket()))
				poolBkt := bucket{root.Bucket(poolBucket)}
				leaseBkt := bucket{root.Bucket(leaseBucket)}

				c := leaseBkt.Cursor()
				var (
					k   uint64
					v   lease
					err error
				)
				for err = c.First(&k, &v); err == nil; err = c.Next(&k, &v) {
					if v.Deadline.Before(now) || v.Deadline.Equal(now) {
						ctx := context.Background()
						var cancel context.CancelFunc
						if p.config.GetTimeout() != 0 {
							ctx, cancel = context.WithTimeout(ctx, p.config.GetTimeout())
						}
						drained, err := p.charger.IsDrained(ctx, v.KeyIndex)
						if cancel != nil {
							cancel()
						}
						if err != nil {
							return err
						}
						if !drained {
							k, _ := poolBkt.NextSequence()
							// put back
							log.WithField("pkh", p.charger.Hash(v.KeyIndex)).Info("Recycling")
							if err := poolBkt.Put(&k, &v.KeyIndex); err != nil {
								return err
							}
						}
						if err := c.Delete(); err != nil {
							return err
						}
					}
				}
				if err != nil && err != errEOF {
					return err
				}
				return p.schedule(tx)
			})
			if err != nil {
				log.Error(err)
			}

		case <-p.stop:
			p.done <- struct{}{}
			return
		}
	}
}

func (p *Pool) schedule(tx *bolt.Tx) error {
	b := bucket{tx.Bucket([]byte(p.config.GetBucket())).Bucket(leaseBucket)}
	c := b.Cursor()
	var (
		i            int
		nextDeadline time.Time
		k            uint64
		v            lease
		err          error
	)
	for err = c.First(&k, &v); err == nil; err = c.Next(&k, &v) {
		if i == 0 || v.Deadline.Before(nextDeadline) {
			nextDeadline = v.Deadline
		}
		i++
	}
	if err != nil && err != errEOF {
		return err
	}
	if i != 0 {
		if !p.timeout.Stop() {
			select {
			case <-p.timeout.C:
			default:
			}
		}
		p.timeout.Reset(time.Until(nextDeadline))
	}
	return nil
}

func (p *Pool) fill(tx *bolt.Tx) error {
	b := bucket{tx.Bucket([]byte(p.config.GetBucket())).Bucket(poolBucket)}
	n := b.Stats().KeyN
	if n > p.config.GetBufferThreshold() {
		return nil
	}
	keys := make([]uint64, p.config.GetBufferLength()-n)
	for i := range keys {
		k, _ := b.NextSequence()
		keys[i] = k
		if err := b.Put(&k, &k); err != nil {
			return err
		}
	}
	ctx := context.Background()
	if p.config.GetTimeout() != 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, p.config.GetTimeout())
		defer cancel()
	}
	return p.charger.ChargeKeys(ctx, keys)
}
