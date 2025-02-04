package headtracker

import (
	"context"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/smartcontractkit/chainlink/core/services/postgres"
	"github.com/smartcontractkit/chainlink/core/store/models"
)

type HeadSaver struct {
	highestSeenHead *models.Head
	orm             *ORM
	config          Config
	headMutex       sync.RWMutex
}

func NewHeadSaver(orm *ORM, config Config) *HeadSaver {
	return &HeadSaver{
		orm:    orm,
		config: config,
	}
}

// Save updates the latest block number, if indeed the latest, and persists
// this number in case of reboot. Thread safe.
func (ht *HeadSaver) Save(ctx context.Context, h models.Head) error {
	ht.headMutex.Lock()
	if h.GreaterThan(ht.highestSeenHead) {
		ht.highestSeenHead = &h
	}
	ht.headMutex.Unlock()

	err := ht.orm.IdempotentInsertHead(ctx, h)
	if ctx.Err() != nil {
		return nil
	} else if err != nil {
		return err
	}
	return ht.orm.TrimOldHeads(ctx, ht.config.EthHeadTrackerHistoryDepth())
}

// HighestSeenHead returns the block header with the highest number that has been seen, or nil
func (ht *HeadSaver) HighestSeenHead() *models.Head {
	ht.headMutex.RLock()
	defer ht.headMutex.RUnlock()

	if ht.highestSeenHead == nil {
		return nil
	}
	h := *ht.highestSeenHead
	return &h
}

func (ht *HeadSaver) IdempotentInsertHead(ctx context.Context, head models.Head) error {
	return ht.orm.IdempotentInsertHead(ctx, head)
}

func (ht *HeadSaver) SetHighestSeenHeadFromDB() (*models.Head, error) {
	ht.headMutex.RLock()
	defer ht.headMutex.RUnlock()

	head, err := ht.HighestSeenHeadFromDB()
	if err != nil {
		return nil, err
	}
	ht.highestSeenHead = head
	return head, nil
}

func (ht *HeadSaver) HighestSeenHeadFromDB() (*models.Head, error) {
	ctxQuery, _ := postgres.DefaultQueryCtx()
	return ht.orm.LastHead(ctxQuery)
}

func (ht *HeadSaver) Chain(ctx context.Context, hash common.Hash, depth uint) (models.Head, error) {
	return ht.orm.Chain(ctx, hash, depth)
}

func (ht *HeadSaver) HeadByHash(ctx context.Context, hash common.Hash) (*models.Head, error) {
	return ht.orm.HeadByHash(ctx, hash)
}
