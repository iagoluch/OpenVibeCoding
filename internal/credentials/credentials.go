// Package credentials uses OS credential storage by default. File fallback is
// an explicit user choice; failure to unlock a keychain never writes plaintext.
package credentials

import (
	"encoding/json"
	"errors"
	"github.com/zalando/go-keyring"
	"os"
	"path/filepath"
	"runtime"
)

const service = "openvibecoding"

func Get(dir, name string) string {
	if s, e := keyring.Get(service, name); e == nil {
		return s
	}
	path := filepath.Join(dir, "credentials.json")
	info, e := os.Stat(path)
	if e != nil || (runtime.GOOS != "windows" && info.Mode().Perm()&0077 != 0) {
		return ""
	}
	b, e := os.ReadFile(path)
	if e != nil {
		return ""
	}
	var m map[string]string
	if json.Unmarshal(b, &m) != nil {
		return ""
	}
	return m[name]
}
func Set(dir, name, value string, file bool) error {
	if !file {
		if e := keyring.Set(service, name, value); e != nil {
			return errors.New("OS credential store unavailable; unlock it or explicitly select --credential-file")
		}
		return nil
	}
	if e := os.MkdirAll(dir, 0700); e != nil {
		return e
	}
	path := filepath.Join(dir, "credentials.json")
	m := map[string]string{}
	b, e := os.ReadFile(path)
	if e == nil {
		if e = json.Unmarshal(b, &m); e != nil {
			return errors.New("invalid credential file")
		}
	} else if !errors.Is(e, os.ErrNotExist) {
		return e
	}
	if value == "" {
		delete(m, name)
	} else {
		m[name] = value
	}
	b, e = json.Marshal(m)
	if e != nil {
		return e
	}
	f, e := os.CreateTemp(dir, ".credentials-*")
	if e != nil {
		return e
	}
	defer os.Remove(f.Name())
	if _, e = f.Write(b); e != nil {
		f.Close()
		return e
	}
	if e = f.Sync(); e != nil {
		f.Close()
		return e
	}
	if e = f.Close(); e != nil {
		return e
	}
	return os.Rename(f.Name(), path)
}
func Delete(dir, name string, file bool) error {
	if file {
		return Set(dir, name, "", true)
	}
	if e := keyring.Delete(service, name); e != nil {
		return errors.New("could not delete OS credential")
	}
	return nil
}
