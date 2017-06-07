package node

import (
	"DNA/common"
	"DNA/common/log"
	"DNA/core/ledger"
	"DNA/core/transaction"
	"DNA/errors"
	msg "DNA/net/message"
	. "DNA/net/protocol"
	va "DNA/core/validation"
	"fmt"
	"sync"
)

type TXNPool struct {
	sync.RWMutex
	txnCnt uint64
	list   map[common.Uint256]*transaction.Transaction
}

func (txnPool *TXNPool) GetTransaction(hash common.Uint256) *transaction.Transaction {
	txnPool.RLock()
	defer txnPool.RUnlock()
	txn := txnPool.list[hash]
	return txn
}

func (txnPool *TXNPool) AppendTxnPool(txn *transaction.Transaction) bool {
	//verify transaction with Concurrency
	if err := va.VerifyTransaction(txn); err != nil {
		log.Warn(fmt.Sprintf("Transaction hash=%x, verification failed with error=%s\n"),txn.Hash(),err)
		return false
	}
	if err := va.VerifyTransactionWithLedger(txn, ledger.DefaultLedger); err != nil {
		log.Warn(fmt.Sprintf("Transaction hash=%x, verification failed with ledger, error=%s\n"),txn.Hash(),err)
		return false
	}

	//verify transaction by pool with lock
	txnPool.Lock()
	defer txnPool.Unlock()
	if err := va.VerifyTransactionWithTxPool(txn,txnPool.getlist()); err != nil {
		log.Warn(fmt.Sprintf("Transaction hash=%x, verification failed with TxPool, error=%s\n"),txn.Hash(),err)
		return false
	}
	txnPool.appendToProcessList(txn)
	return true
}

// Attention: clean the trasaction Pool after the consensus confirmed all of the transcation
func (txnPool *TXNPool) GetTxnPool(cleanPool bool) map[common.Uint256]*transaction.Transaction {
	txnPool.Lock()
	defer txnPool.Unlock()

	list := txnPool.list
	result :=DeepCopy(list)
	if cleanPool == true {
		txnPool.init()
	}
	return result
}

func DeepCopy(mapIn map[common.Uint256]*transaction.Transaction) map[common.Uint256]*transaction.Transaction {
	reply := make(map[common.Uint256]*transaction.Transaction)
	for k, v := range mapIn {
		reply[k] = v
	}
	return reply
}

// Attention: clean the trasaction Pool with committed transactions.
func (txnPool *TXNPool) cleanTxnPool(txs []*transaction.Transaction) error {
	txsNum := len(txs)
	cleaned := 0
	// skip the first bookkeeping transaction
	for _, tx := range txs[1:] {
		delete(txnPool.list, tx.Hash())
		cleaned++
	}
	txnPool.txnCnt = uint64(len(txnPool.list))
	if txsNum-cleaned != 1 {
		log.Info(fmt.Sprintf("The Transactions num Unmatched. Expect %d, got %d .\n", txsNum, cleaned))
	}
	log.Debug(fmt.Sprintf("[cleanTxnPool], Requested %d clean, %d transactions cleaned from localNode.TransPool and remains %d still in TxPool", txsNum, cleaned, txnPool.txnCnt))
	return nil
}

func (txnPool *TXNPool) init() {
	txnPool.list = make(map[common.Uint256]*transaction.Transaction)
	txnPool.txnCnt = 0
}

func (node *node) SynchronizeTxnPool() {
	node.nbrNodes.RLock()
	defer node.nbrNodes.RUnlock()

	for _, n := range node.nbrNodes.List {
		if n.state == ESTABLISH {
			msg.ReqTxnPool(n)
		}
	}
}

func (txnPool *TXNPool) CleanSubmittedTransactions(block *ledger.Block) error {
	txnPool.Lock()
	defer txnPool.Unlock()
	log.Debug()

	err := txnPool.cleanTxnPool(block.Transactions)
	if err != nil {
		return errors.NewDetailErr(err, errors.ErrNoCode, "[TxnPool], CleanSubmittedTransactions failed.")
	}
	return nil
}

//Add transaction to ProcessList
func (txp *TXNPool) appendToProcessList(txs *transaction.Transaction) {
	txp.list[txs.Hash()] = txs
	txp.txnCnt++
}

//Get the transactions as array.
func (txp *TXNPool) getlist() []*transaction.Transaction {
	txs := make([]*transaction.Transaction,0,txp.txnCnt)
	for _, v := range txp.list {
		txs = append(txs,v)
	}

	return txs
}