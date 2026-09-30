package main

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"unsafe"

	"github.com/lxn/win"
)

const (
	trayCallback = win.WM_APP + 1
	trayUpdate   = win.WM_APP + 2
)

// Window callbacks, icon creation and Shell calls all run on the UI thread.
func runPlatform(path string, _ tickerConfig) error {
	listener, err := listenPipe(path)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", path, err)
	}
	defer listener.Close()
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	updates := make(chan string, 1)
	notifications := make(chan string, 1)
	done := make(chan struct{})
	defer close(done)
	var data win.NOTIFYICONDATA
	data.CbSize = uint32(unsafe.Sizeof(data))
	data.UID = 1
	data.UFlags = win.NIF_ICON | win.NIF_MESSAGE | win.NIF_TIP
	data.UCallbackMessage = trayCallback
	copy(data.SzTip[:], syscall.StringToUTF16("sock-emoji"))
	data.HIcon, err = renderIcon("🧦")
	if err != nil {
		return err
	}
	defer func() { win.DestroyIcon(data.HIcon) }()
	restarted := win.RegisterWindowMessage(syscall.StringToUTF16Ptr("TaskbarCreated"))
	if restarted == 0 {
		return fmt.Errorf("register TaskbarCreated failed")
	}
	callback := syscall.NewCallback(func(hwnd win.HWND, msg uint32, wparam, lparam uintptr) uintptr {
		switch msg {
		case restarted:
			win.Shell_NotifyIcon(win.NIM_ADD, &data)
		case trayUpdate:
			select {
			case value := <-notifications:
				icon, err := renderIcon(iconText(value))
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					break
				}
				previous := data.HIcon
				data.HIcon = icon
				data.SzTip = [128]uint16{}
				// Bound the tooltip without splitting a UTF-16 surrogate pair.
				tip := syscall.StringToUTF16("sock-emoji: " + value)
				if len(tip) > 128 {
					tip = tip[:127]
					if tip[len(tip)-1] >= 0xd800 && tip[len(tip)-1] <= 0xdbff {
						tip = tip[:len(tip)-1]
					}
				}
				copy(data.SzTip[:], tip)
				if !win.Shell_NotifyIcon(win.NIM_MODIFY, &data) {
					win.Shell_NotifyIcon(win.NIM_ADD, &data)
				}
				win.DestroyIcon(previous)
			default:
			}
		case trayCallback:
			if uint32(lparam) == win.WM_RBUTTONUP || uint32(lparam) == win.WM_CONTEXTMENU {
				showTrayMenu(hwnd)
			}
		case win.WM_CLOSE:
			win.DestroyWindow(hwnd)
		case win.WM_DESTROY:
			win.Shell_NotifyIcon(win.NIM_DELETE, &data)
			win.PostQuitMessage(0)
		default:
			return win.DefWindowProc(hwnd, msg, wparam, lparam)
		}
		return 0
	})
	class := syscall.StringToUTF16Ptr("SockEmojiWindow")
	wc := win.WNDCLASSEX{LpfnWndProc: callback, HInstance: win.GetModuleHandle(nil), LpszClassName: class}
	wc.CbSize = uint32(unsafe.Sizeof(wc))
	if win.RegisterClassEx(&wc) == 0 {
		return fmt.Errorf("register tray window failed")
	}
	defer win.UnregisterClass(class)
	// A hidden top-level window receives Explorer's TaskbarCreated broadcast.
	hwnd := win.CreateWindowEx(0, class, syscall.StringToUTF16Ptr(path), 0, 0, 0, 0, 0, 0, 0, wc.HInstance, nil)
	if hwnd == 0 {
		return fmt.Errorf("create tray window failed")
	}
	defer win.DestroyWindow(hwnd)
	data.HWnd = hwnd
	if !win.Shell_NotifyIcon(win.NIM_ADD, &data) {
		return fmt.Errorf("add tray icon failed")
	}
	defer win.Shell_NotifyIcon(win.NIM_DELETE, &data)
	go servePipe(listener, updates)
	go func() {
		for {
			select {
			case value := <-updates:
				sendLatest(notifications, value)
				win.PostMessage(hwnd, trayUpdate, 0, 0)
			case <-done:
				return
			}
		}
	}()
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)
	go func() {
		select {
		case <-signals:
			win.PostMessage(hwnd, win.WM_CLOSE, 0, 0)
		case <-done:
		}
	}()
	var msg win.MSG
	for {
		result := win.GetMessage(&msg, 0, 0, 0)
		if result == -1 {
			return fmt.Errorf("tray message loop failed")
		}
		if result == 0 {
			return nil
		}
		win.TranslateMessage(&msg)
		win.DispatchMessage(&msg)
	}
}

func showTrayMenu(hwnd win.HWND) {
	menu := win.CreatePopupMenu()
	if menu == 0 {
		return
	}
	defer win.DestroyMenu(menu)
	item := win.MENUITEMINFO{FMask: win.MIIM_ID | win.MIIM_STRING, WID: 1, DwTypeData: syscall.StringToUTF16Ptr("Quit")}
	item.CbSize = uint32(unsafe.Sizeof(item))
	win.InsertMenuItem(menu, 0, true, &item)
	var point win.POINT
	win.GetCursorPos(&point)
	win.SetForegroundWindow(hwnd)
	command := win.TrackPopupMenu(menu, win.TPM_RETURNCMD|win.TPM_RIGHTBUTTON, point.X, point.Y, 0, hwnd, nil)
	win.PostMessage(hwnd, win.WM_NULL, 0, 0)
	if command == 1 {
		win.PostMessage(hwnd, win.WM_CLOSE, 0, 0)
	}
}
