package osr

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"

	b58 "github.com/jbenet/go-base58"
	ic "github.com/libp2p/go-libp2p-crypto"
	"github.com/multiformats/go-multicodec"
)

type Record struct {
	CID       string `json:"cid"`
	Order     uint64 `json:"ord"`
	PublicKey string `json:"pkey"`
	Salt      string `json:"salt"`
}

type signedRecord struct {
	Record    json.RawMessage `json:"rec"`
	Signature string          `json:"sig"`
}

var HeaderOSR = multicodec.Header([]byte("/ipfs/record/mildred-ordered-signed-record"))
var HeaderJSON = multicodec.Header([]byte("/json"))

var ErrInvalidSignature error = errors.New("Invalid Signature")

func Decode(rec []byte) (*Record, error) {
	if bytes.HasPrefix(rec, HeaderOSR) {
		rec = rec[len(HeaderOSR):]
	}
	if bytes.HasPrefix(rec, HeaderJSON) {
		rec = rec[len(HeaderJSON):]
	}

	var sr signedRecord
	var ur Record

	err := json.Unmarshal(rec, &sr)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(sr.Record, &ur)
	if err != nil {
		return nil, err
	}

	pkd, err := base64.RawStdEncoding.DecodeString(ur.PublicKey)
	if err != nil {
		return nil, err
	}

	pk, err := ic.UnmarshalPublicKey(pkd)
	if err != nil {
		return nil, err
	}

	sig, err := base64.RawStdEncoding.DecodeString(sr.Signature)
	if err != nil {
		return nil, err
	}

	ok, err := pk.Verify(sr.Record, sig)
	if err != nil {
		return nil, err
	} else if !ok {
		return nil, ErrInvalidSignature
	}

	return &ur, nil
}

func Path(salt string, pk ic.PubKey) (string, error) {
	data, err := pk.Hash()
	if err != nil {
		return "", err
	}
	if salt != "" {
		salt = "/" + salt
	}
	return "/osr/" + b58.Encode(data) + salt, nil
}

func (r *Record) Path() (string, error) {
	pk, err := r.GetPublicKey()
	if err != nil {
		return "", err
	}

	return Path(r.Salt, pk)
}

func (r *Record) GetPublicKey() (ic.PubKey, error) {
	pkd, err := base64.RawStdEncoding.DecodeString(r.PublicKey)
	if err != nil {
		return nil, err
	}

	return ic.UnmarshalPublicKey(pkd)
}

func (r *Record) Encode(sk ic.PrivKey) ([]byte, error) {
	pk, err := sk.GetPublic().Bytes()
	if err != nil {
		return nil, err
	}

	var ur Record = *r
	ur.PublicKey = base64.RawStdEncoding.EncodeToString(pk)

	urd, err := json.Marshal(ur)
	if err != nil {
		return nil, err
	}

	sig, err := sk.Sign(urd)
	if err != nil {
		return nil, err
	}

	sr := signedRecord{
		Record:    urd,
		Signature: base64.RawStdEncoding.EncodeToString(sig),
	}

	srd, err := json.Marshal(&sr)
	if err != nil {
		return nil, err
	}

	return append(append(HeaderOSR, HeaderJSON...), srd...), nil
}
