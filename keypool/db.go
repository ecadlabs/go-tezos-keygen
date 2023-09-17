package keypool

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"errors"

	bolt "go.etcd.io/bbolt"
)

type bucket struct {
	*bolt.Bucket
}

func (b *bucket) Get(key any, out any) (bool, error) {
	var k bytes.Buffer
	if err := binary.Write(&k, binary.BigEndian, key); err != nil {
		return false, err
	}
	v := b.Bucket.Get(k.Bytes())
	if v == nil {
		return false, nil
	}
	return true, gob.NewDecoder(bytes.NewReader(v)).Decode(out)
}

func (b *bucket) Put(key, value any) error {
	var k bytes.Buffer
	if err := binary.Write(&k, binary.BigEndian, key); err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(value); err != nil {
		return err
	}
	return b.Bucket.Put(k.Bytes(), buf.Bytes())
}

func (b *bucket) Cursor() *cursor {
	return &cursor{Cursor: b.Bucket.Cursor()}
}

type cursor struct {
	*bolt.Cursor
}

var errEOF = errors.New("EOF")

func (c *cursor) First(key, val any) error {
	k, v := c.Cursor.First()
	if k == nil {
		return errEOF
	}
	if err := binary.Read(bytes.NewReader(k), binary.BigEndian, key); err != nil {
		return err
	}
	return gob.NewDecoder(bytes.NewReader(v)).Decode(val)
}

func (c *cursor) Next(key, val any) error {
	k, v := c.Cursor.Next()
	if k == nil {
		return errEOF
	}
	if err := binary.Read(bytes.NewReader(k), binary.BigEndian, key); err != nil {
		return err
	}
	return gob.NewDecoder(bytes.NewReader(v)).Decode(val)
}
