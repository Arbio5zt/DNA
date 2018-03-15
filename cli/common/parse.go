package common

import (
	. "DNA/common"
	"DNA/core/code"
	"DNA/core/contract"
	"DNA/core/contract/program"
	"DNA/core/ledger"
	"DNA/core/transaction"
	txpl "DNA/core/transaction/payload"
	"DNA/crypto"
	. "DNA/net/httpjsonrpc"
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
)

type BlockInfo2 struct {
	Hash         string
	BlockData    *BlockHead
	Transactions []*TransactionInfo
}

type TransactionInfo struct {
	TxType            transaction.TransactionType
	PayloadVersion    byte
	Payload           json.RawMessage
	Attributes        []TxAttributeInfo
	UTXOInputs        []UTXOTxInputInfo
	BalanceInputs     []BalanceTxInputInfo
	Outputs           []TxoutputInfo
	Programs          []ProgramInfo
	AssetOutputs      []TxoutputMap
	AssetInputAmount  []AmountMap
	AssetOutputAmount []AmountMap
	Hash              string
}

func ParseBlock(blockInfo *BlockInfo2) (*ledger.Block, error) {
	txs := make([]*transaction.Transaction, len(blockInfo.Transactions))
	for i, txStr := range blockInfo.Transactions {
		tx, err := ParseTransaction(txStr)
		if err != nil {
			return nil, fmt.Errorf("ParseTransaction transactions:%s error:%s", txStr, err)
		}
		txs[i] = tx
	}

	program, err := ParseTransactionPrograms(&blockInfo.BlockData.Program)
	if err != nil {
		return nil, fmt.Errorf("ParseTransactionPrograms Program:%s error:%s", blockInfo.BlockData.Program, err)
	}
	nextBookKeeper, err := ParseUint160FromString(blockInfo.BlockData.NextBookKeeper)
	if err != nil {
		return nil, fmt.Errorf("ParseUint160FromString NextBookKeeper:%s error:%s", blockInfo.BlockData.NextBookKeeper, err)
	}
	prevBlockHash, err := ParseUint256FromString(blockInfo.BlockData.PrevBlockHash)
	if err != nil {
		return nil, fmt.Errorf("ParseUint256FromString PrevBlockHash:%s error:%s", blockInfo.BlockData.PrevBlockHash, err)
	}
	txRoot, err := ParseUint256FromString(blockInfo.BlockData.TransactionsRoot)
	if err != nil {
		return nil, fmt.Errorf("ParseUint256FromString TransactionsRoot:%s error:%s", blockInfo.BlockData.TransactionsRoot, err)
	}
	blockHead := &ledger.Blockdata{}
	blockHead.Program = program
	blockHead.NextBookKeeper = nextBookKeeper
	blockHead.Height = blockInfo.BlockData.Height
	blockHead.Timestamp = blockInfo.BlockData.Timestamp
	blockHead.Version = blockInfo.BlockData.Version
	blockHead.PrevBlockHash = prevBlockHash
	blockHead.ConsensusData = blockInfo.BlockData.ConsensusData
	blockHead.TransactionsRoot = txRoot

	return &ledger.Block{
		Blockdata:    blockHead,
		Transactions: txs,
	}, nil
}

func ParseTransaction(txStr *TransactionInfo) (*transaction.Transaction, error) {
	payload, err := ParseToPayload(txStr.TxType, []byte(txStr.Payload))
	if err != nil {
		return nil, fmt.Errorf("ParseToPayload:%s error:%s", txStr.Payload, err)
	}

	attris := make([]*transaction.TxAttribute, len(txStr.Attributes))
	for i, attr := range txStr.Attributes {
		txAttr, err := ParseTransactionAttributes(&attr)
		if err != nil {
			return nil, fmt.Errorf("ParseTransactionAttributes:%+v error:%s", attr, err)
		}
		attris[i] = txAttr
	}

	utxoInputs := make([]*transaction.UTXOTxInput, len(txStr.UTXOInputs))
	for i, input := range txStr.UTXOInputs {
		txInput, err := ParseTransactionUTXOTxInput(&input)
		if err != nil {
			return nil, fmt.Errorf("ParseTransactionUTXOTxInput:%+v error:%s", input, err)
		}
		utxoInputs[i] = txInput
	}

	balance := make([]*transaction.BalanceTxInput, len(txStr.BalanceInputs))
	for i, input := range txStr.BalanceInputs {
		txInput, err := ParseTransactionBalanceTxInputInfo(&input)
		if err != nil {
			return nil, fmt.Errorf("ParseTransactionBalanceTxInputInfo:%+v error:%s", input, err)
		}
		balance[i] = txInput
	}

	outputs := make([]*transaction.TxOutput, len(txStr.Outputs))
	for i, output := range txStr.Outputs {
		txOutput, err := ParseTransactionOutputs(&output)
		if err != nil {
			return nil, fmt.Errorf("ParseTransactionOutputs:%+v error:%s", output, err)
		}
		outputs[i] = txOutput
	}

	programs := make([]*program.Program, len(txStr.Programs))
	for i, p := range txStr.Programs {
		txProgram, err := ParseTransactionPrograms(&p)
		if err != nil {
			return nil, fmt.Errorf("ParseTransactionPrograms:%+v error:%s", p, err)
		}
		programs[i] = txProgram
	}

	assetOutputs := make(map[Uint256][]*transaction.TxOutput, len(txStr.AssetOutputs))
	for _, assetOutput := range txStr.AssetOutputs {
		outputs := make([]*transaction.TxOutput, len(assetOutput.Txout))
		for i, output := range assetOutput.Txout {
			txOutput, err := ParseTransactionOutputs(&output)
			if err != nil {
				return nil, fmt.Errorf("AssetOutputs ParseTransactionOutputs:%+v error:%s", output, err)
			}
			outputs[i] = txOutput
		}
		assetOutputs[assetOutput.Key] = outputs
	}

	assetInputAmounts := make(map[Uint256]Fixed64, len(txStr.AssetInputAmount))
	for _, assetInputAmount := range txStr.AssetInputAmount {
		assetInputAmounts[assetInputAmount.Key] = assetInputAmount.Value
	}

	assetOutputAmounts := make(map[Uint256]Fixed64, len(txStr.AssetOutputAmount))
	for _, assetOutputAmount := range txStr.AssetOutputAmount {
		assetOutputAmounts[assetOutputAmount.Key] = assetOutputAmount.Value
	}

	tx := &transaction.Transaction{}
	tx.TxType = transaction.TransactionType(txStr.TxType)
	tx.PayloadVersion = txStr.PayloadVersion
	tx.Payload = payload
	tx.AssetOutputAmount = assetOutputAmounts
	tx.AssetInputAmount = assetInputAmounts
	tx.AssetOutputs = assetOutputs
	tx.Programs = programs
	tx.Outputs = outputs
	tx.BalanceInputs = balance
	tx.Attributes = attris
	tx.UTXOInputs = utxoInputs

	txHash, err := ParseUint256FromString(txStr.Hash)
	if err != nil {
		return nil, fmt.Errorf("Hash ParseUint256FromString:%s error:%s", txStr.Hash, err)
	}
	tx.SetHash(txHash)
	return tx, nil
}

func ParseToPayload(payloadType transaction.TransactionType, data json.RawMessage) (transaction.Payload, error) {
	var payload transaction.Payload

	switch payloadType {
	case transaction.BookKeeping:
		p := &BookKeepingInfo{}
		err := json.Unmarshal(data, p)
		if err != nil {
			return nil, fmt.Errorf("json.Unmarshal payload BookKeepingInfo:%s error:%s", data, err)
		}
		bookKeeping, err := ParseBookKeeping(p)
		if err != nil {
			return nil, fmt.Errorf("ParseBookKeeping error %s", err)
		}
		payload = bookKeeping
	case transaction.BookKeeper:
		p := &BookkeeperInfo{}
		err := json.Unmarshal(data, p)
		if err != nil {
			return nil, fmt.Errorf("json.Unmarshal payload BookkeeperInfo:%s error:%s", data, err)
		}
		bookKeeper, err := ParseBookKeeper(p)
		if err != nil {
			return nil, fmt.Errorf("ParseBookKeeper error %s", err)
		}
		payload = bookKeeper
	case transaction.IssueAsset:
		p := &IssueAssetInfo{}
		err := json.Unmarshal(data, p)
		if err != nil {
			return nil, fmt.Errorf("json.Unmarshal payload IssueAsset:%s error:%s", data, err)
		}
		issueAsset, err := ParseIssueAsset(p)
		if err != nil {
			return nil, fmt.Errorf("ParseIssueAsset error %s", err)
		}
		payload = issueAsset
	case transaction.RegisterAsset:
		p := &RegisterAssetInfo{}
		err := json.Unmarshal(data, p)
		if err != nil {
			return nil, fmt.Errorf("json.Unmarshal payload RegisterAssetInfo:%s error:%s", data, err)
		}
		regAsset, err := ParseRegisterAssetInfo(p)
		if err != nil {
			return nil, fmt.Errorf("ParsePayloadRegisterAssetInfo error:%s", err)
		}
		payload = regAsset
	case transaction.TransferAsset:
	case transaction.Record:
		p := &RecordInfo{}
		err := json.Unmarshal(data, p)
		if err != nil {
			return nil, fmt.Errorf("json.Unmarshal payload Record:%s error:%s", data, err)
		}
		record, err := ParseRecord(p)
		if err != nil {
			return nil, fmt.Errorf("ParsePayloadRecord error:%s", err)
		}
		payload = record
	case transaction.DeployCode:
		p := &DeployCodeInfo{}
		err := json.Unmarshal(data, p)
		if err != nil {
			return nil, fmt.Errorf("json.Unmarshal payload DeployCodeInfo:%s error:%s", data, err)
		}
		deplyCode, err := ParseDeployCodeInfo(p)
		if err != nil {
			return nil, fmt.Errorf("ParsePayloadDeployCodeInfo error:%s", err)
		}
		payload = deplyCode
	case transaction.PrivacyPayload:
		p := &PrivacyPayloadInfo{}
		err := json.Unmarshal(data, p)
		if err != nil {
			return nil, fmt.Errorf("json.Unmarshal payload PrivacyPayloadInfo:%s error:%s", data, err)
		}
		privacy, err := ParsePrivacyPayloadInfo(p)
		if err != nil {
			return nil, fmt.Errorf("ParsePrivacyPayloadInfo error:%s", err)
		}
		payload = privacy
	case transaction.DataFile:
		p := &DataFileInfo{}
		err := json.Unmarshal(data, p)
		if err != nil {
			return nil, fmt.Errorf("json.Unmarshal payload DataFileInfo:%s error:%s", data, err)
		}
		dataFile, err := ParseDataFile(p)
		if err != nil {
			return nil, fmt.Errorf("ParseDataFile error:%s", err)
		}
		payload = dataFile
	case transaction.StateUpdate:
		p := &StateUpdateInfo{}
		err := json.Unmarshal(data, p)
		if err != nil {
			return nil, fmt.Errorf("json.Unmarshal payload StateUpdateInfo:%s error:%s", data, err)
		}
		dataFile, err := ParseStateUpdate(p)
		if err != nil {
			return nil, fmt.Errorf("ParseStateUpdate error:%s", err)
		}
		payload = dataFile
	case transaction.DestroyUTXO:
	default:
		return nil, fmt.Errorf("unknow transaction payload %s", payloadType)
	}

	return payload, nil
}

func ParseTransactionAttributes(attr *TxAttributeInfo) (*transaction.TxAttribute, error) {
	data, err := hex.DecodeString(attr.Data)
	if err != nil {
		return nil, fmt.Errorf("hex.DecodeString TxAttributeInfo.Data:%s error:%s", attr.Data, err)
	}
	txAttr := &transaction.TxAttribute{}
	txAttr.Usage = transaction.TransactionAttributeUsage(attr.Usage)
	txAttr.Data = data
	return txAttr, nil
}

func ParseTransactionUTXOTxInput(input *UTXOTxInputInfo) (*transaction.UTXOTxInput, error) {
	txId, err := ParseUint256FromString(input.ReferTxID)
	if err != nil {
		return nil, fmt.Errorf("ParseUint256FromString UTXOTxInputInfo.ReferTxID:%s error:%s", input.ReferTxID, err)
	}
	return &transaction.UTXOTxInput{
		ReferTxID:          txId,
		ReferTxOutputIndex: input.ReferTxOutputIndex,
	}, nil
}

func ParseTransactionBalanceTxInputInfo(input *BalanceTxInputInfo) (*transaction.BalanceTxInput, error) {
	assetId, err := ParseUint256FromString(input.AssetID)
	if err != nil {
		return nil, fmt.Errorf("ParseUint256FromString BalanceTxInputInfo.AssetID:%s error:%s", input.AssetID, err)
	}
	programHash, err := ParseUint160FromString(input.ProgramHash)
	if err != nil {
		return nil, fmt.Errorf("ParseUint160FromString BalanceTxInputInfo.ProgramHash:%s error:%s", input.ProgramHash, err)
	}
	return &transaction.BalanceTxInput{
		AssetID:     assetId,
		Value:       input.Value,
		ProgramHash: programHash,
	}, nil
}

func ParseTransactionOutputs(output *TxoutputInfo) (*transaction.TxOutput, error) {
	assetId, err := ParseUint256FromString(output.AssetID)
	if err != nil {
		return nil, fmt.Errorf("ParseUint256FromString TxOutput.AssetID:%s error:%s", output.AssetID, err)
	}
	programHash, err := ParseUint160FromString(output.ProgramHash)
	if err != nil {
		return nil, fmt.Errorf("ParseUint160FromString TxOutput.ProgramHash:%s error:%s", output.ProgramHash, err)
	}
	return &transaction.TxOutput{
		AssetID:     assetId,
		Value:       output.Value,
		ProgramHash: programHash,
	}, nil
}

func ParseTransactionPrograms(p *ProgramInfo) (*program.Program, error) {
	code, err := hex.DecodeString(p.Code)
	if err != nil {
		return nil, fmt.Errorf("hex.DecodeString Code:%s error:%s", p.Code, err)
	}
	param, err := hex.DecodeString(p.Parameter)
	if err != nil {
		return nil, fmt.Errorf("hex.DecodeString Parameter:%s error:%s", p.Parameter, err)
	}
	return &program.Program{
		Code:      code,
		Parameter: param,
	}, nil
}

func ParseBookKeeping(p *BookKeepingInfo) (*txpl.BookKeeping, error) {
	bookKeeping := &txpl.BookKeeping{}
	bookKeeping.Nonce = p.Nonce
	return bookKeeping, nil
}

func ParseBookKeeper(p *BookkeeperInfo) (*txpl.BookKeeper, error) {
	bookKeeper := &txpl.BookKeeper{}
	switch p.Action {
	case "add":
		bookKeeper.Action = txpl.BookKeeperAction_ADD
	case "sub":
		bookKeeper.Action = txpl.BookKeeperAction_SUB
	default:
	}
	pkData, err := hex.DecodeString(p.PubKey)
	if err != nil {
		return nil, fmt.Errorf("hex.DecodeString PubKey:%s error:%s", p.PubKey, err)
	}
	pk, err := crypto.DecodePoint(pkData)
	if err != nil {
		return nil, fmt.Errorf("DecodePoint error %s", err)
	}
	bookKeeper.PubKey = pk
	return bookKeeper, nil
}

func ParseIssueAsset(issue *IssueAssetInfo) (*txpl.IssueAsset, error) {
	issueAsset := &txpl.IssueAsset{
		Nonce: issue.Nonce,
	}
	return issueAsset, nil
}

func ParseRegisterAssetInfo(p *RegisterAssetInfo) (*txpl.RegisterAsset, error) {
	regAsset := &txpl.RegisterAsset{}
	regAsset.Asset = p.Asset
	regAsset.Amount = p.Amount

	controler, err := ParseUint160FromString(p.Controller)
	if err != nil {
		return nil, fmt.Errorf("Controller:%s ParseUint160FromString error:%s", p.Controller, err)
	}
	regAsset.Controller = controler

	x := &big.Int{}
	_, err = fmt.Sscan(p.Issuer.X, x)
	if err != nil {
		return nil, fmt.Errorf("fmt.Sscan Issuer.X:%s error:%s", p.Issuer.X, err)
	}
	y := &big.Int{}
	_, err = fmt.Sscan(p.Issuer.Y, y)
	if err != nil {
		return nil, fmt.Errorf("fmt.Sscan Issuer.Y:%s error:%s", p.Issuer.Y, err)
	}

	issuer := &crypto.PubKey{
		X: x,
		Y: y,
	}
	regAsset.Issuer = issuer
	return regAsset, nil
}

func ParsePrivacyPayloadInfo(p *PrivacyPayloadInfo) (*txpl.PrivacyPayload, error) {
	privacy := &txpl.PrivacyPayload{}
	privacy.PayloadType = txpl.EncryptedPayloadType(p.PayloadType)
	privacy.EncryptType = txpl.PayloadEncryptType(p.EncryptType)
	data, err := hex.DecodeString(p.Payload)
	if err != nil {
		return nil, fmt.Errorf("hex.DecodeString Payload error %s", err)
	}
	privacy.Payload = data
	data, err = hex.DecodeString(p.EncryptAttr)
	if err != nil {
		return nil, fmt.Errorf("hex.DecodeStrin EncryptAttr error %s", err)
	}
	if privacy.EncryptType != txpl.ECDH_AES256 {
		return nil, fmt.Errorf("unknow EncryptType:%v", privacy.EncryptType)
	}
	ecdhaes := &txpl.EcdhAes256{}
	reader := bytes.NewBuffer(data)
	err = ecdhaes.Deserialize(reader)
	if err != nil {
		return nil, fmt.Errorf("ecdhaes.Deserialize error %s", err)
	}
	privacy.EncryptAttr = ecdhaes
	return privacy, nil
}

func ParseDataFile(p *DataFileInfo) (*txpl.DataFile, error) {
	dataFile := &txpl.DataFile{}
	dataFile.Filename = p.Filename
	dataFile.IPFSPath = p.IPFSPath
	dataFile.Note = p.Note
	x := &big.Int{}
	_, err := fmt.Sscan(p.Issuer.X, x)
	if err != nil {
		return nil, fmt.Errorf("fmt.Sscan Issuer.X:%s error:%s", p.Issuer.X, err)
	}
	y := &big.Int{}
	_, err = fmt.Sscan(p.Issuer.Y, y)
	if err != nil {
		return nil, fmt.Errorf("fmt.Sscan Issuer.Y:%s error:%s", p.Issuer.Y, err)
	}
	issuer := &crypto.PubKey{
		X: x,
		Y: y,
	}
	dataFile.Issuer = issuer
	return dataFile, nil
}

func ParseStateUpdate(p *StateUpdateInfo) (*txpl.StateUpdate, error) {
	stateUpdate := &txpl.StateUpdate{}
	ns, err := hex.DecodeString(p.Namespace)
	if err != nil {
		return nil, fmt.Errorf(" hex.DecodeString Namespace error:%s", err)
	}
	key, err := hex.DecodeString(p.Key)
	if err != nil {
		return nil, fmt.Errorf("hex.DecodeString Key error %s", err)
	}
	value, err := hex.DecodeString(p.Value)
	if err != nil {
		return nil, fmt.Errorf("hex.DecodeString Value error %s", err)
	}
	x := &big.Int{}
	_, err = fmt.Sscan(p.Updater.X, x)
	if err != nil {
		return nil, fmt.Errorf("fmt.Sscan Issuer.X:%s error:%s", p.Updater.X, err)
	}
	y := &big.Int{}
	_, err = fmt.Sscan(p.Updater.Y, y)
	if err != nil {
		return nil, fmt.Errorf("fmt.Sscan Issuer.Y:%s error:%s", p.Updater.Y, err)
	}
	updater := &crypto.PubKey{
		X: x,
		Y: y,
	}
	stateUpdate.Updater = updater
	stateUpdate.Value = value
	stateUpdate.Key = key
	stateUpdate.Namespace = ns
	return stateUpdate, nil
}

func ParseRecord(p *RecordInfo) (*txpl.Record, error) {
	record := &txpl.Record{}
	record.RecordType = p.RecordType
	data, err := hex.DecodeString(p.RecordData)
	if err != nil {
		return nil, fmt.Errorf("hex.DecodeString RecordData:%s error:%s", p.RecordData, err)
	}

	record.RecordData = data
	return record, nil
}

func ParseDeployCodeInfo(p *DeployCodeInfo) (*txpl.DeployCode, error) {
	c, err := hex.DecodeString(p.Code.Code)
	if err != nil {
		return nil, fmt.Errorf("hex.DecodeString Code:%s error:%s", p.Code.Code, err)
	}
	paramByte, err := hex.DecodeString(p.Code.ParameterTypes)
	if err != nil {
		return nil, fmt.Errorf("hex.DecodeString ParameterTypes:%s error:%s", p.Code.ParameterTypes, err)
	}
	param := contract.ByteToContractParameterType(paramByte)
	retByte, err := hex.DecodeString(p.Code.ReturnTypes)
	if err != nil {
		return nil, fmt.Errorf("hex.DecodeString ReturnTypes:%s error:%s", p.Code.ReturnTypes, err)
	}
	ret := contract.ByteToContractParameterType(retByte)

	deplyCode := &txpl.DeployCode{}
	deplyCode.Code = &code.FunctionCode{
		Code:           c,
		ParameterTypes: param,
		ReturnTypes:    ret,
	}
	deplyCode.Name = p.Name
	deplyCode.Author = p.Author
	deplyCode.CodeVersion = p.CodeVersion
	deplyCode.Description = p.Description
	deplyCode.Email = p.Email
	return deplyCode, nil
}

func ParseUint160FromString(value string) (Uint160, error) {
	data, err := hex.DecodeString(value)
	if err != nil {
		return Uint160{}, fmt.Errorf("hex.DecodeString error:%s", err)
	}
	res, err := Uint160ParseFromBytes(data)
	if err != nil {
		return Uint160{}, fmt.Errorf("Uint160ParseFromBytes error:%s", err)
	}
	return res, nil
}

func ParseUint256FromString(value string) (Uint256, error) {
	data, err := hex.DecodeString(value)
	if err != nil {
		return Uint256{}, fmt.Errorf("hex.DecodeString error:%s", err)
	}
	res, err := Uint256ParseFromBytes(data)
	if err != nil {
		return Uint256{}, fmt.Errorf("Uint160ParseFromBytes error:%s", err)
	}
	return res, nil
}

