package validation

import (
	"DNA/common"
	"DNA/common/log"
	"DNA/core/ledger"
	tx "DNA/core/transaction"
	"DNA/core/transaction/payload"
	"errors"
	"fmt"
	"math"
)

// VerifyTransaction verifys received single transaction
func VerifyTransaction(Tx *tx.Transaction) error {

	if err := CheckDuplicateInput(Tx); err != nil {
		return err
	}

	if err := CheckAssetPrecision(Tx); err != nil {
		return err
	}

	if err := CheckTransactionBalance(Tx); err != nil {
		return err
	}

	if err := CheckAttributeProgram(Tx); err != nil {
		return err
	}

	if err := CheckTransactionContracts(Tx); err != nil {
		return err
	}

	return nil
}

// VerifyTransactionWithTxPool verifys a transaction with current transaction pool in memory
func VerifyTransactionWithTxPool(Tx *tx.Transaction, TxPool []*tx.Transaction) error {
	if err := CheckDuplicateWithTxPool(Tx, TxPool); err != nil {
		return err
	}

	//check by payload.
	switch Tx.TxType {
	case tx.IssueAsset:
		results := Tx.GetMergedAssetIDValueFromOutputs()
		for k, _ := range results {
			//Get the Asset amount when RegisterAsseted.
			trx, err := tx.TxStore.GetTransaction(k)
			if trx.TxType != tx.RegisterAsset {
				return errors.New("[VerifyTransaction], TxType is illegal.")
			}
			AssetReg := trx.Payload.(*payload.RegisterAsset)

			//Get the amount has been issued of this assetID
			var quantity_issued common.Fixed64
			if AssetReg.Amount < common.Fixed64(0) {
				continue
			} else {
				quantity_issued, err = tx.TxStore.GetQuantityIssued(k)
				if err != nil {
					return errors.New("[VerifyTransaction], GetQuantityIssued failed.")
				}
			}

			//calc the amounts in txPool which are also IssueAsset
			var txPoolAmounts common.Fixed64
			for _, t := range TxPool {
				if t.TxType == tx.IssueAsset {
					outputResult := t.GetMergedAssetIDValueFromOutputs()
					for txidInPool, txValueInPool := range outputResult {
						if txidInPool == k {
							txPoolAmounts = txPoolAmounts + txValueInPool
						}
					}
				}
			}

			//calc weather out off the amount when Registed.
			//AssetReg.Amount : amount when RegisterAsset of this assedID
			//quantity_issued : amount has been issued of this assedID
			//txPoolAmounts   : amount in transactionPool of this assedID of issue transaction.
			if AssetReg.Amount-quantity_issued < txPoolAmounts {
				return errors.New("[VerifyTransaction], Amount check error.")
			}
		}
	}
	return nil
}

// VerifyTransactionWithLedger verifys a transaction with history transaction in ledger
func VerifyTransactionWithLedger(Tx *tx.Transaction, ledger *ledger.Ledger) error {
	if IsDoubleSpend(Tx, ledger) {
		return errors.New("[IsDoubleSpend] faild.")
	}
	return nil
}

//validate the transaction of duplicate UTXO input
func CheckDuplicateInput(tx *tx.Transaction) error {
	if len(tx.UTXOInputs) == 0 {
		return nil
	}
	for i, utxoin := range tx.UTXOInputs {
		for j := 0; j < i; j++ {
			if utxoin.ReferTxID == tx.UTXOInputs[j].ReferTxID && utxoin.ReferTxOutputIndex == tx.UTXOInputs[j].ReferTxOutputIndex {
				return errors.New("invalid transaction")
			}
		}
	}
	return nil
}

func CheckDuplicateWithTxPool(tx *tx.Transaction, txPool []*tx.Transaction) error {
	// TODO: Optimize performance with incremental checking and deal with the duplicated tx
	//1.check duplicate transaction hash.
	//2.check utxo double spend.
	var txInputs, txPoolInputs []string
	for _, t := range tx.UTXOInputs {
		txInputs = append(txInputs, t.ToString())
	}
	count :=0
	for _, t := range txPool {
		if t.Hash() == tx.Hash(){
			count++
		}else{
			for _, u := range t.UTXOInputs {
				txPoolInputs = append(txPoolInputs, u.ToString())
			}
		}
	}
	if count >1{
		return errors.New("Duplicated transaction hash found in tx pool")
	}
	for _, i := range txInputs {
		for _, j := range txPoolInputs {
			if i == j {
				return errors.New("Duplicated UTXO inputs found in tx pool")
			}
		}
	}
	return nil
}

func IsDoubleSpend(tx *tx.Transaction, ledger *ledger.Ledger) bool {
	return ledger.IsDoubleSpend(tx)
}

func CheckAssetPrecision(Tx *tx.Transaction) error {
	assetOutputs :=     make(map[common.Uint256][]*tx.TxOutput,len(Tx.Outputs))

	for _, v := range Tx.Outputs {
		assetOutputs[v.AssetID] = append(assetOutputs[v.AssetID],v)
	}
	for k, outputs := range assetOutputs {
		asset, err := ledger.DefaultLedger.GetAsset(k)
		if err != nil {
			return errors.New("The asset not exist in local blockchain.")
		}
		precision := asset.Precision
		for _, output := range outputs {
			if output.Value.GetData()%int64(math.Pow(10, 8-float64(precision))) != 0 {
				return errors.New("The precision of asset is incorrect.")
			}
		}
	}
	return nil
}

func CheckTransactionBalance(Tx *tx.Transaction) error {
	results, err := Tx.GetTransactionResults()
	if err != nil {
		return err
	}
	for k, v := range results {
		if v != 0 {
			log.Debug(fmt.Sprintf("AssetID %x in Transfer transactions %x , Input/output UTXO not equal.", k, Tx.Hash()))
			return errors.New(fmt.Sprintf("AssetID %x in Transfer transactions %x , Input/output UTXO not equal.", k, Tx.Hash()))
		}
	}
	return nil
}

func CheckAttributeProgram(Tx *tx.Transaction) error {
	//TODO: implement CheckAttributeProgram
	return nil
}

func CheckTransactionContracts(Tx *tx.Transaction) error {
	flag, err := VerifySignableData(Tx)
	if flag && err == nil {
		return nil
	} else {
		return err
	}
}
