// UsbNetControl - native Windows GUI for USB storage and network adapter control.
// License: MIT
// Target: Windows 10/11 x64
package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

const (
	appName = "UsbNetControl"

	WS_OVERLAPPEDWINDOW = 0x00CA0000
	WS_VISIBLE          = 0x10000000
	WS_CHILD            = 0x40000000
	WS_VSCROLL          = 0x00200000
	WS_TABSTOP          = 0x00010000

	BS_PUSHBUTTON = 0x00000000
	BS_GROUPBOX   = 0x00000007

	CBS_DROPDOWNLIST = 0x0003

	WM_CREATE      = 0x0001
	WM_DESTROY     = 0x0002
	WM_COMMAND     = 0x0111
	WM_SETFONT     = 0x0030
	WM_CLOSE       = 0x0010
	WM_APP_REFRESH = 0x8001

	CB_ADDSTRING    = 0x0143
	CB_RESETCONTENT = 0x014B
	CB_GETCURSEL    = 0x0147
	CB_SETCURSEL    = 0x014E

	MB_OK          = 0x00000000
	MB_YESNO       = 0x00000004
	MB_ICONERROR   = 0x00000010
	MB_ICONWARNING = 0x00000030
	MB_ICONINFO    = 0x00000040
	IDYES          = 6

	SW_SHOWNORMAL = 1

	COLOR_WINDOW    = 5
	IDI_APPLICATION = 32512
	IDI_SHIELD      = 32518
	IDC_ARROW       = 32512

	DEFAULT_GUI_FONT = 17

	ID_USB_STATUS   = 1001
	ID_USB_DETAIL   = 1002
	ID_USB_ALLOW    = 1010
	ID_USB_READONLY = 1011
	ID_USB_BLOCK    = 1012

	ID_NET_COMBO   = 2001
	ID_NET_DETAIL  = 2002
	ID_NET_REFRESH = 2010
	ID_NET_ENABLE  = 2011
	ID_NET_DISABLE = 2012

	ID_STATUS = 3001

	CBN_SELCHANGE = 1
)

const (
	KEY_QUERY_VALUE      = 0x0001
	KEY_SET_VALUE        = 0x0002
	KEY_CREATE_SUB_KEY   = 0x0004
	KEY_WOW64_64KEY      = 0x0100
	REG_DWORD            = 4
	ERROR_SUCCESS        = 0
	ERROR_FILE_NOT_FOUND = 2
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")
	advapi32 = syscall.NewLazyDLL("advapi32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")

	procRegisterClassExW = user32.NewProc("RegisterClassExW")
	procCreateWindowExW  = user32.NewProc("CreateWindowExW")
	procDefWindowProcW   = user32.NewProc("DefWindowProcW")
	procDestroyWindow    = user32.NewProc("DestroyWindow")
	procPostQuitMessage  = user32.NewProc("PostQuitMessage")
	procGetMessageW      = user32.NewProc("GetMessageW")
	procTranslateMessage = user32.NewProc("TranslateMessage")
	procDispatchMessageW = user32.NewProc("DispatchMessageW")
	procSetWindowTextW   = user32.NewProc("SetWindowTextW")
	procSendMessageW     = user32.NewProc("SendMessageW")
	procMessageBoxW      = user32.NewProc("MessageBoxW")
	procLoadCursorW      = user32.NewProc("LoadCursorW")
	procLoadIconW        = user32.NewProc("LoadIconW")
	procPostMessageW     = user32.NewProc("PostMessageW")
	procGetSystemMetrics = user32.NewProc("GetSystemMetrics")
	procSetWindowPos     = user32.NewProc("SetWindowPos")

	procGetModuleHandleW = kernel32.NewProc("GetModuleHandleW")
	procGetStockObject   = gdi32.NewProc("GetStockObject")
	procIsUserAnAdmin    = shell32.NewProc("IsUserAnAdmin")
	procShellExecuteW    = shell32.NewProc("ShellExecuteW")

	procRegOpenKeyExW    = advapi32.NewProc("RegOpenKeyExW")
	procRegCreateKeyExW  = advapi32.NewProc("RegCreateKeyExW")
	procRegQueryValueExW = advapi32.NewProc("RegQueryValueExW")
	procRegSetValueExW   = advapi32.NewProc("RegSetValueExW")
	procRegDeleteValueW  = advapi32.NewProc("RegDeleteValueW")
	procRegCloseKey      = advapi32.NewProc("RegCloseKey")
)

var appVersion = "1.3.1"

const hkeyLocalMachine = uintptr(0x80000002)
const usbRoot = `SOFTWARE\Policies\Microsoft\Windows\RemovableStorageDevices`
const usbClass = usbRoot + `\{53f5630d-b6bf-11d0-94f2-00a0c91efb8b}`

type wndClassEx struct {
	cbSize        uint32
	style         uint32
	lpfnWndProc   uintptr
	cbClsExtra    int32
	cbWndExtra    int32
	hInstance     uintptr
	hIcon         uintptr
	hCursor       uintptr
	hbrBackground uintptr
	lpszMenuName  *uint16
	lpszClassName *uint16
	hIconSm       uintptr
}

type point struct{ x, y int32 }
type msg struct {
	hwnd     uintptr
	message  uint32
	wParam   uintptr
	lParam   uintptr
	time     uint32
	pt       point
	lPrivate uint32
}

type adapter struct {
	IfIndex     int
	Name        string
	Status      string
	LinkSpeed   string
	MacAddress  string
	IsDefault   bool
	IsActive    bool
	Description string
}

var (
	mainHwnd      uintptr
	usbStatusHwnd uintptr
	usbDetailHwnd uintptr
	netComboHwnd  uintptr
	netDetailHwnd uintptr
	statusHwnd    uintptr
	fontHandle    uintptr
	adapters      []adapter
)

func u16(s string) *uint16 { return syscall.StringToUTF16Ptr(s) }

func messageBox(hwnd uintptr, text, title string, flags uintptr) int {
	r, _, _ := procMessageBoxW.Call(hwnd, uintptr(unsafe.Pointer(u16(text))), uintptr(unsafe.Pointer(u16(title))), flags)
	return int(r)
}

func setText(hwnd uintptr, text string) {
	procSetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(u16(text))))
}

func send(hwnd uintptr, msg uint32, w, l uintptr) uintptr {
	r, _, _ := procSendMessageW.Call(hwnd, uintptr(msg), w, l)
	return r
}

func isAdmin() bool {
	r, _, _ := procIsUserAnAdmin.Call()
	return r != 0
}

func elevateOrExit() {
	if isAdmin() {
		return
	}
	exePath, err := os.Executable()
	if err != nil {
		messageBox(0, "无法获取程序路径，请右键选择“以管理员身份运行”。", appName, MB_OK|MB_ICONERROR)
		os.Exit(1)
	}
	r, _, _ := procShellExecuteW.Call(
		0,
		uintptr(unsafe.Pointer(u16("runas"))),
		uintptr(unsafe.Pointer(u16(exePath))),
		0,
		0,
		SW_SHOWNORMAL,
	)
	if r <= 32 {
		messageBox(0, "需要管理员权限才能修改 USB 策略和网卡状态。", "权限不足", MB_OK|MB_ICONWARNING)
	}
	os.Exit(0)
}

func createControl(className, text string, style uint32, x, y, w, h int32, parent uintptr, id int) uintptr {
	hCtrl, _, _ := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(u16(className))),
		uintptr(unsafe.Pointer(u16(text))),
		uintptr(style|WS_CHILD|WS_VISIBLE),
		uintptr(x), uintptr(y), uintptr(w), uintptr(h),
		parent, uintptr(id), 0, 0,
	)
	if hCtrl != 0 && fontHandle != 0 {
		send(hCtrl, WM_SETFONT, fontHandle, 1)
	}
	return hCtrl
}

func regOpen(path string, access uint32) (uintptr, error) {
	var h uintptr
	r, _, _ := procRegOpenKeyExW.Call(
		hkeyLocalMachine,
		uintptr(unsafe.Pointer(u16(path))),
		0,
		uintptr(access|KEY_WOW64_64KEY),
		uintptr(unsafe.Pointer(&h)),
	)
	if r != ERROR_SUCCESS {
		return 0, syscall.Errno(r)
	}
	return h, nil
}

func regCreate(path string) (uintptr, error) {
	var h uintptr
	var disp uint32
	r, _, _ := procRegCreateKeyExW.Call(
		hkeyLocalMachine,
		uintptr(unsafe.Pointer(u16(path))),
		0, 0, 0,
		KEY_QUERY_VALUE|KEY_SET_VALUE|KEY_CREATE_SUB_KEY|KEY_WOW64_64KEY,
		0,
		uintptr(unsafe.Pointer(&h)),
		uintptr(unsafe.Pointer(&disp)),
	)
	if r != ERROR_SUCCESS {
		return 0, syscall.Errno(r)
	}
	return h, nil
}

func regClose(h uintptr) {
	if h != 0 {
		procRegCloseKey.Call(h)
	}
}

func regGetDWORD(path, name string) (*uint32, error) {
	h, err := regOpen(path, KEY_QUERY_VALUE)
	if err != nil {
		if errno, ok := err.(syscall.Errno); ok && uintptr(errno) == ERROR_FILE_NOT_FOUND {
			return nil, nil
		}
		return nil, err
	}
	defer regClose(h)
	var typ uint32
	var value uint32
	size := uint32(4)
	r, _, _ := procRegQueryValueExW.Call(
		h,
		uintptr(unsafe.Pointer(u16(name))),
		0,
		uintptr(unsafe.Pointer(&typ)),
		uintptr(unsafe.Pointer(&value)),
		uintptr(unsafe.Pointer(&size)),
	)
	if r == ERROR_FILE_NOT_FOUND {
		return nil, nil
	}
	if r != ERROR_SUCCESS {
		return nil, syscall.Errno(r)
	}
	if typ != REG_DWORD {
		return nil, fmt.Errorf("注册表值 %s 不是 DWORD", name)
	}
	return &value, nil
}

func regSetDWORD(path, name string, value uint32) error {
	h, err := regCreate(path)
	if err != nil {
		return err
	}
	defer regClose(h)
	r, _, _ := procRegSetValueExW.Call(
		h,
		uintptr(unsafe.Pointer(u16(name))),
		0, REG_DWORD,
		uintptr(unsafe.Pointer(&value)), 4,
	)
	if r != ERROR_SUCCESS {
		return syscall.Errno(r)
	}
	return nil
}

func regDeleteValue(path, name string) error {
	h, err := regOpen(path, KEY_SET_VALUE)
	if err != nil {
		if errno, ok := err.(syscall.Errno); ok && uintptr(errno) == ERROR_FILE_NOT_FOUND {
			return nil
		}
		return err
	}
	defer regClose(h)
	r, _, _ := procRegDeleteValueW.Call(h, uintptr(unsafe.Pointer(u16(name))))
	if r == ERROR_FILE_NOT_FOUND {
		return nil
	}
	if r != ERROR_SUCCESS {
		return syscall.Errno(r)
	}
	return nil
}

func dwordIsOne(v *uint32) bool { return v != nil && *v == 1 }

func refreshUSB() {
	global, e1 := regGetDWORD(usbRoot, "Deny_All")
	read, e2 := regGetDWORD(usbClass, "Deny_Read")
	write, e3 := regGetDWORD(usbClass, "Deny_Write")
	execv, e4 := regGetDWORD(usbClass, "Deny_Execute")
	if e1 != nil || e2 != nil || e3 != nil || e4 != nil {
		setUSBVisualState(usbStateUnknown)
		setText(usbStatusHwnd, "读取状态失败")
		setText(usbDetailHwnd, "无法读取 Windows 可移动存储策略。")
		return
	}
	switch {
	case dwordIsOne(global):
		setUSBVisualState(usbStateManaged)
		setText(usbStatusHwnd, "上级策略已禁用")
		setText(usbDetailHwnd, "检测到系统 Deny_All=1，本工具不会自动删除该上级策略。")
	case dwordIsOne(read) && dwordIsOne(write):
		setUSBVisualState(usbStateBlock)
		setText(usbStatusHwnd, "USB 存储已禁用")
		setText(usbDetailHwnd, "已拒绝可移动磁盘的读取、写入和执行访问。")
	case !dwordIsOne(read) && dwordIsOne(write):
		setUSBVisualState(usbStateReadOnly)
		setText(usbStatusHwnd, "USB 存储只读")
		setText(usbDetailHwnd, "允许读取，拒绝写入；更改后建议重新插拔存储设备。")
	case !dwordIsOne(read) && !dwordIsOne(write) && !dwordIsOne(execv):
		setUSBVisualState(usbStateAllow)
		setText(usbStatusHwnd, "USB 存储允许读写")
		setText(usbDetailHwnd, "当前未设置可移动磁盘拒绝策略。")
	default:
		setUSBVisualState(usbStateCustom)
		setText(usbStatusHwnd, "检测到自定义策略")
		setText(usbDetailHwnd, "当前策略不是标准的允许 / 只读 / 禁用组合。")
	}
}

func setUSB(mode string) error {
	if h, err := regCreate(usbRoot); err != nil {
		return err
	} else {
		regClose(h)
	}
	if h, err := regCreate(usbClass); err != nil {
		return err
	} else {
		regClose(h)
	}

	switch mode {
	case "allow":
		if err := regDeleteValue(usbClass, "Deny_Read"); err != nil {
			return err
		}
		if err := regDeleteValue(usbClass, "Deny_Write"); err != nil {
			return err
		}
		if err := regDeleteValue(usbClass, "Deny_Execute"); err != nil {
			return err
		}
	case "readonly":
		if err := regDeleteValue(usbClass, "Deny_Read"); err != nil {
			return err
		}
		if err := regSetDWORD(usbClass, "Deny_Write", 1); err != nil {
			return err
		}
		if err := regDeleteValue(usbClass, "Deny_Execute"); err != nil {
			return err
		}
	case "block":
		for _, n := range []string{"Deny_Read", "Deny_Write", "Deny_Execute"} {
			if err := regSetDWORD(usbClass, n, 1); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("未知 USB 模式")
	}
	return nil
}

func runPowerShell(script string) (string, error) {
	prefix := `[Console]::OutputEncoding = New-Object System.Text.UTF8Encoding($false); $ErrorActionPreference='Stop'; `
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", prefix+script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("%s", msg)
	}
	return strings.TrimSpace(stdout.String()), nil
}

func loadAdapters() ([]adapter, error) {
	// Determine the adapter Windows is actually using for the best default route.
	// Only Up adapters are candidates, and the effective metric is RouteMetric + InterfaceMetric.
	script := "$ads=@(Get-NetAdapter -ErrorAction Stop); $up=@($ads | Where-Object { $_.Status -eq 'Up' } | Select-Object -ExpandProperty ifIndex); $defs=@(); try { $defs += @(Get-NetRoute -DestinationPrefix '0.0.0.0/0' -ErrorAction SilentlyContinue | Select-Object -ExpandProperty ifIndex -Unique); $defs += @(Get-NetRoute -DestinationPrefix '::/0' -ErrorAction SilentlyContinue | Select-Object -ExpandProperty ifIndex -Unique) } catch {}; $active=$null; try { $r4=@(Get-NetRoute -DestinationPrefix '0.0.0.0/0' -AddressFamily IPv4 -ErrorAction SilentlyContinue | Where-Object { $up -contains [int]$_.ifIndex } | ForEach-Object { $ip=Get-NetIPInterface -InterfaceIndex $_.ifIndex -AddressFamily IPv4 -ErrorAction SilentlyContinue | Select-Object -First 1; $im=if($ip){[int]$ip.InterfaceMetric}else{0}; [pscustomobject]@{IfIndex=[int]$_.ifIndex;Score=([int]$_.RouteMetric+$im)} } | Sort-Object Score,IfIndex); if($r4.Count -gt 0){$active=[int]$r4[0].IfIndex} } catch {}; if($null -eq $active){ try { $r6=@(Get-NetRoute -DestinationPrefix '::/0' -AddressFamily IPv6 -ErrorAction SilentlyContinue | Where-Object { $up -contains [int]$_.ifIndex } | ForEach-Object { $ip=Get-NetIPInterface -InterfaceIndex $_.ifIndex -AddressFamily IPv6 -ErrorAction SilentlyContinue | Select-Object -First 1; $im=if($ip){[int]$ip.InterfaceMetric}else{0}; [pscustomobject]@{IfIndex=[int]$_.ifIndex;Score=([int]$_.RouteMetric+$im)} } | Sort-Object Score,IfIndex); if($r6.Count -gt 0){$active=[int]$r6[0].IfIndex} } catch {} }; if($null -eq $active -and $up.Count -gt 0){$active=[int]$up[0]}; $ads | Sort-Object ifIndex | ForEach-Object { $d=if($defs -contains [int]$_.ifIndex){'1'}else{'0'}; $a=if($null -ne $active -and [int]$_.ifIndex -eq $active){'1'}else{'0'}; $f=@([string]$_.ifIndex,[string]$_.Name,[string]$_.Status,[string]$_.LinkSpeed,[string]$_.MacAddress,$d,$a,[string]$_.InterfaceDescription); (($f | ForEach-Object { $_ -replace \"`t\",' ' -replace \"`r|`n\",' ' }) -join \"`t\") }"
	out, err := runPowerShell(script)
	if err != nil {
		return nil, err
	}
	var result []adapter
	if strings.TrimSpace(out) == "" {
		return result, nil
	}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimRight(line, "\r")
		parts := strings.SplitN(line, "\t", 8)
		if len(parts) < 8 {
			continue
		}
		idx, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			continue
		}
		result = append(result, adapter{
			IfIndex:     idx,
			Name:        parts[1],
			Status:      parts[2],
			LinkSpeed:   parts[3],
			MacAddress:  parts[4],
			IsDefault:   strings.TrimSpace(parts[5]) == "1",
			IsActive:    strings.TrimSpace(parts[6]) == "1",
			Description: parts[7],
		})
	}
	return result, nil
}

func refreshAdapters() {
	setText(statusHwnd, "正在读取网卡...")

	list, err := loadAdapters()
	if err != nil {
		adapters = nil
		send(netComboHwnd, CB_RESETCONTENT, 0, 0)
		setNetVisualState("")
		setText(netStateHwnd, "读取失败")
		setText(netSpeedHwnd, "-")
		setText(netMacHwnd, "-")
		setText(netRouteHwnd, "-")
		setText(netDescHwnd, "无法读取网卡列表。")
		procEnableWindow.Call(netEnableButton, 0)
		procEnableWindow.Call(netDisableButton, 0)
		setText(statusHwnd, "读取网卡失败")
		messageBox(mainHwnd, "无法读取网卡列表：\r\n"+err.Error(), "操作失败", MB_OK|MB_ICONERROR)
		return
	}
	adapters = list
	send(netComboHwnd, CB_RESETCONTENT, 0, 0)
	selectedIndex := 0
	activeIndex := -1
	firstUpIndex := -1
	for i, a := range adapters {
		label := fmt.Sprintf("%s (接口 %d)", a.Name, a.IfIndex)
		send(netComboHwnd, CB_ADDSTRING, 0, uintptr(unsafe.Pointer(u16(label))))
		if a.IsActive {
			activeIndex = i
		}
		if firstUpIndex < 0 && strings.EqualFold(strings.TrimSpace(a.Status), "Up") {
			firstUpIndex = i
		}
	}
	// Startup and Refresh always focus the adapter currently used by Windows.
	// If no best default route can be identified, fall back to the first Up adapter.
	if activeIndex >= 0 {
		selectedIndex = activeIndex
	} else if firstUpIndex >= 0 {
		selectedIndex = firstUpIndex
	}
	if len(adapters) > 0 {
		send(netComboHwnd, CB_SETCURSEL, uintptr(selectedIndex), 0)
		updateAdapterDetail()
	} else {
		setNetVisualState("")
		setText(netStateHwnd, "未检测到")
		setText(netSpeedHwnd, "-")
		setText(netMacHwnd, "-")
		setText(netRouteHwnd, "-")
		setText(netDescHwnd, "未检测到网络适配器。")
		procEnableWindow.Call(netEnableButton, 0)
		procEnableWindow.Call(netDisableButton, 0)
	}
	setText(statusHwnd, fmt.Sprintf("就绪 · 已加载 %d 个网络适配器", len(adapters)))
}

func selectedAdapter() (*adapter, bool) {
	idx := int(int32(send(netComboHwnd, CB_GETCURSEL, 0, 0)))
	if idx < 0 || idx >= len(adapters) {
		return nil, false
	}
	return &adapters[idx], true
}

func updateAdapterDetail() {
	a, ok := selectedAdapter()
	if !ok {
		setNetVisualState("")
		setText(netStateHwnd, "请选择网卡")
		setText(netSpeedHwnd, "-")
		setText(netMacHwnd, "-")
		setText(netRouteHwnd, "-")
		setText(netDescHwnd, "请选择一个网络适配器。")
		procEnableWindow.Call(netEnableButton, 0)
		procEnableWindow.Call(netDisableButton, 0)
		return
	}

	setNetVisualState(a.Status)
	statusText := a.Status
	switch strings.ToLower(strings.TrimSpace(a.Status)) {
	case "up", "connected":
		statusText = "已连接"
	case "disconnected":
		statusText = "未连接"
	case "disabled":
		statusText = "已禁用"
	case "not present":
		statusText = "设备不存在"
	case "lowerlayerdown", "down":
		statusText = "连接中断"
	}

	routeText := "否"
	if a.IsActive {
		routeText = "是 · 正在使用"
	} else if a.IsDefault {
		routeText = "是 · 备用默认路由"
	}

	setText(netStateHwnd, statusText)
	setText(netSpeedHwnd, emptyAsDash(a.LinkSpeed))
	setText(netMacHwnd, emptyAsDash(a.MacAddress))
	setText(netRouteHwnd, routeText)
	setText(netDescHwnd, "设备："+emptyAsDash(a.Description))

	isDisabled := strings.EqualFold(strings.TrimSpace(a.Status), "Disabled")
	if isDisabled {
		procEnableWindow.Call(netEnableButton, 1)
		procEnableWindow.Call(netDisableButton, 0)
	} else {
		procEnableWindow.Call(netEnableButton, 0)
		procEnableWindow.Call(netDisableButton, 1)
	}
}

func emptyAsDash(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "-"
	}
	return s
}

func setAdapterEnabled(ifIndex int, enable bool) error {
	verb := "Enable-NetAdapter"
	if !enable {
		verb = "Disable-NetAdapter"
	}
	script := fmt.Sprintf("Get-NetAdapter -InterfaceIndex %d -ErrorAction Stop | %s -Confirm:$false -ErrorAction Stop | Out-Null", ifIndex, verb)
	_, err := runPowerShell(script)
	return err
}

func handleUSB(mode string) {
	if mode == "block" {
		ans := messageBox(mainHwnd, "确定要禁用 USB 存储吗？\r\n\r\n这会拒绝可移动磁盘的读取、写入和执行访问，但不会禁用 USB 键盘和鼠标。", "确认禁用 USB 存储", MB_YESNO|MB_ICONWARNING)
		if ans != IDYES {
			return
		}
	}
	if err := setUSB(mode); err != nil {
		messageBox(mainHwnd, "USB 策略修改失败：\r\n"+err.Error(), "操作失败", MB_OK|MB_ICONERROR)
		return
	}
	refreshUSB()
	switch mode {
	case "allow":
		setText(statusHwnd, "USB 存储已设置为允许读写")
	case "readonly":
		setText(statusHwnd, "USB 存储已设置为只读")
	case "block":
		setText(statusHwnd, "USB 存储已禁用")
	}
}

func handleAdapter(enable bool) {
	a, ok := selectedAdapter()
	if !ok {
		return
	}
	if !enable {
		warning := fmt.Sprintf("确定要禁用网卡“%s”吗？\r\n\r\n禁用后，该网卡的网络连接会立即中断。", a.Name)
		if a.IsDefault {
			warning += "\r\n\r\n警告：此网卡当前承担默认路由。若你正在远程连接这台电脑，连接可能立即断开，且无法远程恢复。"
		}
		if messageBox(mainHwnd, warning, "确认禁用网卡", MB_YESNO|MB_ICONWARNING) != IDYES {
			return
		}
	}
	action := "启用"
	if !enable {
		action = "禁用"
	}
	setText(statusHwnd, fmt.Sprintf("正在%s %s...", action, a.Name))
	if err := setAdapterEnabled(a.IfIndex, enable); err != nil {
		messageBox(mainHwnd, action+"网卡失败：\r\n"+err.Error(), "操作失败", MB_OK|MB_ICONERROR)
		setText(statusHwnd, action+"网卡失败")
		return
	}
	name := a.Name
	refreshAdapters()
	setText(statusHwnd, fmt.Sprintf("已%s网卡：%s", action, name))
}

func wndProc(hwnd uintptr, message uint32, wParam, lParam uintptr) uintptr {
	if handled, result := modernWndProc(hwnd, message, wParam, lParam); handled {
		return result
	}
	switch message {
	case WM_CREATE:
		mainHwnd = hwnd
		fontHandle, _, _ = procGetStockObject.Call(DEFAULT_GUI_FONT)

		createControl("STATIC", "USB 与网卡控制", 0, 22, 16, 300, 28, hwnd, 0)
		createControl("STATIC", "原生单文件工具：USB 存储策略 + 本机网络适配器管理", 0, 22, 45, 650, 24, hwnd, 0)

		createControl("BUTTON", "USB 存储访问", BS_GROUPBOX, 20, 78, 840, 170, hwnd, 0)
		usbStatusHwnd = createControl("STATIC", "正在读取 USB 策略...", 0, 42, 108, 760, 24, hwnd, ID_USB_STATUS)
		usbDetailHwnd = createControl("STATIC", "", 0, 42, 136, 780, 24, hwnd, ID_USB_DETAIL)
		createControl("BUTTON", "允许读写", BS_PUSHBUTTON|WS_TABSTOP, 42, 172, 135, 38, hwnd, ID_USB_ALLOW)
		createControl("BUTTON", "设置为只读", BS_PUSHBUTTON|WS_TABSTOP, 192, 172, 135, 38, hwnd, ID_USB_READONLY)
		createControl("BUTTON", "禁用 USB 存储", BS_PUSHBUTTON|WS_TABSTOP, 342, 172, 160, 38, hwnd, ID_USB_BLOCK)
		createControl("STATIC", "仅控制可移动磁盘存储访问，不会禁用 USB 键盘、鼠标、打印机或 USB 网卡。策略变更后建议重新插拔存储设备。", 0, 42, 218, 790, 22, hwnd, 0)

		createControl("BUTTON", "网卡控制", BS_GROUPBOX, 20, 260, 840, 258, hwnd, 0)
		createControl("STATIC", "选择网卡：", 0, 42, 294, 95, 24, hwnd, 0)
		netComboHwnd = createControl("COMBOBOX", "", CBS_DROPDOWNLIST|WS_VSCROLL|WS_TABSTOP, 122, 289, 405, 260, hwnd, ID_NET_COMBO)
		createControl("BUTTON", "刷新列表", BS_PUSHBUTTON|WS_TABSTOP, 544, 286, 92, 34, hwnd, ID_NET_REFRESH)
		createControl("BUTTON", "启用选中网卡", BS_PUSHBUTTON|WS_TABSTOP, 646, 286, 118, 34, hwnd, ID_NET_ENABLE)
		createControl("BUTTON", "禁用选中网卡", BS_PUSHBUTTON|WS_TABSTOP, 772, 286, 118, 34, hwnd, ID_NET_DISABLE)
		netDetailHwnd = createControl("STATIC", "正在读取网卡...", 0, 42, 342, 790, 96, hwnd, ID_NET_DETAIL)
		createControl("STATIC", "提示：禁用承担默认路由的网卡会立即断网；远程操作时尤其谨慎。", 0, 42, 460, 760, 26, hwnd, 0)

		statusHwnd = createControl("STATIC", "就绪", 0, 22, 536, 830, 24, hwnd, ID_STATUS)
		procPostMessageW.Call(hwnd, WM_APP_REFRESH, 0, 0)
		return 0

	case WM_APP_REFRESH:
		refreshUSB()
		refreshAdapters()
		return 0

	case WM_COMMAND:
		id := int(wParam & 0xFFFF)
		code := int((wParam >> 16) & 0xFFFF)
		switch id {
		case ID_USB_ALLOW:
			if code == 0 {
				handleUSB("allow")
			}
		case ID_USB_READONLY:
			if code == 0 {
				handleUSB("readonly")
			}
		case ID_USB_BLOCK:
			if code == 0 {
				handleUSB("block")
			}
		case ID_NET_REFRESH:
			if code == 0 {
				refreshAdapters()
			}
		case ID_NET_ENABLE:
			if code == 0 {
				handleAdapter(true)
			}
		case ID_NET_DISABLE:
			if code == 0 {
				handleAdapter(false)
			}
		case ID_NET_COMBO:
			if code == CBN_SELCHANGE {
				updateAdapterDetail()
			}
		}
		return 0

	case WM_CLOSE:
		procDestroyWindow.Call(hwnd)
		return 0
	case WM_DESTROY:
		procPostQuitMessage.Call(0)
		return 0
	}
	r, _, _ := procDefWindowProcW.Call(hwnd, uintptr(message), wParam, lParam)
	return r
}

func centerWindow(hwnd uintptr, width, height int32) {
	sw, _, _ := procGetSystemMetrics.Call(0)
	sh, _, _ := procGetSystemMetrics.Call(1)
	x := (int32(sw) - width) / 2
	y := (int32(sh) - height) / 2
	const SWP_NOSIZE = 0x0001
	const SWP_NOZORDER = 0x0004
	procSetWindowPos.Call(hwnd, 0, uintptr(x), uintptr(y), 0, 0, SWP_NOSIZE|SWP_NOZORDER)
}

func main() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	elevateOrExit()

	hInst, _, _ := procGetModuleHandleW.Call(0)
	className := u16("UsbNetControlWindowClass")
	icon, _, _ := procLoadIconW.Call(0, IDI_SHIELD)
	if icon == 0 {
		icon, _, _ = procLoadIconW.Call(0, IDI_APPLICATION)
	}
	cursor, _, _ := procLoadCursorW.Call(0, IDC_ARROW)
	wc := wndClassEx{
		cbSize:        uint32(unsafe.Sizeof(wndClassEx{})),
		lpfnWndProc:   syscall.NewCallback(wndProc),
		hInstance:     hInst,
		hIcon:         icon,
		hCursor:       cursor,
		hbrBackground: COLOR_WINDOW + 1,
		lpszClassName: className,
		hIconSm:       icon,
	}
	atom, _, regErr := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	if atom == 0 {
		messageBox(0, "窗口类注册失败："+regErr.Error(), appName, MB_OK|MB_ICONERROR)
		return
	}

	const width = 980
	const height = 700
	hwnd, _, createErr := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(u16("UsbNetControl - USB 与网卡控制"))),
		WS_OVERLAPPEDWINDOW|WS_VISIBLE,
		100, 100, width, height,
		0, 0, hInst, 0,
	)
	if hwnd == 0 {
		messageBox(0, "主窗口创建失败："+createErr.Error(), appName, MB_OK|MB_ICONERROR)
		return
	}
	centerWindow(hwnd, width, height)

	var m msg
	for {
		r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) == -1 || r == 0 {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
}
