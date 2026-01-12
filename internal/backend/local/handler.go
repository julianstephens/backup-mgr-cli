package local

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/julianstephens/warden/internal/backend/common"
	"github.com/julianstephens/warden/internal/warden"
)

type LocalHandler struct{}

var (
	ErrInvalidByteReader = errors.New("invalid byte reader")
	ErrNoStoreLocation   = errors.New("no store location provided")
)

func write(filename string, reader *common.ByteReader) (err error) {
	warden.Log.Debug().Msgf("writing %s", filename)
	err = writeBytes(filename, reader.Reader, reader.Len)
	if err != nil {
		return
	}
	warden.Log.Debug().Msg("write successful.")
	return
}

func (h *LocalHandler) WriteConfig(ctx context.Context, reader common.IReader) error {
	bReader, ok := reader.(*common.ByteReader)
	if !ok {
		return ErrInvalidByteReader
	}

	loc := getCtxLocation(ctx, LocationCtxKey("location"))
	if loc == nil || loc == "" {
		return ErrNoStoreLocation
	}

	filePath := path.Join(loc.(string), "config.json")
	err := write(filePath, bReader)
	if err != nil {
		return err
	}

	return nil
}

func (h *LocalHandler) WriteKey(ctx context.Context, filename string, reader common.IReader) error {
	bReader, ok := reader.(*common.ByteReader)
	if !ok {
		return ErrInvalidByteReader
	}

	loc := getCtxLocation(ctx, LocationCtxKey("location"))
	if loc == nil {
		return ErrNoStoreLocation
	}

	err := warden.EnsureDir(path.Join(loc.(string), "keys"))
	if err != nil {
		return fmt.Errorf("unable to create key dir: %+v", err)
	}

	keyfileLoc := path.Join(loc.(string), "keys", filename)
	err = write(keyfileLoc, bReader)
	if err != nil {
		return err
	}

	return nil
}

func (h *LocalHandler) WritePack(data []byte) error { return nil }

func writeBytes(file string, reader io.Reader, readerLen int64) (err error) {
	_, err = os.Stat(file)

	var f *os.File
	var wroteBytes int64

	if os.IsNotExist(err) {
		f, err = os.Create(file)
		if err != nil {
			err = fmt.Errorf("unable to create file: %+v", err)
			return
		}
		defer f.Close()

		wroteBytes, err = io.Copy(f, reader)
		if err != nil {
			return
		}

		if wroteBytes != readerLen {
			err = fmt.Errorf("expected to write %d bytes, wrote %d", readerLen, wroteBytes)
			return
		}

		err = f.Close()
		if err != nil {
			return
		}

		err = makeReadonly(file)
		if err != nil {
			return
		}
	} else {
		err = fmt.Errorf("file conflict: %s", file)
		return
	}

	return
}
