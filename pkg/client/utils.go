package client

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stackup-wallet/stackup-bundler/pkg/entrypoint/filter"
	"github.com/stackup-wallet/stackup-bundler/pkg/fees"
	"github.com/stackup-wallet/stackup-bundler/pkg/gas"
	"github.com/stackup-wallet/stackup-bundler/pkg/meerchange"
	"github.com/stackup-wallet/stackup-bundler/pkg/signer"
	"github.com/stackup-wallet/stackup-bundler/pkg/state"
	"github.com/stackup-wallet/stackup-bundler/pkg/userop"
)

type QngWeb3Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
type QngWeb3Result struct {
	ID      int          `json:"id"`
	JsonRpc string       `json:"jsonrpc"`
	Message string       `json:"message,omitempty"`
	Result  interface{}  `json:"result,omitempty"`
	Error   QngWeb3Error `json:"error,omitempty"`
}

// GetUserOpReceiptFunc is a general interface for fetching a UserOperationReceipt given a userOpHash,
// EntryPoint address, and block range.
type GetUserOpReceiptFunc = func(hash string, ep common.Address, blkRange uint64) (*filter.UserOperationReceipt, error)

func getUserOpReceiptNoop() GetUserOpReceiptFunc {
	return func(hash string, ep common.Address, blkRange uint64) (*filter.UserOperationReceipt, error) {
		return nil, nil
	}
}

// GetUserOpReceiptWithEthClient returns an implementation of GetUserOpReceiptFunc that relies on an eth
// client to fetch a UserOperationReceipt.
func GetUserOpReceiptWithEthClient(eth *ethclient.Client) GetUserOpReceiptFunc {
	return func(hash string, ep common.Address, blkRange uint64) (*filter.UserOperationReceipt, error) {
		return filter.GetUserOperationReceipt(eth, hash, ep, blkRange)
	}
}

// GetGasPricesFunc is a general interface for fetching values for maxFeePerGas and maxPriorityFeePerGas.
type GetGasPricesFunc = func() (*fees.GasPrices, error)

type QngWeb3Func = func(method string, params []interface{}) (interface{}, error)

type QngUserOp struct {
	Txid string `json:"txid"`
	Idx  uint32 `json:"idx"`
	Fee  uint64 `json:"fee"`
	Sig  string `json:"sig"`
}

type QngCrossFunc = func(QngUserOp) (string, error)

func getGasPricesNoop() GetGasPricesFunc {
	return func() (*fees.GasPrices, error) {
		return &fees.GasPrices{
			MaxFeePerGas:         big.NewInt(0),
			MaxPriorityFeePerGas: big.NewInt(0),
		}, nil
	}
}

// GetGasPricesWithEthClient returns an implementation of GetGasPricesFunc that relies on an eth client to
// fetch values for maxFeePerGas and maxPriorityFeePerGas.
func GetGasPricesWithEthClient(eth *ethclient.Client) GetGasPricesFunc {
	return func() (*fees.GasPrices, error) {
		return fees.NewGasPrices(eth)
	}
}

// GetGasEstimateFunc is a general interface for fetching an estimate for verificationGasLimit and
// callGasLimit given a userOp and EntryPoint address.
type GetGasEstimateFunc = func(
	ep common.Address,
	op *userop.UserOperation,
	sos state.OverrideSet,
) (verificationGas uint64, callGas uint64, err error)

func getGasEstimateNoop() GetGasEstimateFunc {
	return func(
		ep common.Address,
		op *userop.UserOperation,
		sos state.OverrideSet,
	) (verificationGas uint64, callGas uint64, err error) {
		return 0, 0, nil
	}
}

// GetGasEstimateWithEthClient returns an implementation of GetGasEstimateFunc that relies on an eth client to
// fetch an estimate for verificationGasLimit and callGasLimit.
func GetGasEstimateWithEthClient(
	rpc *rpc.Client,
	ov *gas.Overhead,
	chain *big.Int,
	maxGasLimit *big.Int,
	tracer string,
) GetGasEstimateFunc {
	return func(
		ep common.Address,
		op *userop.UserOperation,
		sos state.OverrideSet,
	) (verificationGas uint64, callGas uint64, err error) {
		return gas.EstimateGas(&gas.EstimateInput{
			Rpc:         rpc,
			EntryPoint:  ep,
			Op:          op,
			Sos:         sos,
			Ov:          ov,
			ChainID:     chain,
			MaxGasLimit: maxGasLimit,
			Tracer:      tracer,
		})
	}
}

// GetUserOpByHashFunc is a general interface for fetching a UserOperation given a userOpHash, EntryPoint
// address, chain ID, and block range.
type GetUserOpByHashFunc func(hash string, ep common.Address, chain *big.Int, blkRange uint64) (*filter.HashLookupResult, error)

func getUserOpByHashNoop() GetUserOpByHashFunc {
	return func(hash string, ep common.Address, chain *big.Int, blkRange uint64) (*filter.HashLookupResult, error) {
		return nil, nil
	}
}

// GetUserOpByHashWithEthClient returns an implementation of GetUserOpByHashFunc that relies on an eth client
// to fetch a UserOperation.
func GetUserOpByHashWithEthClient(eth *ethclient.Client) GetUserOpByHashFunc {
	return func(hash string, ep common.Address, chain *big.Int, blkRange uint64) (*filter.HashLookupResult, error) {
		return filter.GetUserOperationByHash(eth, hash, ep, chain, blkRange)
	}
}

func QngWeb3Request(
	rpcUrl string,
) QngWeb3Func {
	return func(
		method string,
		params []interface{},
	) (interface{}, error) {
		bodyReq := JsonReq{Jsonrpc: "2.0", Method: method, Params: params, ID: 1}
		body, err := json.Marshal(bodyReq)
		if err != nil {
			return nil, err
		}

		proxyReq, err := http.NewRequest(http.MethodPost, rpcUrl, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		client := &http.Client{}
		proxyReq.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(proxyReq)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		var data QngWeb3Result
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			return nil, err
		}
		if data.Error.Code != 0 {
			return nil, errors.New(data.Error.Message)
		}
		if data.Result == nil {
			return nil, errors.New("network request exception")
		}
		return data.Result, nil
	}
}

func QngCrossMeerChange(
	eoa *signer.EOA,
	eth *ethclient.Client,
	meerchangeAddr string,
	chainId *big.Int,
) QngCrossFunc {
	return func(
		qngOp QngUserOp,
	) (string, error) {
		meerchangeClient, err := meerchange.NewMeerchange(common.HexToAddress(meerchangeAddr), eth)
		if err != nil {
			return "", err
		}
		auth, err := bind.NewKeyedTransactorWithChainID(eoa.PrivateKey, chainId)
		if err != nil {
			return "", err
		}
		b, _ := hex.DecodeString(qngOp.Txid)
		txidBytes := common.BytesToHash(b)
		tx, err := meerchangeClient.Export4337(auth, txidBytes, qngOp.Idx, qngOp.Fee, qngOp.Sig)
		if err != nil {
			return "", err
		}
		return tx.Hash().Hex(), nil
	}
}
