package store

import (
	"context"
	"encoding/json"
	"fmt"
	"path"

	"github.com/julianstephens/warden/internal/backend"
	"github.com/julianstephens/warden/internal/backend/common"
	"github.com/julianstephens/warden/internal/crypto"
	"github.com/julianstephens/warden/internal/warden"
)

type Store struct {
	conf     warden.Config
	backend  common.Backend
	master   *Key
	session  *crypto.Key
	Location string
}

func NewStore(be common.Backend, loc string) *Store {
	return &Store{backend: be, Location: loc}
}

func OpenStore(ctx context.Context, storeLoc string) (*Store, error) {
	// TODO: limit open attempts
	warden.Log.Debug().Msg("==> store.OpenStore")

	warden.Log.Debug().Msg("initializing backend...")
	be, err := backend.NewBackend(common.LocalStorage, common.LocalStorageParams{Location: storeLoc})
	if err != nil {
		return nil, fmt.Errorf("unable to initialize localstorage backend: %+v", err)
	}
	warden.Log.Debug().Msg("localstorage backend initialized.")

	s := NewStore(be, storeLoc)
	warden.Log.Debug().Msg("store created.")

	warden.Log.Debug().Msgf("attempting to open store at %s...", storeLoc)
	err = s.open(ctx, storeLoc)
	if err != nil {
		return nil, err
	}
	warden.Log.Debug().Msg("store opened.")

	warden.Log.Debug().Msg("<== store.OpenStore")

	return s, nil
}

func (s *Store) open(ctx context.Context, storeLoc string) (err error) {
	warden.Log.Debug().Msg("reading store password...")
	password, err := crypto.ReadPassword()
	if err != nil {
		return
	}
	warden.Log.Debug().Msg("password read.")

	warden.Log.Debug().Msg("loading store config...")
	config, err := warden.LoadJSON[warden.Config](path.Join(storeLoc, "config.json"))
	if err != nil {
		return
	}
	s.conf = config
	warden.Log.Debug().Msg("config loaded.")

	params, err := warden.MapToStruct[crypto.Params](config.Params)
	if err != nil {
		return
	}

	warden.Log.Debug().Msg("loading master key...")
	master, err := LoadKey(ctx, s, storeLoc, params, password)
	if err != nil {
		return
	}
	s.master = master
	warden.Log.Debug().Msg("master key loaded.")

	return
}

func (s *Store) Init(ctx context.Context, params crypto.Params, password string) error {
	warden.Log.Debug().Msg("==> store.OpenStore")

	warden.Log.Debug().Msg("creating store config...")
	conf, err := warden.CreateConfig(params.ToMap())
	if err != nil {
		return err
	}
	s.conf = conf
	warden.Log.Debug().Msg("store config created.")

	return s.init(ctx, password, conf)
}

func (s *Store) init(ctx context.Context, password string, config warden.Config) (err error) {
	params, err := warden.MapToStruct[*crypto.Params](config.Params)
	if err != nil {
		return
	}

	warden.Log.Debug().Msg("creating new master key...")
	master, err := AddKey(ctx, s, *params, password)
	if err != nil {
		return
	}
	s.master = master
	warden.Log.Debug().Msg("master key created.")

	confJson, err := json.Marshal(&s.conf)
	if err != nil {
		return
	}

	warden.Log.Debug().Msg("saving store config...")
	err = s.backend.Save(ctx, common.Event{Name: nil, Type: common.Config}, common.NewByteReader(confJson))
	if err != nil {
		return
	}
	warden.Log.Debug().Msg("store config saved.")

	return
}

func (s *Store) Key() *Key {
	return s.master
}

func (s *Store) Config() warden.Config {
	return s.conf
}
