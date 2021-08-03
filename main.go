package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"strconv"
	"strings"

	sdk "github.com/polynetwork/poly-go-sdk"
	"github.com/polynetwork/poly/common"
)

var heightRoots string
var rpc string
var walletPath string
var psw string

func init() {
	flag.StringVar(&heightRoots, "heightRoots", "", "specify poly heights and roots")
	flag.StringVar(&rpc, "rpc", "", "specify poly rpc url")
	flag.StringVar(&walletPath, "wallet", "", "specify wallet path")
	flag.StringVar(&psw, "psw", "", "specify wallet password")

	flag.Parse()
}

func setUpPoly(polySdk *sdk.PolySdk, rpcAddr string) error {
	polySdk.NewRpcClient().SetAddress(rpcAddr)
	hdr, err := polySdk.GetHeaderByHeight(0)
	if err != nil {
		return err
	}
	polySdk.SetChainId(hdr.ChainID)
	return nil
}

func main() {
	polySdk := sdk.NewPolySdk()
	err := setUpPoly(polySdk, rpc)
	if err != nil {
		panic(err)
	}

	wallet, err := polySdk.OpenWallet(walletPath)
	if err != nil {
		panic(err)
	}
	signer, err := wallet.GetDefaultAccount([]byte(psw))
	if err != nil {
		panic(err)
	}

	heightRootArray := strings.Split(heightRoots, ",")

	for _, heightRoot := range heightRootArray {
		heightRootParts := strings.Split(heightRoot, ":")
		if len(heightRootParts) != 2 {
			panic(fmt.Sprintf("invalid heightRoot:%s", heightRoot))
		}
		heightInt, err := strconv.ParseUint(heightRootParts[0], 10, 64)
		if err != nil {
			panic(err)
		}
		hdr, err := polySdk.GetHeaderByHeight(uint32(heightInt))
		if err != nil {
			panic(err)
		}
		hdr.CrossStateRoot, err = common.Uint256FromHexString(heightRootParts[1])
		if err != nil {
			panic(err)
		}

		blkHash := hdr.Hash()
		sig, err := signer.Sign(blkHash[:])
		if err != nil {
			panic(err)
		}

		fmt.Printf("signature for height %d: %s\n", heightInt, hex.EncodeToString(sig))
	}

}
