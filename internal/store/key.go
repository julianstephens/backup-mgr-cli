package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/julianstephens/warden/internal/backend/common"
	"github.com/julianstephens/warden/internal/crypto"
	"github.com/julianstephens/warden/internal/warden"
)

type Key struct {
	id warden.ID

	master *crypto.Key
	user   *crypto.Key

	Username  string        `json:"username"`
	Hostname  string        `json:"hostname"`
	CreatedAt time.Time     `json:"createdAt"`
	Params    crypto.Params `json:"params"`
	Salt      []byte        `json:"salt"`
	Data      []byte        `json:"data"`
}

func (k *Key) ID() warden.ID {
	return k.id
}

func (k *Key) Valid() bool {
	// TODO: valdiate key
	return true
}

func (k *Key) String() string {
	template := func(data string) string { return fmt.Sprintf("<Key | %s>", data) }

	if k == nil {
		return template("nil")
	}

	return template(fmt.Sprintf("user: %s, host: %s, created: %s", k.Username, k.Hostname, k.CreatedAt))
}

func (k *Key) Decrypt() *crypto.Key {
	return k.master
}

// LoadKey decrypts the store master key with a password
func LoadKey(ctx context.Context, store *Store, storeLoc string, params crypto.Params, password string) (*Key, error) {
	key, err := findKey(path.Join(storeLoc, "keys"), password)
	if err != nil {
		return nil, err
	}

	keyJson, err := json.Marshal(key)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal key to json: %+v", err)
	}

	id := crypto.Hash(keyJson)
	key.id = id

	return key, nil
}

// AddKey creates a new master key and saves it
func AddKey(ctx context.Context, store *Store, params crypto.Params, password string) (*Key, error) {
	salt := crypto.NewSalt()
	warden.Log.Debug().Msg("generated store salt.")

	warden.Log.Debug().Msg("deriving master key from password, params, and salt...")
	k, err := deriveKey(params, password, salt)
	if err != nil {
		return nil, err
	}
	warden.Log.Debug().Msg("master key created.")

	warden.Log.Debug().Msg("generating keyfile...")
	keyJson, err := json.Marshal(k)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal key to json: %+v", err)
	}
	warden.Log.Debug().Msg("keyfile created.")

	id := crypto.Hash(keyJson)
	k.id = id
	name := id.String()

	event := common.Event{
		Type: common.Key,
		Name: &name,
	}

	err = store.backend.Save(ctx, event, common.NewByteReader(keyJson))
	if err != nil {
		return nil, err
	}

	return k, nil
}

func findKey(keyDir string, password string) (*Key, error) {
	keys, err := os.ReadDir(keyDir)
	if err != nil {
		return nil, err
	}

	var derivedKey *Key
	found := false

	for _, k := range keys {
		if k.IsDir() {
			return nil, errors.New("malformed store: invalid keys dir")
		}

		if !strings.HasSuffix(k.Name(), "json") {
			return nil, errors.New("malformed key: invalid key extension")
		}

		loadedKey, err := warden.LoadJSON[Key](path.Join(keyDir, k.Name()))
		if err != nil {
			return nil, fmt.Errorf("unable to load key (%s): %+v", k.Name(), err)
		}

		derivedKey, err = deriveKey(loadedKey.Params, password, loadedKey.Salt)
		if err != nil {
			continue
		}

		_, err = crypto.Decrypt(*derivedKey.user, loadedKey.Data, nil)
		if err != nil {
			continue
		} else {
			found = true
		}
	}

	if !found {
		return nil, errors.New("unable to retrieve store key")
	}

	return derivedKey, nil
}

func deriveKey(params crypto.Params, password string, salt []byte) (key *Key, err error) {
	derivedUser, err := crypto.NewIDKey(params, password, salt)
	if err != nil {
		return
	}

	master, err := crypto.NewSessionKey()
	if err != nil {
		return
	}

	masterJson, err := json.Marshal(master)
	if err != nil {
		return
	}

	encMaster, err := crypto.Encrypt(*derivedUser, masterJson, nil)
	if err != nil {
		return
	}

	username, hostname, err := warden.GetSystemInfo()
	if err != nil {
		return
	}

	key = &Key{
		master:    master,
		user:      derivedUser,
		Username:  username,
		Hostname:  hostname,
		CreatedAt: time.Now(),
		Params:    params,
		Salt:      salt,
		Data:      encMaster,
	}

	return
}
