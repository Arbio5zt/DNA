package node

import (
	. "DNA/cli/common"
	"DNA/common/serialization"
	"DNA/core/ledger"
	"DNA/net/httpjsonrpc"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

var exportFile = "blocks.dat"

func ExportBlocks(path string, to int) error {
	if path == "" {
		path = exportFile
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return fmt.Errorf("OpenFile %s error %s", path, err)
	}
	defer file.Close()

	height, err := getCurrBlockHeight()
	if err != nil {
		return fmt.Errorf("getCurrBlockHeight error %s", err)
	}
	height -= 1

	if to >0 && to < int(height) {
		height = uint32(to)
	}

	err = serialization.WriteUint32(file, height)
	if err != nil {
		return fmt.Errorf("serialization.WriteUint32 block heigh %d error %s", height, err)
	}
	fmt.Fprintf(os.Stdout, "Start export blocks total:%d\n", height)

	for i := uint32(0); i <= height; i++ {
		buf := bytes.NewBuffer(nil)
		block, err := getBlockByHeight(i)
		if err != nil {
			return fmt.Errorf("getBlockByHeight %d error %s", i, err)
		}
		err = block.Serialize(buf)
		if err != nil {
			return fmt.Errorf("block.Serialize height:%d error %s", i, err)
		}
		data, err := ZLibCompress(buf.Bytes())
		if err != nil {
			return fmt.Errorf("zlibCompress height:%d error %s", i, err)
		}
		err = serialization.WriteUint32(file, uint32(len(data)))
		if err != nil {
			return fmt.Errorf("serialization.WriteUint32 height:%d block size:%d error %s", i, len(data), err)
		}
		_, err = file.Write(data)
		if err != nil {
			return fmt.Errorf("file.Write block data height:%d error %s", i, err)
		}
		printCompletePercent(height, i)
		time.Sleep(time.Millisecond)
	}

	fmt.Fprintf(os.Stdout, "Export blocks complete\n")
	return nil
}

func printCompletePercent(total, current uint32) {
	steps := uint32(1)
	if total < 100 {
		steps = 1
	} else if total < 10000 {
		steps = 10
	} else {
		steps = 100
	}
	if steps == 1 {
		return
	}
	n := total / steps
	if n == 0 {
		return
	}
	if current%n != 0 {
		return
	}
	p := current / n
	if p == 0 {
		return
	}
	p = p * 100 / steps
	if p > 100 {
		return
	}
	fmt.Fprintf(os.Stdout, "Total:%d complete:%v%%\n", total, p)
}

func getCurrBlockHeight() (uint32, error) {
	resp, err := httpjsonrpc.Call(Address(), "getblockcount", "0", []interface{}{})
	if err != nil {
		return 0, fmt.Errorf("GetBestBlockHash error %s", err)
	}
	data, err := HandleRpcResult(resp)
	if err != nil {
		return 0, fmt.Errorf("GetBestBlockHash error %s", err)
	}
	height := uint32(0)
	err = json.Unmarshal(data, &height)
	if err != nil {
		return 0, fmt.Errorf("json.Unmarshal height:%s error:%s", data, err)
	}
	return height, nil
}

func getBlockByHeight(height uint32) (*ledger.Block, error) {
	resp, err := httpjsonrpc.Call(Address(), "getblock", "0", []interface{}{height})
	if err != nil {
		return nil, fmt.Errorf("GetBlock error %s", err)
	}
	data, err := HandleRpcResult(resp)
	if err != nil {
		return nil, fmt.Errorf("getblock error %s", err)
	}
	blockInfo := &BlockInfo2{}
	err = json.Unmarshal(data, blockInfo)
	if err != nil {
		return nil, fmt.Errorf("json.Unmarshal BlockInfo2:%s error %s", data, err)
	}
	block, err := ParseBlock(blockInfo)
	if err != nil {
		return nil, fmt.Errorf("getblock ParseBlock error %s", err)
	}
	return block, nil
}
