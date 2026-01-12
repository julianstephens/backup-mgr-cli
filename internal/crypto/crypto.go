package crypto

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"

	pkgerr "github.com/pkg/errors"
	passwordvalidator "github.com/wagslane/go-password-validator"
	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/term"

	"github.com/julianstephens/warden/internal/warden"
)

type Key struct {
	Data []byte `json:"data"`
}

const (
	KeySize         = chacha20poly1305.KeySize
	passwordEntropy = 60
	saltSize        = 32
	nonceSize       = chacha20poly1305.NonceSizeX
)

var (
	ErrInvalidSalt       = errors.New("invalid salt")
	ErrInvalidRandomSize = errors.New("cannot generate random array of zero length")
	ErrInvalidKeyLen     = fmt.Errorf("key data must be length %d", KeySize)
)

func Hash(data []byte) warden.ID {
	return sha256.Sum256(data)
}

func SecureHash(data []byte, secret []byte) string {
	hmac := hmac.New(sha256.New, secret)
	hmac.Write(data)
	dataHmac := hmac.Sum(nil)
	return hex.EncodeToString(dataHmac)
}

// NewIDKey generates a new user key with a password
func NewIDKey(params Params, password string, salt []byte) (key *Key, err error) {
	if len(salt) != saltSize {
		err = pkgerr.Wrap(ErrInvalidSalt, fmt.Sprintf("expected len %d but got %d", saltSize, len(salt)))
		return
	}

	err = passwordvalidator.Validate(password, passwordEntropy)
	if err != nil {
		err = &warden.InvalidPasswordError{Msg: err.Error()}
		return
	}

	k := argon2.IDKey([]byte(password), salt, uint32(params.T), uint32(params.M), uint8(params.P), uint32(params.L))
	key = &Key{
		Data: k,
	}
	return
}

// NewSessionKey generates a new random file encryption key
func NewSessionKey() (key *Key, err error) {
	key = &Key{Data: make([]byte, KeySize)}
	r, err := NewRandom(KeySize)
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

	return
}

func validateSaltLen(salt []byte) {
	if len(salt) != saltSize {
		panic(pkgerr.Wrap(ErrInvalidSalt, fmt.Sprintf("expected len %d, got %d", saltSize, len(salt))))
	}
}

func ReadPassword() (string, error) {
	state, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		panic(os.Stdin.Fd())
	}
	defer term.Restore(int(os.Stdin.Fd()), state)

	terminal := term.NewTerminal(os.Stdout, "")
	pwd, err := terminal.ReadPassword("Enter password for new store: ")
	if err != nil {
		return "", &warden.InvalidPasswordError{Msg: err.Error()}
	}

	if pwd == "" {
		return "", &warden.InvalidPasswordError{Msg: "password cannot be empty"}
	}

	confPwd, err := terminal.ReadPassword("Confirm password: ")
	if err != nil {
		return "", &warden.InvalidPasswordError{Msg: err.Error()}
	}
	term.Restore(int(os.Stdin.Fd()), state)

	if pwd != confPwd {
		return "", &warden.InvalidPasswordError{Msg: "passwords do not match"}
	}

	return pwd, nil
}

func LoadKey(data []byte) (*Key, error) {
	if len(data) != KeySize {
		return nil, ErrInvalidKeyLen
	}
	return &Key{Data: data}, nil
}

func (k *Key) Equals(key *Key) bool {
	return bytes.Equal(k.Data, key.Data)
}
