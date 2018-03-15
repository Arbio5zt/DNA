package common

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	DnaRpcInvalidHashErr        = "invalid hash"
	DnaRpcInvalidBlockErr       = "invalid block"
	DnaRpcInvalidTransactionErr = "invalid transaction"
	DnaRpcInvalidParameterErr   = "invalid parameter"
	DnaRpcUnknownBlockErr       = "unknown block"
	DnaRpcUnknownTransactionErr = "unknown transaction"
	DnaRpcNilErr                = "null"
	DnaRpcUnsupportedErr        = "Unsupported"
	DnaRpcInternalErrorErr      = "internal error"
	DnaRpcIOErrorErr            = "internal IO error"
	DnaRpcAPIErrorErr           = "internal API error"
)

var DNARpcError map[string]string = map[string]string{
	DnaRpcInvalidHashErr:        "",
	DnaRpcInvalidBlockErr:       "",
	DnaRpcInvalidTransactionErr: "",
	DnaRpcInvalidParameterErr:   "",
	DnaRpcUnknownBlockErr:       "",
	DnaRpcUnknownTransactionErr: "",
	DnaRpcUnsupportedErr:        "",
	DnaRpcInternalErrorErr:      "",
	DnaRpcNilErr:                "",
	DnaRpcIOErrorErr:            "",
	DnaRpcAPIErrorErr:           "",
}

type DNAJsonRpcRes struct {
	Id      interface{}     `json:"id"`
	JsonRpc string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result"`
}

func (this *DNAJsonRpcRes) HandleResult() ([]byte, error) {
	res := strings.Trim(string(this.Result), "\"")
	_, ok := DNARpcError[res]
	if ok {
		return nil, fmt.Errorf(res)
	}
	return []byte(res), nil
}

func HandleRpcResult(data []byte) ([]byte, error) {
	res := &DNAJsonRpcRes{}
	err := json.Unmarshal(data, res)
	if err != nil {
		return nil, fmt.Errorf("json.Unmarshal DNAJsonRpcRes:%s error:%s", res, err)
	}
	data, err = res.HandleResult()
	if err != nil {
		return nil, err
	}
	return data, nil
}
