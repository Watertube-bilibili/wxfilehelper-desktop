// SPDX-License-Identifier: AGPL-3.0-only
package localsendui

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

//go:embed official-files.json
var index []byte

func preference() string {
	d, e := os.UserConfigDir()
	if e != nil {
		return ""
	}
	return filepath.Join(d, "UltraFile", "localsend-component")
}
func componentCandidates() []string {
	var dirs []string
	if exe, e := os.Executable(); e == nil {
		dirs = append(dirs, filepath.Join(filepath.Dir(exe), "components", "LocalSend", Version))
	}
	if d, e := os.UserCacheDir(); e == nil {
		dirs = append(dirs, filepath.Join(d, "Programs", "WXFileHelperNative", "components", "LocalSend", Version))
	}
	if b, e := os.ReadFile(preference()); e == nil && len(b) < 32768 {
		dirs = append(dirs, string(b))
	}
	return dirs
}
func verifiedFolder(ctx context.Context, dir string) (string, error) {
	var files map[string]string
	if e := json.Unmarshal(index, &files); e != nil {
		return "", e
	}
	for relative, want := range files {
		if e := ctx.Err(); e != nil {
			return "", e
		}
		path := filepath.Join(dir, filepath.FromSlash(relative))
		f, e := os.Open(path)
		if e != nil {
			return "", e
		}
		hash := sha256.New()
		_, e = io.Copy(hash, f)
		f.Close()
		if e != nil {
			return "", e
		}
		if hex.EncodeToString(hash.Sum(nil)) != want {
			return "", fmt.Errorf("官方组件文件校验失败 / Official file checksum mismatch: %s", relative)
		}
	}
	entries, e := os.ReadDir(dir)
	if e != nil {
		return "", e
	}
	for _, entry := range entries {
		if strings.EqualFold(filepath.Ext(entry.Name()), ".dll") {
			if _, ok := files[entry.Name()]; !ok {
				return "", errors.New("组件目录包含额外的 DLL / Unexpected library in the component directory")
			}
		}
	}
	return filepath.Join(dir, "localsend_app.exe"), nil
}
func Select(ctx context.Context, exe string) (string, error) {
	if !strings.EqualFold(filepath.Base(exe), "localsend_app.exe") {
		return "", errors.New("请选择 LocalSend 官方便携版的 localsend_app.exe / Select the official portable localsend_app.exe")
	}
	dir, e := filepath.Abs(filepath.Dir(exe))
	if e != nil {
		return "", e
	}
	full, e := verifiedFolder(ctx, dir)
	if e != nil {
		return "", e
	}
	path := preference()
	if e = os.MkdirAll(filepath.Dir(path), 0700); e == nil {
		e = os.WriteFile(path, []byte(dir), 0600)
	}
	return full, e
}
