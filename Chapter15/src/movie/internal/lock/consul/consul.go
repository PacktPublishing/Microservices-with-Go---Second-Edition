package consul

import (
	"context"
	"errors"

	"github.com/hashicorp/consul/api"
	"go.uber.org/zap"
)

// LockProvider defines a Consul-based lock provider.
type LockProvider struct {
	client    *api.Client
	logger    *zap.Logger
	sessionID string
}

// NewLockProvider creates a Consul-based lock provider.
func NewLockProvider(logger *zap.Logger, addr string) (*LockProvider, error) {
	config := api.DefaultConfig()
	config.Address = addr
	client, err := api.NewClient(config)
	if err != nil {
		return nil, err
	}
	sessionID, _, err := client.Session().Create(&api.SessionEntry{
		TTL: "10s", // Lock expires if our processor instance gets unavailable after 10s.
	}, nil)
	if err != nil {
		return nil, err
	}
	return &LockProvider{client, logger, sessionID}, nil
}

// Acquire acquires a lock for the given key. If successful, it returns a function that releases the lock.
func (p *LockProvider) Acquire(ctx context.Context, key string) (bool, func() error, error) {
	kvPair := &api.KVPair{
		Key:     key,
		Value:   []byte("locked"),
		Session: p.sessionID,
	}
	p.logger.Info("Acquiring a lock", zap.String("key", key), zap.String("sessionID", p.sessionID))
	acquired, _, err := p.client.KV().Acquire(kvPair, nil)
	if err != nil {
		return false, nil, err
	}
	if !acquired {
		return false, nil, errors.New("failed to acquire lock")
	}
	return true, func() error {
		return releaseLock(p, key, p.sessionID, kvPair)
	}, nil
}

// Close closes the lock provider.
func (p *LockProvider) Close() error {
	p.logger.Info("Destroying a session", zap.String("sessionID", p.sessionID))
	if _, err := p.client.Session().Destroy(p.sessionID, nil); err != nil {
		return err
	}
	return nil
}

func releaseLock(p *LockProvider, key string, sessionID string, kvPair *api.KVPair) error {
	p.logger.Info("Releasing a lock", zap.String("key", key), zap.String("sessionID", sessionID))
	_, _, err := p.client.KV().Release(kvPair, nil)
	return err
}
