// SPDX-License-Identifier: AGPL-3.0-only
//go:build windows

// Package localsendui installs the upstream portable distribution on demand.
package localsendui

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
)

const Version = "1.19.0"
const archiveName = "LocalSend-" + Version + "-windows-x86-64.zip"
const ArchiveBytes int64 = 26092132
const digest = "1b131a225e30bb46c7b4ffee23577716b2d0605108aded27899375033ff69abc"
const archiveURL = "https://github.com/localsend/localsend/releases/download/v" + Version + "/" + archiveName

// ArchiveSource also supports a portable copy with the official ZIP alongside it.
func ArchiveSource() (path string, local bool) {
	if exe, err := os.Executable(); err == nil {
		root := filepath.Dir(exe)
		for _, dir := range []string{filepath.Join(root, "components", "LocalSend"), root} {
			path := filepath.Join(dir, archiveName)
			if f, err := os.Stat(path); err == nil && f.Mode().IsRegular() {
				return path, true
			}
		}
	}
	return archiveURL, false
}

func cacheDir() (string, error) {
	d, e := os.UserCacheDir()
	if e != nil {
		return "", e
	}
	return filepath.Join(d, "UltraFile", "LocalSend", Version), nil
}
func installed(dir string) bool {
	b, e := os.ReadFile(filepath.Join(dir, ".ultrafile-release"))
	if e != nil || string(b) != digest {
		return false
	}
	f, e := os.Stat(filepath.Join(dir, "localsend_app.exe"))
	return e == nil && f.Mode().IsRegular()
}
func Ready() bool {
	for _, dir := range componentCandidates() {
		if f, e := os.Stat(filepath.Join(dir, "localsend_app.exe")); e == nil && f.Mode().IsRegular() {
			return true
		}
	}
	dir, err := cacheDir()
	return err == nil && installed(dir)
}
func Prepare(ctx context.Context, progress func(int64)) (string, error) {
	for _, dir := range componentCandidates() {
		if _, e := os.Stat(filepath.Join(dir, "localsend_app.exe")); e == nil {
			exe, e := verifiedFolder(ctx, dir)
			if e != nil {
				return "", e
			}
			return exe, nil
		}
	}
	dir, err := cacheDir()
	if err != nil {
		return "", err
	}
	if installed(dir) {
		return verifiedFolder(ctx, dir)
	}
	parent := filepath.Dir(dir)
	if err = os.MkdirAll(parent, 0700); err != nil {
		return "", err
	}
	stage, err := os.MkdirTemp(parent, "stage-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(stage)
	zipPath := filepath.Join(stage, "archive.zip")
	var source io.ReadCloser
	path, local := ArchiveSource()
	if local {
		source, err = os.Open(path)
		if err != nil {
			return "", err
		}
	} else {
		req, e := http.NewRequestWithContext(ctx, "GET", path, nil)
		if e != nil {
			return "", e
		}
		cl := http.Client{Timeout: 8 * time.Minute}
		res, e := cl.Do(req)
		if e != nil {
			return "", e
		}
		if res.StatusCode != 200 {
			res.Body.Close()
			return "", fmt.Errorf("LocalSend download: HTTP %d", res.StatusCode)
		}
		source = res.Body
	}
	defer source.Close()
	z, e := os.Create(zipPath)
	if e != nil {
		return "", e
	}
	hash := sha256.New()
	var total int64
	buf := make([]byte, 96<<10)
	for {
		if e = ctx.Err(); e != nil {
			break
		}
		var n int
		n, e = source.Read(buf)
		if n > 0 {
			total += int64(n)
			if total > ArchiveBytes {
				e = errors.New("unexpected LocalSend archive size")
				break
			}
			if _, we := z.Write(buf[:n]); we != nil {
				e = we
				break
			}
			hash.Write(buf[:n])
			progress(total)
		}
		if e != nil {
			break
		}
	}
	ce := z.Close()
	if e != io.EOF {
		return "", e
	}
	if ce != nil {
		return "", ce
	}
	if total != ArchiveBytes || hex.EncodeToString(hash.Sum(nil)) != digest {
		return "", errors.New("LocalSend 官方包校验失败 / Package checksum mismatch")
	}
	zipfile, e := zip.OpenReader(zipPath)
	if e != nil {
		return "", e
	}
	defer zipfile.Close()
	dest := filepath.Join(stage, "app")
	if e = os.Mkdir(dest, 0700); e != nil {
		return "", e
	}
	for _, v := range zipfile.File {
		if e = ctx.Err(); e != nil {
			return "", e
		}
		name := filepath.FromSlash(v.Name)
		if !filepath.IsLocal(name) || v.Mode()&os.ModeSymlink != 0 {
			return "", errors.New("invalid package member")
		}
	}
	// Use the standard Windows ZIP extractor with literal paths. The exact
	// upstream archive has already passed the pinned SHA-256 check.
	system, err := windows.GetSystemDirectory()
	if err != nil {
		return "", err
	}
	shell := filepath.Join(system, "WindowsPowerShell", "v1.0", "powershell.exe")
	const extractCommand = "$ErrorActionPreference='Stop'; Add-Type -AssemblyName System.IO.Compression.FileSystem; [IO.Compression.ZipFile]::ExtractToDirectory('archive.zip', 'app')"
	cmd := exec.CommandContext(ctx, shell, "-NoLogo", "-NoProfile", "-NonInteractive", "-Command", extractCommand)
	cmd.Dir = stage
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("Windows 解压官方组件失败 / Windows could not extract LocalSend: %w: %.600s", err, output)
	}
	if _, e = os.Stat(filepath.Join(dest, "localsend_app.exe")); e != nil {
		return "", e
	}
	if e = os.WriteFile(filepath.Join(dest, ".ultrafile-release"), []byte(digest), 0600); e != nil {
		return "", e
	}
	if e = os.Rename(dest, dir); e != nil && !installed(dir) {
		return "", e
	}
	return verifiedFolder(ctx, dir)
}
