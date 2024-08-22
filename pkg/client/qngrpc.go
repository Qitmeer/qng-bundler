package client

type JsonReq struct {
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      int           `json:"id"`
	Jsonrpc string        `json:"jsonrpc"`
}

func (r *RpcAdapter) Qng_getBalance(addr string, coinid int) (interface{}, error) {
	return r.client.qngWeb3("qng_getBalance", []interface{}{addr, coinid})
}
func (r *RpcAdapter) Qng_addBalance(addr string) (interface{}, error) {
	return r.client.qngWeb3("qng_addBalance", []interface{}{addr})
}

func (r *RpcAdapter) Qng_getUTXOs(addr string, limit int, locked bool) (interface{}, error) {
	return r.client.qngWeb3("qng_getUTXOs", []interface{}{addr, limit, locked})
}

func (r *RpcAdapter) Qng_sendRawTransaction(signRawTx string, allowHightFee bool) (interface{}, error) {
	return r.client.qngWeb3("qng_sendRawTransaction", []interface{}{signRawTx, allowHightFee})
}

func (r *RpcAdapter) Qng_crossSend(txid string, idx uint32, fee uint64, sig string) (interface{}, error) {
	// TODO params check
	r.client.qngCross(QngUserOp{
		Txid: txid,
		Idx:  idx,
		Fee:  fee,
		Sig:  sig,
	})
	return nil, nil
}
