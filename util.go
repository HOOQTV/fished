package fished

import (
	"bytes"
	"crypto/md5"
	"encoding/gob"
)

func getBytes(data interface{}) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(data)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func getInterface(bts []byte, data interface{}) error {
	buf := bytes.NewBuffer(bts)
	dec := gob.NewDecoder(buf)
	err := dec.Decode(data)
	if err != nil {
		return err
	}
	return nil
}

func getMD5Hash(bs []byte) []byte {
	hasher := md5.New()
	hasher.Write(bs)
	return hasher.Sum(nil)
}
