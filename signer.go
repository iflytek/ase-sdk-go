package ase

import (
	"crypto/hmac"
	"encoding/base64"
	"hash"
)

type Signer interface {
	Sign(data []byte) string
}

func NewSigner(secret string, alg func() hash.Hash) Signer {
	return &signer{
		secret: []byte(secret),
		alg:    alg,
	}
}

type signer struct {
	secret []byte
	alg    func() hash.Hash
}

func (s *signer) Sign(data []byte) string {
	mac := hmac.New(s.alg, s.secret)
	mac.Write(data)
	encodeData := mac.Sum(nil)
	return base64.StdEncoding.EncodeToString(encodeData)
}
