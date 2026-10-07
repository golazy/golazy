package lazymcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
)

func (err *rpcError) Error() string { return err.Message }

func rpcFailure(id json.RawMessage, code int, message string) rpcResponse {
	return rpcResponse{JSONRPC: "2.0", ID: id, Error: &rpcError{Code: code, Message: message}}
}

func decodeRPC(body []byte) (rpcRequest, *rpcResponse) {
	var request rpcRequest
	if !json.Valid(body) {
		failure := rpcFailure(nil, -32700, "Parse error")
		return request, &failure
	}
	if err := json.Unmarshal(body, &request); err != nil || request.JSONRPC != "2.0" || request.Method == "" || !validRPCID(request.ID) {
		failure := rpcFailure(nil, -32600, "Invalid Request")
		return request, &failure
	}
	return request, nil
}

func validRPCID(id json.RawMessage) bool {
	if len(id) == 0 {
		return true
	}
	decoder := json.NewDecoder(bytes.NewReader(id))
	decoder.UseNumber()
	var value any
	if decoder.Decode(&value) != nil {
		return false
	}
	switch value.(type) {
	case nil, string, json.Number:
		return true
	default:
		return false
	}
}

func (scope *Scope) dispatchRPC(ctx context.Context, request rpcRequest) rpcResponse {
	result, err := scope.handleRPC(ctx, request.Method, request.Params)
	if err != nil {
		var protocolError *rpcError
		if errors.As(err, &protocolError) {
			return rpcFailure(request.ID, protocolError.Code, protocolError.Message)
		}
		return rpcFailure(request.ID, -32000, err.Error())
	}
	return rpcResponse{JSONRPC: "2.0", ID: request.ID, Result: result}
}
