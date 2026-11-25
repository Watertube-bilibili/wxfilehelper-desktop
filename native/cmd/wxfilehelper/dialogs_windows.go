// SPDX-License-Identifier: AGPL-3.0-only
package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

type openFilename struct {
	Size                      uint32
	Owner, Instance           uintptr
	Filter, CustomFilter      *uint16
	MaxFilter, FilterIndex    uint32
	File                      *uint16
	MaxFile                   uint32
	FileTitle                 *uint16
	MaxFileTitle              uint32
	InitialDir, Title         *uint16
	Flags                     uint32
	FileOffset, FileExtension uint16
	DefaultExt                *uint16
	CustomData, Hook          uintptr
	Template                  *uint16
	Reserved                  uintptr
	Reserved2, FlagsEx        uint32
}

func fileDialog(owner uintptr, title, filename, directory string, save bool) ([]string, error) {
	buf := make([]uint16, 65536)
	if filename != "" {
		copy(buf, syscall.StringToUTF16(filename))
	}
	// A double-NUL-terminated series of display-name/pattern pairs.
	filter := append(syscall.StringToUTF16("所有文件 / All files"), syscall.StringToUTF16("*.*")...)
	filter = append(filter, 0)
	of := openFilename{Owner: owner, Filter: &filter[0], FilterIndex: 1, File: &buf[0], MaxFile: uint32(len(buf)), Title: wide(title),
		Flags: 0x80000 | 0x800000 | 0x800 | 0x8 | 0x4} // explorer, resizable, existing directory, preserve cwd, hide read-only
	of.Size = uint32(unsafe.Sizeof(of))
	if directory != "" {
		of.InitialDir = wide(directory)
	}
	proc := comdlg32.NewProc("GetOpenFileNameW")
	if save {
		of.Flags |= 0x2
		proc = comdlg32.NewProc("GetSaveFileNameW")
	} else {
		of.Flags |= 0x200 | 0x1000
	}
	ok, _, _ := proc.Call(uintptr(unsafe.Pointer(&of)))
	if ok == 0 {
		code, _, _ := comdlg32.NewProc("CommDlgExtendedError").Call()
		if code != 0 {
			return nil, fmt.Errorf("文件选择器 / File dialog: 0x%04x", code)
		}
		return nil, nil
	}
	var paths []string
	for start, end := 0, 0; end < len(buf); end++ {
		if buf[end] != 0 {
			continue
		}
		if start == end {
			break
		}
		paths = append(paths, syscall.UTF16ToString(buf[start:end]))
		start = end + 1
	}
	if len(paths) < 2 {
		return paths, nil
	}
	for i := 1; i < len(paths); i++ {
		paths[i] = filepath.Join(paths[0], paths[i])
	}
	return paths[1:], nil
}

func droppedFiles(drop uintptr) ([]string, error) {
	query := shell32.NewProc("DragQueryFileW")
	count, _, _ := query.Call(drop, 0xffffffff, 0, 0)
	if count > 100 {
		return nil, errors.New("每次最多选择 100 个文件。 / Select up to 100 files per batch.")
	}
	paths := make([]string, 0, count)
	for i := uintptr(0); i < count; i++ {
		n, _, _ := query.Call(drop, i, 0, 0)
		if n == 0 {
			continue
		}
		buf := make([]uint16, n+1)
		if actual, _, _ := query.Call(drop, i, uintptr(unsafe.Pointer(&buf[0])), n+1); actual > 0 {
			paths = append(paths, syscall.UTF16ToString(buf))
		}
	}
	return paths, nil
}
func clipboardFiles(h uintptr) ([]string, error) {
	if available, _, _ := user32.NewProc("IsClipboardFormatAvailable").Call(15); available == 0 {
		return nil, nil
	} // CF_HDROP
	ok, _, err := user32.NewProc("OpenClipboard").Call(h)
	if ok == 0 {
		return nil, err
	}
	defer user32.NewProc("CloseClipboard").Call()
	drop, _, err := user32.NewProc("GetClipboardData").Call(15)
	if drop == 0 {
		return nil, err
	}
	return droppedFiles(drop)
}
func copyText(h uintptr, s string) error {
	chars, err := syscall.UTF16FromString(s)
	if err != nil {
		return err
	}
	ok, _, err := user32.NewProc("OpenClipboard").Call(h)
	if ok == 0 {
		return err
	}
	defer user32.NewProc("CloseClipboard").Call()
	mem, _, err := kernel32.NewProc("GlobalAlloc").Call(2, uintptr(len(chars)*2))
	if mem == 0 {
		return err
	}
	ptr, _, err := kernel32.NewProc("GlobalLock").Call(mem)
	if ptr == 0 {
		kernel32.NewProc("GlobalFree").Call(mem)
		return err
	}
	copy(unsafe.Slice((*uint16)(unsafe.Pointer(ptr)), len(chars)), chars)
	kernel32.NewProc("GlobalUnlock").Call(mem)
	user32.NewProc("EmptyClipboard").Call()
	v, _, err := user32.NewProc("SetClipboardData").Call(13, mem)
	if v == 0 {
		kernel32.NewProc("GlobalFree").Call(mem)
		return err
	}
	return nil
}

func downloadsDirectory() string {
	// FOLDERID_Downloads, including folders redirected by the user or their school.
	id := syscall.GUID{Data1: 0x374DE290, Data2: 0x123F, Data3: 0x4565, Data4: [8]byte{0x91, 0x64, 0x39, 0xC4, 0x92, 0x5E, 0x46, 0x7B}}
	var ptr *uint16
	hr, _, _ := shell32.NewProc("SHGetKnownFolderPath").Call(uintptr(unsafe.Pointer(&id)), 0, 0, uintptr(unsafe.Pointer(&ptr)))
	if hr == 0 && ptr != nil {
		defer syscall.NewLazyDLL("ole32.dll").NewProc("CoTaskMemFree").Call(uintptr(unsafe.Pointer(ptr)))
		end := 0
		for *(*uint16)(unsafe.Add(unsafe.Pointer(ptr), uintptr(end*2))) != 0 {
			end++
		}
		return syscall.UTF16ToString(unsafe.Slice(ptr, end))
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Downloads")
}

func openTarget(h uintptr, target string) error {
	result, _, _ := shell32.NewProc("ShellExecuteW").Call(h, uintptr(unsafe.Pointer(wide("open"))), uintptr(unsafe.Pointer(wide(target))), 0, 0, 1)
	if result <= 32 {
		return fmt.Errorf("Windows 无法打开 / Cannot open: %s (%d)", target, result)
	}
	return nil
}
