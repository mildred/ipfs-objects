package secio

import (
	"errors"
	"fmt"
	"strings"

	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"hash"

	bfish "gx/ipfs/Qme1boxspcQWR8FBzMxeppqug2fYgYc15diNWmqgDVnvn2/go-crypto/blowfish"
	ci "gx/ipfs/QmfWDLQjGjVe4fr5CoztYW2DYYjRysMJrFe1RCsXLPTf46/go-libp2p-crypto"
)

// List of supported ECDH curves
var SupportedExchanges = "P-256,P-384,P-521"

// List of supported Ciphers
var SupportedCiphers = "AES-256,AES-128,Blowfish"

// List of supported Hashes
var SupportedHashes = "SHA256,SHA512"

type HMAC struct {
	hash.Hash
	size int
}

// encParams represent encryption parameters
type encParams struct {
	// keys
	permanentPubKey ci.PubKey
	ephemeralPubKey []byte
	keys            ci.StretchedKeys

	// selections
	curveT  string
	cipherT string
	hashT   string

	// cipher + mac
	cipher cipher.Stream
	mac    HMAC
}

func (e *encParams) makeMacAndCipher() error {
	m, err := newMac(e.hashT, e.keys.MacKey)
	if err != nil {
		return err
	}

	bc, err := newBlockCipher(e.cipherT, e.keys.CipherKey)
	if err != nil {
		return err
	}

	e.cipher = cipher.NewCTR(bc, e.keys.IV)
	e.mac = m
	return nil
}

func newMac(hashType string, key []byte) (HMAC, error) {
	switch hashType {
	case "SHA1":
		return HMAC{hmac.New(sha1.New, key), sha1.Size}, nil
	case "SHA512":
		return HMAC{hmac.New(sha512.New, key), sha512.Size}, nil
	case "SHA256":
		return HMAC{hmac.New(sha256.New, key), sha256.Size}, nil
	default:
		return HMAC{}, fmt.Errorf("Unrecognized hash type: %s", hashType)
	}
}

func newBlockCipher(cipherT string, key []byte) (cipher.Block, error) {
	switch cipherT {
	case "AES-128", "AES-256":
		return aes.NewCipher(key)
	case "Blowfish":
		return bfish.NewCipher(key)
	default:
		return nil, fmt.Errorf("Unrecognized cipher type: %s", cipherT)
	}
}

// Determines which algorithm to use.  Note:  f(a, b) = f(b, a)
func selectBest(order int, p1, p2 string) (string, error) {
	var f, s []string
	switch {
	case order < 0:
		f = strings.Split(p2, ",")
		s = strings.Split(p1, ",")
	case order > 0:
		f = strings.Split(p1, ",")
		s = strings.Split(p2, ",")
	default: // Exact same preferences.
		p := strings.Split(p1, ",")
		return p[0], nil
	}

	for _, fc := range f {
		for _, sc := range s {
			if fc == sc {
				return fc, nil
			}
		}
	}

	return "", errors.New("No algorithms in common!")
}
