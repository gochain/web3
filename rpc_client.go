package web3

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

const DefaultUserAgent = "gochain-web3/1.0"

// RPCError represents a JSON-RPC 2.0 error response, including optional revert data.
type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func (e *RPCError) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("rpc error %d: %s", e.Code, e.Message)
}

// RevertData extracts raw revert bytes from an RPCError if present (common on eth_call / eth_estimateGas).
func (e *RPCError) RevertData() []byte {
	if e == nil || e.Data == nil {
		return nil
	}
	switch v := e.Data.(type) {
	case string:
		// Geth / Reth / Anvil return "0x..." or "Reverted 0x..."
		s := strings.TrimSpace(v)
		if idx := strings.Index(s, "0x"); idx >= 0 {
			if b, err := DecodeHex(s[idx:]); err == nil {
				return b
			}
		}
	case map[string]interface{}:
		if d, ok := v["data"].(string); ok {
			if idx := strings.Index(d, "0x"); idx >= 0 {
				if b, err := DecodeHex(d[idx:]); err == nil {
					return b
				}
			}
		}
	}
	return nil
}

// BatchElem is an element in a JSON-RPC batch request.
type BatchElem struct {
	Method string
	Args   []interface{}
	Result interface{}
	Error  error
}

type jsonrpcRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      uint64        `json:"id"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

type jsonrpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      uint64          `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *RPCError       `json:"error"`
}

// RPCClient is a pure-Go JSON-RPC 2.0 client supporting HTTP(S) and WS(S).
type RPCClient struct {
	endpoint   string
	httpClient *http.Client
	headers    http.Header
	nextID     atomic.Uint64

	isWS   bool
	wsMu   sync.Mutex
	wsConn *websocket.Conn
}

// DialRPC connects to an HTTP(S) or WS(S) JSON-RPC endpoint.
func DialRPC(rawURL string) (*RPCClient, error) {
	return DialRPCContext(context.Background(), rawURL)
}

// DialRPCContext connects to an HTTP(S) or WS(S) JSON-RPC endpoint using ctx.
func DialRPCContext(ctx context.Context, rawURL string) (*RPCClient, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid rpc url %q: %w", rawURL, err)
	}
	headers := make(http.Header)
	headers.Set("User-Agent", DefaultUserAgent)

	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		headers.Set("Content-Type", "application/json")
		headers.Set("Accept", "application/json")
		return &RPCClient{
			endpoint: rawURL,
			httpClient: &http.Client{
				Timeout: 30 * time.Second,
			},
			headers: headers,
		}, nil

	case "ws", "wss":
		dialer := websocket.Dialer{
			HandshakeTimeout: 15 * time.Second,
		}
		conn, _, err := dialer.DialContext(ctx, rawURL, headers)
		if err != nil {
			return nil, err
		}
		return &RPCClient{
			endpoint: rawURL,
			headers:  headers,
			isWS:     true,
			wsConn:   conn,
		}, nil

	default:
		return nil, fmt.Errorf("unsupported rpc url scheme %q (expected http, https, ws, or wss)", u.Scheme)
	}
}

// SetHeader sets a custom HTTP header on outgoing requests.
func (c *RPCClient) SetHeader(key, value string) {
	c.headers.Set(key, value)
}

// Close closes any underlying WebSocket or idle HTTP connections.
func (c *RPCClient) Close() {
	c.wsMu.Lock()
	defer c.wsMu.Unlock()
	if c.wsConn != nil {
		_ = c.wsConn.Close()
		c.wsConn = nil
	}
	if c.httpClient != nil {
		c.httpClient.CloseIdleConnections()
	}
}

// CallContext executes a single JSON-RPC method call.
func (c *RPCClient) CallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error {
	if args == nil {
		args = []interface{}{}
	}
	id := c.nextID.Add(1)
	reqMsg := jsonrpcRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  args,
	}
	body, err := json.Marshal(reqMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal rpc request: %w", err)
	}

	rawResp, err := c.roundTrip(ctx, body)
	if err != nil {
		return err
	}

	var resp jsonrpcResponse
	if err := json.Unmarshal(rawResp, &resp); err != nil {
		return fmt.Errorf("failed to decode rpc response: %w", err)
	}
	if resp.Error != nil {
		return resp.Error
	}
	if result == nil || len(resp.Result) == 0 || bytes.Equal(resp.Result, []byte("null")) {
		return nil
	}
	return json.Unmarshal(resp.Result, result)
}

// BatchCallContext executes a batch of JSON-RPC calls in a single round-trip.
func (c *RPCClient) BatchCallContext(ctx context.Context, b []BatchElem) error {
	if len(b) == 0 {
		return nil
	}
	reqs := make([]jsonrpcRequest, len(b))
	byID := make(map[uint64]*BatchElem, len(b))
	for i := range b {
		args := b[i].Args
		if args == nil {
			args = []interface{}{}
		}
		id := c.nextID.Add(1)
		reqs[i] = jsonrpcRequest{
			JSONRPC: "2.0",
			ID:      id,
			Method:  b[i].Method,
			Params:  args,
		}
		b[i].Error = nil
		byID[id] = &b[i]
	}

	body, err := json.Marshal(reqs)
	if err != nil {
		return err
	}
	rawResp, err := c.roundTrip(ctx, body)
	if err != nil {
		return err
	}

	var resps []jsonrpcResponse
	if err := json.Unmarshal(rawResp, &resps); err != nil {
		return fmt.Errorf("failed to decode batch rpc response: %w", err)
	}
	for _, r := range resps {
		elem, ok := byID[r.ID]
		if !ok {
			continue
		}
		if r.Error != nil {
			elem.Error = r.Error
			continue
		}
		if elem.Result != nil && len(r.Result) > 0 && !bytes.Equal(r.Result, []byte("null")) {
			elem.Error = json.Unmarshal(r.Result, elem.Result)
		}
	}
	return nil
}

func (c *RPCClient) roundTrip(ctx context.Context, payload []byte) ([]byte, error) {
	if c.isWS {
		c.wsMu.Lock()
		defer c.wsMu.Unlock()
		if c.wsConn == nil {
			return nil, errors.New("websocket connection closed")
		}
		if deadline, ok := ctx.Deadline(); ok {
			_ = c.wsConn.SetWriteDeadline(deadline)
			_ = c.wsConn.SetReadDeadline(deadline)
		} else {
			_ = c.wsConn.SetWriteDeadline(time.Now().Add(30 * time.Second))
			_ = c.wsConn.SetReadDeadline(time.Now().Add(30 * time.Second))
		}
		if err := c.wsConn.WriteMessage(websocket.TextMessage, payload); err != nil {
			return nil, err
		}
		_, respBytes, err := c.wsConn.ReadMessage()
		return respBytes, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	for k, vals := range c.headers {
		for _, v := range vals {
			req.Header.Add(k, v)
		}
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Still check if body is a valid JSON-RPC error object first
		var jerr jsonrpcResponse
		if json.Unmarshal(respBytes, &jerr) == nil && jerr.Error != nil {
			return respBytes, nil
		}
		return nil, fmt.Errorf("http %d %s: %s", resp.StatusCode, resp.Status, strings.TrimSpace(string(respBytes)))
	}
	return respBytes, nil
}
