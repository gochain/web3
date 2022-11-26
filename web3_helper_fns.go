package web3

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"reflect"

	"github.com/gochain/gochain/v4/common"
	"github.com/gochain/gochain/v4/common/hexutil"
)

func isValidUrl(toTest string) bool {
	u, err := url.Parse(toTest)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return false
	}
	return true
}
func downloadFile(url string) ([]byte, error) {
	var dst bytes.Buffer
	response, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	_, err = io.Copy(&dst, response.Body)
	if err != nil {
		return nil, err
	}
	return dst.Bytes(), nil
}

func convertOutputParams(params []interface{}) []interface{} {
	for i := range params {
		p := params[i]
		if h, ok := p.(common.Hash); ok {
			params[i] = h
		} else if a, okAddr := p.(common.Address); okAddr {
			params[i] = a
		} else if b, okBytes := p.(hexutil.Bytes); okBytes {
			params[i] = b
		} else if v := reflect.ValueOf(p); v.Kind() == reflect.Array {
			if t := v.Type(); t.Elem().Kind() == reflect.Uint8 {
				b := make([]byte, t.Len())
				bv := reflect.ValueOf(b)
				// Copy since we can't t.Slice() unaddressable arrays.
				for i := 0; i < t.Len(); i++ {
					bv.Index(i).Set(v.Index(i))
				}
				params[i] = hexutil.Bytes(b)
			}
		}
	}
	return params
}
