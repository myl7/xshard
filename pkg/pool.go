package pkg

import (
	"sync"
)

type TxPool struct {
	waitingSubPool map[string]Tx
	pendingSubPool map[string]Tx
	mutex          sync.Mutex
}

func NewTxPool() *TxPool {
	return &TxPool{
		waitingSubPool: make(map[string]Tx),
		pendingSubPool: make(map[string]Tx),
	}
}

func (p *TxPool) Add(tx Tx) int {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.waitingSubPool[tx.TxHash] = tx
	return len(p.waitingSubPool)
}

func (p *TxPool) SelectTxesForBlock() []Tx {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if len(p.waitingSubPool) < Cfg.BlockSize {
		return nil
	}

	var txes []Tx
	size := 0
	for k, tx := range p.waitingSubPool {
		txes = append(txes, tx)
		delete(p.waitingSubPool, k)
		p.pendingSubPool[k] = tx

		size += 1
		if size >= Cfg.BlockSize {
			break
		}
	}

	return txes
}

func (p *TxPool) RemoveTxesForBlock(txes []Tx) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	for _, tx := range txes {
		delete(p.pendingSubPool, tx.TxHash)
	}
}

func (p *TxPool) RemoveWaitingTxesForBlock(txes []Tx) []Tx {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	var removedTxes []Tx
	for _, tx := range txes {
		t, exist := p.waitingSubPool[tx.TxHash]
		if exist {
			delete(p.waitingSubPool, tx.TxHash)
			removedTxes = append(removedTxes, t)
		}
	}
	return removedTxes
}
