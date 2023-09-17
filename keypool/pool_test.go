package keypool_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/ecadlabs/go-tezos-keygen/keypool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	bolt "go.etcd.io/bbolt"
)

type ChargerMock struct {
	mock.Mock
}

func (c *ChargerMock) ChargeKeys(ctx context.Context, keys []uint64) error {
	args := c.Called(keys)
	return args.Error(0)
}

func (c *ChargerMock) IsDrained(ctx context.Context, key uint64) (bool, error) {
	args := c.Called(key)
	return args.Bool(0), args.Error(1)
}

type config struct {
	bucket          string
	bufferLength    int
	bufferThreshold int
	timeout         time.Duration
}

func (n *config) GetBucket() string         { return n.bucket }
func (n *config) GetBufferLength() int      { return n.bufferLength }
func (n *config) GetBufferThreshold() int   { return n.bufferThreshold }
func (n *config) GetTimeout() time.Duration { return n.timeout }

func TestPool(t *testing.T) {
	fd, err := os.CreateTemp("", "bolt")
	require.NoError(t, err)
	dbName := fd.Name()
	fd.Close()
	defer os.Remove(dbName)

	db, err := bolt.Open(dbName, 0600, nil)
	require.NoError(t, err)
	defer db.Close()

	charger := ChargerMock{}
	charger.On("ChargeKeys", []uint64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}).Return(nil)
	charger.On("ChargeKeys", []uint64{11, 12, 13, 14, 15, 16, 17, 18, 19, 20}).Return(nil)
	charger.On("ChargeKeys", []uint64{21, 22, 23, 24, 25, 26, 27, 28, 29, 30}).Return(nil)
	charger.On("IsDrained", uint64(21)).Return(false, nil)

	pool, err := keypool.New(db, &config{
		bucket:          "test",
		bufferLength:    10,
		bufferThreshold: 0,
		timeout:         0,
	}, &charger)
	require.NoError(t, err)

	// test get
	for n := 0; n < 20; n++ {
		idx, err := pool.Get(context.Background())
		require.NoError(t, err)
		assert.Equal(t, uint64(n+1), idx)
	}

	// test lease
	idx, err := pool.Lease(context.Background(), time.Now().Add(time.Second/2))
	require.NoError(t, err)
	assert.Equal(t, uint64(21), idx)
	<-time.After(time.Second)

	for n := 0; n < 9; n++ {
		idx, err := pool.Get(context.Background())
		require.NoError(t, err)
		assert.Equal(t, uint64(n+22), idx)
	}
	// 21 comes last
	idx, err = pool.Get(context.Background())
	require.NoError(t, err)
	assert.Equal(t, uint64(21), idx)

	charger.AssertExpectations(t)
	require.NoError(t, pool.Stop(context.Background()))
}
