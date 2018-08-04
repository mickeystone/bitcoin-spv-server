package main

import (
	"bytes"
	"encoding/hex"
	"errors"
	"github.com/gorilla/mux"
	"github.com/gorilla/rpc"
	"github.com/gorilla/rpc/json"
	"github.com/mutalisk999/bitcoin-lib/src/bigint"
	"github.com/mutalisk999/bitcoin-lib/src/transaction"
	"github.com/mutalisk999/go-lib/src/sched/goroutine_mgr"
	"io"
	"net/http"
)

type Service struct {
}

func (s *Service) GetBlockCount(r *http.Request, args *interface{}, reply *uint32) error {
	*reply = startBlockHeight
	return nil
}

func (s *Service) GetAddressTrxs(r *http.Request, args *string, reply *[]string) error {
	trxIds, err := addressTrxDBMgr.DBGet(*args)
	if err != nil {
		return errors.New("address not found")
	}
	for trxId, _ := range trxIds {
		*reply = append(*reply, trxId)
	}
	return nil
}

func (s *Service) GetRawTrx(r *http.Request, args *string, reply *string) error {
	bytesRawTrx, err := rawTrxDBMgr.DBGet(*args)
	if err != nil {
		return errors.New("transaction id not found")
	}
	*reply = hex.EncodeToString(bytesRawTrx)
	return nil
}

func (s *Service) GetTrx(r *http.Request, args *string, reply *transaction.TrxPrintAble) error {
	bytesRawTrx, err := rawTrxDBMgr.DBGet(*args)
	if err != nil {
		return errors.New("transaction id not found")
	}
	trx := new(transaction.Transaction)
	bytesBuf := bytes.NewBuffer(bytesRawTrx)
	bytesReader := io.Reader(bytesBuf)
	err = trx.UnPack(bytesReader)
	if err != nil {
		return errors.New("unpack raw transaction fail")
	}
	trxPrintAble := trx.GetTrxPrintAble()
	*reply = trxPrintAble
	return nil
}

func (s *Service) GetUtxo(r *http.Request, args *UtxoSourcePrintAble, reply *UtxoDetailPrintAble) error {
	utxoSource := args.GetUtxoSource()
	utxoDetail, err := trxUtxoDBMgr.DBGet(utxoSource)
	if err != nil {
		return errors.New("utxo source not found")
	}
	utxoDetailPrintAble := utxoDetail.GetUtxoDetailPrintAble()
	*reply = utxoDetailPrintAble
	return nil
}

func (s *Service) ListUnSpent(r *http.Request, args *string, reply *[]UtxoDetailPrintAble) error {
	trxIds, err := addressTrxDBMgr.DBGet(*args)
	if err != nil {
		return errors.New("address not found")
	}
	for trxId, _ := range trxIds {
		bytesRawTrx, err := rawTrxDBMgr.DBGet(trxId)
		if err != nil {
			return errors.New("transaction id not found")
		}
		trx := new(transaction.Transaction)
		bytesBuf := bytes.NewBuffer(bytesRawTrx)
		bytesReader := io.Reader(bytesBuf)
		err = trx.UnPack(bytesReader)
		if err != nil {
			return errors.New("unpack raw transaction fail")
		}
		for i, _ := range trx.Vout {
			ui256 := new(bigint.Uint256)
			ui256.SetHex(trxId)
			utxoSource := new(UtxoSource)
			utxoSource.TrxId = *ui256
			utxoSource.Vout = uint32(i)
			utxoDetail, err := trxUtxoDBMgr.DBGet(*utxoSource)
			if err != nil {
				return errors.New("utxo source not found")
			}
			if utxoDetail.Status == 0 {
				utxoDetailPrintAble := utxoDetail.GetUtxoDetailPrintAble()
				*reply = append(*reply, utxoDetailPrintAble)
			}
		}
	}
	return nil
}

func rpcServer(goroutine goroutine_mgr.Goroutine, args ...interface{}) {
	defer goroutine.OnQuit()
	rpcServer := rpc.NewServer()
	rpcServer.RegisterCodec(json.NewCodec(), "application/json")
	rpcServer.RegisterCodec(json.NewCodec(), "application/json;charset=UTF-8")

	rpcService := new(Service)
	rpcServer.RegisterService(rpcService, "")

	urlRouter := mux.NewRouter()
	urlRouter.Handle("/", rpcServer)
	http.ListenAndServe(config.RpcServerConfig.RpcListenEndPoint, urlRouter)
}

func startRpcServer() uint64 {
	return goroutineMgr.GoroutineCreatePn("rpcserver", rpcServer, nil)
}
