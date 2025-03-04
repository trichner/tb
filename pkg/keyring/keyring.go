package keyring

import (
	"errors"

	zk "github.com/zalando/go-keyring"
)

var (
	ErrNotFound = errors.New("secret not found in keyring")
	ErrTooBig   = errors.New("secret too big for keyring")
)

type Item struct {
	Secret string
}

type Ring struct {
	serviceName string
}

func Open(service string) (*Ring, error) {
	return &Ring{serviceName: service}, nil
}

func (r *Ring) Put(name string, item *Item) error {
	if item == nil {
		return zk.Delete(r.serviceName, name)
	}

	err := zk.Set(r.serviceName, name, item.Secret)
	if errors.Is(err, zk.ErrSetDataTooBig) {
		return ErrTooBig
	}
	return err
}

func (r *Ring) Get(name string) (*Item, error) {
	secret, err := zk.Get(r.serviceName, name)
	if errors.Is(err, zk.ErrNotFound) {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}
	return &Item{
		Secret: secret,
	}, nil
}
