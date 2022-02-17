package w3

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/lmittmann/w3/core"
)

var (
	// ErrRequestCreation is returned by Client.CallCtx and Client.Call when the
	// creation of the RPC request failes.
	ErrRequestCreation = errors.New("w3: request creation failed")

	// ErrResponseHandling is returned by Client.CallCtx and Client.Call when
	// the handeling of the RPC response failes.
	ErrResponseHandling = errors.New("w3: response handling failed")
)

// Client represents a connection to an RPC endpoint.
type Client struct {
	client *rpc.Client
}

// NewClient returns a new Client given an rpc.Client client.
func NewClient(client *rpc.Client) *Client {
	return &Client{
		client: client,
	}
}

// Dial returns a new Client connected to the URL rawurl. An error is returned
// if the connection establishment failes.
//
// The supported URL schemes are "http", "https", "ws" and "wss". If rawurl is a
// file name with no URL scheme, a local IPC socket connection is established.
func Dial(rawurl string) (*Client, error) {
	client, err := rpc.Dial(rawurl)
	if err != nil {
		return nil, err
	}
	return &Client{
		client: client,
	}, nil
}

// MustDial is like Dial but panics if the connection establishment failes.
func MustDial(rawurl string) *Client {
	client, err := Dial(rawurl)
	if err != nil {
		panic(fmt.Sprintf("w3: %s", err))
	}
	return client
}

// Close closes the RPC connection and cancels any in-flight requests.
//
// Close implements the io.Closer interface.
func (c *Client) Close() error {
	c.client.Close()
	return nil
}

// CallCtx creates the final RPC request, sends it, and handles the RPC
// response.
//
// An error is returned if RPC request creation, networking, or RPC response
// handeling fails.
func (c *Client) CallCtx(ctx context.Context, calls ...core.Caller) error {
	// no requests = nothing to do
	if len(calls) <= 0 {
		return nil
	}

	batchElems := make([]rpc.BatchElem, len(calls))
	var err error

	// create requests
	for i, req := range calls {
		batchElems[i], err = req.CreateRequest()
		if err != nil {
			return fmt.Errorf("%w: %v", ErrRequestCreation, err)
		}
	}

	// do requests
	if len(batchElems) > 1 {
		// batch requests if >1 request
		err = c.client.BatchCallContext(ctx, batchElems)
		if err != nil {
			return err
		}
	} else {
		// non-batch requests if 1 request
		batchElem := batchElems[0]
		err = c.client.CallContext(ctx, batchElem.Result, batchElem.Method, batchElem.Args...)
		if err != nil {
			return err
		}
	}

	// handle responses
	for i, req := range calls {
		err = req.HandleResponse(batchElems[i])
		if err != nil {
			return fmt.Errorf("%w: %v", ErrResponseHandling, err)
		}
	}
	return nil
}

// Call is like CallCtx with ctx equal to context.Background().
func (c *Client) Call(calls ...core.Caller) error {
	return c.CallCtx(context.Background(), calls...)
}
