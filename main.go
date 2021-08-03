package main

import (
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"

	"github.com/howeyc/gopass"
	sdk "github.com/polynetwork/poly-go-sdk"
	"github.com/polynetwork/poly/common"
	"github.com/polynetwork/poly/core/types"
)

var heightRoots string
var heightSigs string
var rpc string
var walletPath string

func init() {
	flag.StringVar(&heightRoots, "heightRoots", "", "specify poly heights and roots")
	flag.StringVar(&heightSigs, "heightSigs", "", "specify poly heights and sigs")
	flag.StringVar(&rpc, "rpc", "", "specify poly rpc url")
	flag.StringVar(&walletPath, "wallet", "", "specify wallet path")

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

func processHeightSigns() {
	heightSigArray := strings.Split(heightSigs, ",")

	var (
		sigMaps []map[uint64]string
		orders  []uint64
	)
	for _, f := range heightSigArray {
		sigBytes, err := ioutil.ReadFile(f)
		if err != nil {
			panic(err)
		}
		m := make(map[uint64]string)
		lines := strings.Split(string(sigBytes), "\n")
		first := len(orders) == 0
		for _, line := range lines {
			cleaned := strings.TrimSpace(line)
			parts := strings.Split(cleaned, ":")
			if len(parts) != 2 {
				panic("invalid sig format")
			}
			height, err := strconv.ParseUint(parts[0], 10, 64)
			if err != nil {
				panic(err)
			}
			m[height] = parts[1]
			if first {
				orders = append(orders, height)
			}
		}
		sigMaps = append(sigMaps, m)
	}
	combinedSigs := make(map[uint64][]string)
	var result []string
	for _, height := range orders {
		for i := 0; i < len(sigMaps); i++ {
			sig, ok := sigMaps[i][height]
			if !ok {
				panic(fmt.Sprintf("sig missing for height %d", height))
			}
			combinedSigs[height] = append(combinedSigs[height], sig)
		}
		sigsForHeight := strings.Join(combinedSigs[height], ":")
		result = append(result, fmt.Sprintf("%d:%s", height, sigsForHeight))
	}

	heightSigs = strings.Join(result, ",")
}

func main() {
	polySdk := sdk.NewPolySdk()
	err := setUpPoly(polySdk, rpc)
	if err != nil {
		panic(err)
	}

	var signer *sdk.Account
	if heightSigs == "" {
		wallet, err := polySdk.OpenWallet(walletPath)
		if err != nil {
			panic(err)
		}
		fmt.Printf("Password: ")
		psw, err := gopass.GetPasswd()
		if err != nil {
			panic(err)
		}
		signer, err = wallet.GetDefaultAccount([]byte(psw))
		if err != nil {
			panic(err)
		}
	} else {
		processHeightSigns()
	}

	heightRootArray := strings.Split(heightRoots, ",")
	heightSigArray := strings.Split(heightSigs, ",")
	if heightSigs != "" && len(heightSigArray) != len(heightRootArray) {
		panic("root and sig not match")
	}

	headers := make(map[uint32]*types.Header)

	for i, heightRoot := range heightRootArray {
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

		if heightSigs == "" {
			blkHash := hdr.Hash()
			sig, err := signer.Sign(blkHash[:])
			if err != nil {
				panic(err)
			}

			fmt.Printf("%d:%s\n", heightInt, hex.EncodeToString(sig))
		} else {
			heightSigParts := strings.Split(heightSigArray[i], ":")
			heightIntSig, err := strconv.ParseUint(heightSigParts[0], 10, 64)
			if err != nil {
				panic(err)
			}
			if heightInt != heightIntSig {
				panic("height mismatch")
			}
			hdr.SigData = nil
			for i := 1; i < len(heightSigParts); i++ {
				sig, err := hex.DecodeString(heightSigParts[i])
				if err != nil {
					panic(err)
				}
				hdr.SigData = append(hdr.SigData, sig)
			}
			headers[hdr.Height] = hdr
		}

	}

	if heightSigs != "" {
		headerJson, err := json.Marshal(headers)
		if err != nil {
			panic(err)
		}
		ioutil.WriteFile("headers.json", headerJson, 0777)
	}

}
