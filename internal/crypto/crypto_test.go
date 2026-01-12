package crypto_test

import (
	"errors"
	"fmt"
	"io"
	"os"
	"testing"

	mp "github.com/agiledragon/gomonkey/v2"
	"github.com/xhd2015/xgo/runtime/mock"
	"golang.org/x/term"

	. "github.com/julianstephens/warden/internal/crypto"
	"github.com/julianstephens/warden/internal/warden"
)

func assertSliceEqual[T comparable](t *testing.T, expected []T, actual []T) {
	t.Helper()
	if len(expected) != len(actual) {
		t.Errorf("expected (%+v) is not equal to actual (%+v): len(expected)=%d len(actual)=%d",
			expected, actual, len(expected), len(actual))
	}
	for i := range expected {
		if expected[i] != actual[i] {
			t.Errorf("expected[%d] (%+v) is not equal to actual[%d] (%+v)",
				i, expected[i],
				i, actual[i])
		}
	}
}

func TestNewRandom(t *testing.T) {
	cases := []struct {
		name     string
		input    int
		expected int
		gotError error
	}{
		{
			name:     "should return byte slice of size 10",
			input:    10,
			expected: 10,
			gotError: nil,
		},
		{
			name:     "should error on input equals zero",
			input:    0,
			expected: 0,
			gotError: ErrInvalidRandomSize,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res, err := NewRandom(c.input)

			if c.gotError != nil {
				if !errors.Is(err, c.gotError) {
					t.Fatalf("expected an error: %+v, but got: %+v", c.gotError, err)
					return
				}
			}

			if len(res) != c.expected {
				t.Fatalf("expected byte array of len: %d, but got: %d", c.expected, len(res))
			}
		})
	}
}

func TestNewIDKey(t *testing.T) {
	salt := NewSalt()
	r, err := NewRandom(24)
	if err != nil {
		t.Fatal(err)
	}
	securePassword := string(r)

	cases := []struct {
		name     string
		input    string
		expected int
		gotError any
	}{
		{
			name:     "should return new encryption key and aead cipher",
			input:    securePassword,
			expected: 32,
			gotError: nil,
		},
		{
			name:     "should return error when password is insecure",
			input:    "blah",
			expected: 0,
			gotError: warden.InvalidPasswordError{Msg: "insecure password, try including more special characters, using uppercase letters, using numbers or using a longer password"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			key, err := NewIDKey(DefaultParams, c.input, salt)

			if c.gotError != nil {
				if !errors.As(err, &c.gotError) {
					t.Fatalf("expected error: %+v -> got: %+v", c.gotError, err)
				}
				return
			}

			if len(key.Data) != DefaultParams.L {
				t.Fatalf("expected key len %d -> got %d", DefaultParams.L, len(key.Data))
			}
		})
	}

	_, err = NewIDKey(DefaultParams, securePassword, make([]byte, 1))
	if !errors.Is(err, ErrInvalidSalt) {
		t.Fatalf("expected an error: %+v, got: %+v", ErrInvalidSalt, nil)
	}
}

func TestEncryptDecrypt(t *testing.T) {
	key, err := NewSessionKey()
	if err != nil {
		t.Errorf("failed to create new encryption key: %+v", err)
	}

	text := "Hello, world!"
	enc, err := Encrypt(*key, []byte(text), nil)
	if err != nil {
		t.Errorf("failed to encrypt text: %+v", err)
	}

	dec, err := Decrypt(*key, enc, nil)
	if err != nil {
		t.Errorf("failed to decrypt text: %+v", err)
	}

	fmt.Println(len(string(enc)))
	fmt.Println(len(string(dec)))

	assertSliceEqual(t, []byte(text), dec)
}

func TestReadPassword(t *testing.T) {
	r, err := NewRandom(24)
	if err != nil {
		t.Fatal(err)
	}
	securePassword := string(r)

	cases := []struct {
		name         string
		input        string
		confirmation string
		expected     string
		gotError     any
	}{
		{
			name:         "should return password string",
			input:        securePassword,
			confirmation: securePassword,
			expected:     securePassword,
			gotError:     nil,
		},
		{
			name:         "should return error when password is empty",
			input:        "",
			confirmation: "blah",
			expected:     "",
			gotError:     warden.InvalidPasswordError{Msg: "password cannot be empty"},
		},
		{
			name:         "should return error when passwords don't match",
			input:        securePassword,
			confirmation: "blah",
			expected:     "",
			gotError:     warden.InvalidPasswordError{Msg: "passwords do not match"},
		},
	}

	tt := &term.Terminal{}
	tmpFile, err := os.CreateTemp("", "test")
	if err != nil {
		t.Fatal(err)
	}

	mock.Patch(term.MakeRaw, func(fd int) (*term.State, error) {
		return &term.State{}, nil
	})
	mock.Patch(term.Restore, func(fd int, state *term.State) error {
		return nil
	})
	mock.Patch(term.NewTerminal, func(c io.ReadWriter, prompt string) *term.Terminal {
		return &term.Terminal{}
	})

	oldStdin := os.Stdin

	defer func() {
		tmpFile.Close()

		os.Stdin = oldStdin

	}()

	os.Stdin = tmpFile

	for _, c := range cases {

		t.Run(c.name, func(t *testing.T) {
			pwdPatch := mp.ApplyMethodSeq(tt, "ReadPassword", []mp.OutputCell{{Values: mp.Params{c.input, nil}}, {Values: mp.Params{c.confirmation, nil}}})

			res, err := ReadPassword()
			if c.gotError != nil {
				if !errors.As(err, &c.gotError) {
					t.Fatalf("expected an error: %+v, got: %+v", c.gotError, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}

			if res != c.expected {
				t.Fatalf("expected %s -> got %s", c.expected, res)
			}

			pwdPatch.Reset()
		})
	}

	err = os.Remove(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}
}
