package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"

	"github.com/alecthomas/units"
	"github.com/julianstephens/warden/internal/warden"
	passwordvalidator "github.com/wagslane/go-password-validator"
	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"

	pkgerr "github.com/pkg/errors"
)

type Params struct {
	T int `json:"t"`
	M int `json:"m"`
	P int `json:"p"`
	L int `json:"T"`
}

type Key struct {
	Data []byte `json:"data"`
}

const (
	passwordEntropy = 60
	saltSize        = 32
	keySize         = chacha20poly1305.KeySize
	nonceSize       = chacha20poly1305.NonceSizeX
)

var DefaultParams = Params{
	T: 1,
	M: int(64 * units.KiB),
	P: 4,
	L: keySize,
}

var (
	ErrInvalidSalt       = errors.New("invalid salt")
	ErrInvalidPassword   = errors.New("invalid password")
	ErrInvalidRandomSize = errors.New("cannot generate random array of zero length")
)

func Hash(data []byte) warden.ID {
	return sha256.Sum256(data)
}

func NewIDKey(params Params, password string, salt []byte) (key *Key, err error) {
	if len(salt) != saltSize {
		err = pkgerr.Wrap(ErrInvalidSalt, fmt.Sprintf("expected len %d but got %d", saltSize, len(salt)))
		return
	}

	err = passwordvalidator.Validate(password, passwordEntropy)
	if err != nil {
		err = pkgerr.Wrap(ErrInvalidPassword, err.Error())
		return
	}

	k := argon2.IDKey([]byte(password), salt, uint32(params.T), uint32(params.M), uint8(params.P), uint32(params.L))
	key = &Key{
		Data: k,
	}
	return
}

// NewSessionKey generates a new random file encryption key
func NewSessionKey(salt []byte) (key *Key, err error) {
	validateSaltLen(salt)

	key = &Key{}
	r, err := NewRandom(keySize)
	if err != nil {
		return
	}
	copy(key.Data[:], r)
	return
}

// NewRandom generates a cryptographically secure random byte array
func NewRandom(size int) (random []byte, err error) {
	if size == 0 {
		err = ErrInvalidRandomSize
		return
	}

	random = make([]byte, size)
	_, err = rand.Read(random)
	if err != nil {
		return
	}

	return
}

func NewSalt() []byte {
	salt, err := NewRandom(saltSize)
	if err != nil {
		panic(pkgerr.Wrap(ErrInvalidSalt, err.Error()))
	}

	validateSaltLen(salt)

	return salt
}

// NewNonce generates a random nonce with capacity for the ciphertext
func NewNonce(ciphertextLen int, overheadLen int) (nonce []byte, err error) {
	nonce = make([]byte, nonceSize, nonceSize+ciphertextLen+overheadLen)
	_, err = rand.Read(nonce)
	if err != nil {
		return
	}

	var total byte
	for _, x := range nonce {
		total |= x
	}

	if total > 0 {
		return
	}

	err = fmt.Errorf("got invalid all-zero nonce")
	return
}

// Encrypt secures data with XChacha20-Poly1305 algo
func Encrypt(key Key, plaintext []byte, additionalData *[]byte) (encrypted []byte, err error) {
	aead, err := chacha20poly1305.NewX([]byte(key.Data[:]))
	if err != nil {
		return
	}

	nonce, err := NewNonce(len(plaintext), aead.Overhead())
	if err != nil {
		return
	}

	if additionalData == nil {
		encrypted = aead.Seal(nonce, nonce, plaintext, nil)
	} else {
		encrypted = aead.Seal(nonce, nonce, plaintext, *additionalData)
	}

	return
}

func Decrypt(key Key, encrypted []byte, additionalData *[]byte) (decrypted []byte, err error) {
	aead, err := chacha20poly1305.NewX([]byte(key.Data[:]))
	if err != nil {
		return
	}

	if len(encrypted) < aead.NonceSize() {
		err = fmt.Errorf("ciphertext is too short")
		return
	}

	nonce, ciphertext := encrypted[:aead.NonceSize()], encrypted[aead.NonceSize():]

	if additionalData == nil {
		decrypted, err = aead.Open(nil, nonce, ciphertext, nil)
	} else {
		decrypted, err = aead.Open(nil, nonce, ciphertext, *additionalData)
	}
	if err != nil {
		return
	}

	return
}

func validateSaltLen(salt []byte) {
	if len(salt) != saltSize {
		panic(pkgerr.Wrap(ErrInvalidSalt, fmt.Sprintf("expected len %d, got %d", saltSize, len(salt))))
	}
}
