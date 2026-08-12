package main

import (
	"fmt"
	"strings"
	"syscall"
	"unsafe"
)

const (
	WM_PAINT          = 0x000F
	WM_ERASEBKGND     = 0x0014
	WM_DRAWITEM       = 0x002B
	WM_CTLCOLORSTATIC = 0x0138

	BS_OWNERDRAW         = 0x0000000B
	CBS_OWNERDRAWFIXED   = 0x0010
	CBS_HASSTRINGS       = 0x0200
	CBS_NOINTEGRALHEIGHT = 0x0400
	SS_CENTER            = 0x00000001
	SS_NOPREFIX          = 0x00000080

	ODT_COMBOBOX     = 3
	ODT_BUTTON       = 4
	ODS_SELECTED     = 0x0001
	ODS_DISABLED     = 0x0004
	ODS_FOCUS        = 0x0010
	ODS_HOTLIGHT     = 0x0040
	ODS_COMBOBOXEDIT = 0x1000

	TRANSPARENT = 1
	PS_SOLID    = 0

	DT_LEFT       = 0x00000000
	DT_CENTER     = 0x00000001
	DT_VCENTER    = 0x00000004
	DT_SINGLELINE = 0x00000020
	DT_NOPREFIX   = 0x00000800

	FW_NORMAL         = 400
	FW_SEMIBOLD       = 600
	FW_BOLD           = 700
	CLEARTYPE_QUALITY = 5

	CB_SETITEMHEIGHT = 0x0153

	ID_HEADER_TITLE    = 4001
	ID_HEADER_SUBTITLE = 4002
	ID_USB_HEADING     = 4010
	ID_USB_HELPER      = 4011
	ID_USB_STATE_LABEL = 4012
	ID_NET_HEADING     = 4020
	ID_NET_HELPER      = 4021
	ID_NET_LABEL       = 4022
	ID_NET_HINT        = 4023
	ID_NET_STATE_LABEL = 4030
	ID_NET_SPEED_LABEL = 4031
	ID_NET_MAC_LABEL   = 4032
	ID_NET_ROUTE_LABEL = 4033
	ID_NET_STATE       = 4040
	ID_NET_SPEED       = 4041
	ID_NET_MAC         = 4042
	ID_NET_ROUTE       = 4043
	ID_NET_DESC        = 4044
)

const (
	usbStateUnknown = iota
	usbStateAllow
	usbStateReadOnly
	usbStateBlock
	usbStateManaged
	usbStateCustom
)

const (
	netStateUnknown = iota
	netStateUp
	netStateDisabled
	netStateDown
)

var (
	uiUser32  = syscall.NewLazyDLL("user32.dll")
	uiGdi32   = syscall.NewLazyDLL("gdi32.dll")
	uiUxTheme = syscall.NewLazyDLL("uxtheme.dll")
	uiDwmapi  = syscall.NewLazyDLL("dwmapi.dll")

	procBeginPaint     = uiUser32.NewProc("BeginPaint")
	procEndPaint       = uiUser32.NewProc("EndPaint")
	procGetClientRect  = uiUser32.NewProc("GetClientRect")
	procFillRect       = uiUser32.NewProc("FillRect")
	procInvalidateRect = uiUser32.NewProc("InvalidateRect")
	procGetDlgCtrlID   = uiUser32.NewProc("GetDlgCtrlID")
	procDrawTextW      = uiUser32.NewProc("DrawTextW")
	procEnableWindow   = uiUser32.NewProc("EnableWindow")

	procCreateFontW      = uiGdi32.NewProc("CreateFontW")
	procCreateSolidBrush = uiGdi32.NewProc("CreateSolidBrush")
	procCreatePen        = uiGdi32.NewProc("CreatePen")
	procSelectObject     = uiGdi32.NewProc("SelectObject")
	procDeleteObject     = uiGdi32.NewProc("DeleteObject")
	procSetBkMode        = uiGdi32.NewProc("SetBkMode")
	procSetTextColor     = uiGdi32.NewProc("SetTextColor")
	procRoundRect        = uiGdi32.NewProc("RoundRect")
	procEllipse          = uiGdi32.NewProc("Ellipse")

	procSetWindowTheme        = uiUxTheme.NewProc("SetWindowTheme")
	procDwmSetWindowAttribute = uiDwmapi.NewProc("DwmSetWindowAttribute")

	logoFont    uintptr
	titleFont   uintptr
	sectionFont uintptr
	stateFont   uintptr
	valueFont   uintptr
	bodyFont    uintptr
	smallFont   uintptr
	tinyFont    uintptr
	buttonFont  uintptr

	brushPage        uintptr
	brushWhite       uintptr
	brushDetail      uintptr
	brushFooter      uintptr
	brushAccent      uintptr
	brushAccentSoft  uintptr
	brushSuccessSoft uintptr
	brushWarnSoft    uintptr
	brushDangerSoft  uintptr
	brushPurpleSoft  uintptr
	brushShadow      uintptr

	penCard   uintptr
	penDetail uintptr
	penShadow uintptr

	usbVisualState = usbStateUnknown
	netVisualState = netStateUnknown

	usbAllowButton    uintptr
	usbReadonlyButton uintptr
	usbBlockButton    uintptr
	netRefreshButton  uintptr
	netEnableButton   uintptr
	netDisableButton  uintptr

	netStateHwnd uintptr
	netSpeedHwnd uintptr
	netMacHwnd   uintptr
	netRouteHwnd uintptr
	netDescHwnd  uintptr
)

type uiRect struct {
	left, top, right, bottom int32
}

type paintStruct struct {
	hdc         uintptr
	fErase      int32
	rcPaint     uiRect
	fRestore    int32
	fIncUpdate  int32
	rgbReserved [32]byte
}

type drawItemStruct struct {
	CtlType    uint32
	CtlID      uint32
	itemID     uint32
	itemAction uint32
	itemState  uint32
	hwndItem   uintptr
	hDC        uintptr
	rcItem     uiRect
	itemData   uintptr
}

func rgb(r, g, b byte) uintptr {
	return uintptr(uint32(r) | uint32(g)<<8 | uint32(b)<<16)
}

func createUIFont(height int32, weight int32) uintptr {
	h, _, _ := procCreateFontW.Call(
		uintptr(height), 0, 0, 0,
		uintptr(weight), 0, 0, 0,
		1, 0, 0, CLEARTYPE_QUALITY, 0,
		uintptr(unsafe.Pointer(u16("Microsoft YaHei UI"))),
	)
	return h
}

func setFont(hwnd, font uintptr) {
	if hwnd != 0 && font != 0 {
		send(hwnd, WM_SETFONT, font, 1)
	}
}

func initModernResources() {
	if bodyFont != 0 {
		return
	}
	logoFont = createUIFont(-21, FW_BOLD)
	titleFont = createUIFont(-25, FW_SEMIBOLD)
	sectionFont = createUIFont(-18, FW_SEMIBOLD)
	stateFont = createUIFont(-19, FW_SEMIBOLD)
	valueFont = createUIFont(-16, FW_SEMIBOLD)
	bodyFont = createUIFont(-15, FW_NORMAL)
	smallFont = createUIFont(-13, FW_NORMAL)
	tinyFont = createUIFont(-12, FW_NORMAL)
	buttonFont = createUIFont(-14, FW_SEMIBOLD)
	fontHandle = bodyFont

	brushPage, _, _ = procCreateSolidBrush.Call(rgb(245, 248, 252))
	brushWhite, _, _ = procCreateSolidBrush.Call(rgb(255, 255, 255))
	brushDetail, _, _ = procCreateSolidBrush.Call(rgb(248, 250, 252))
	brushFooter, _, _ = procCreateSolidBrush.Call(rgb(250, 252, 255))
	brushAccent, _, _ = procCreateSolidBrush.Call(rgb(37, 99, 235))
	brushAccentSoft, _, _ = procCreateSolidBrush.Call(rgb(239, 246, 255))
	brushSuccessSoft, _, _ = procCreateSolidBrush.Call(rgb(240, 253, 244))
	brushWarnSoft, _, _ = procCreateSolidBrush.Call(rgb(255, 251, 235))
	brushDangerSoft, _, _ = procCreateSolidBrush.Call(rgb(254, 242, 242))
	brushPurpleSoft, _, _ = procCreateSolidBrush.Call(rgb(250, 245, 255))
	brushShadow, _, _ = procCreateSolidBrush.Call(rgb(232, 238, 246))

	penCard, _, _ = procCreatePen.Call(PS_SOLID, 1, rgb(220, 229, 240))
	penDetail, _, _ = procCreatePen.Call(PS_SOLID, 1, rgb(226, 232, 240))
	penShadow, _, _ = procCreatePen.Call(PS_SOLID, 1, rgb(232, 238, 246))
}

func createModernUI(hwnd uintptr) {
	mainHwnd = hwnd
	initModernResources()

	// Windows 11 rounded window corners when supported. Safe no-op on older systems.
	corner := uint32(2)
	procDwmSetWindowAttribute.Call(hwnd, 33, uintptr(unsafe.Pointer(&corner)), unsafe.Sizeof(corner))

	h := createControl("STATIC", "UsbNetControl", SS_NOPREFIX, 84, 17, 360, 34, hwnd, ID_HEADER_TITLE)
	setFont(h, titleFont)
	h = createControl("STATIC", "USB 存储与网络适配器控制", SS_NOPREFIX, 84, 51, 440, 24, hwnd, ID_HEADER_SUBTITLE)
	setFont(h, bodyFont)

	h = createControl("STATIC", "USB 存储访问", SS_NOPREFIX, 44, 126, 250, 28, hwnd, ID_USB_HEADING)
	setFont(h, sectionFont)
	h = createControl("STATIC", "仅控制可移动磁盘存储访问，不影响 USB 键盘、鼠标、打印机或 USB 网卡", SS_NOPREFIX, 44, 154, 780, 22, hwnd, ID_USB_HELPER)
	setFont(h, smallFont)

	h = createControl("STATIC", "当前状态", SS_NOPREFIX, 64, 201, 110, 18, hwnd, ID_USB_STATE_LABEL)
	setFont(h, tinyFont)
	usbStatusHwnd = createControl("STATIC", "正在读取状态...", SS_NOPREFIX, 64, 222, 382, 28, hwnd, ID_USB_STATUS)
	setFont(usbStatusHwnd, stateFont)
	usbDetailHwnd = createControl("STATIC", "", SS_NOPREFIX, 64, 252, 382, 24, hwnd, ID_USB_DETAIL)
	setFont(usbDetailHwnd, smallFont)

	usbAllowButton = createControl("BUTTON", "允许读写", BS_OWNERDRAW|WS_TABSTOP, 492, 211, 140, 42, hwnd, ID_USB_ALLOW)
	setFont(usbAllowButton, buttonFont)
	usbReadonlyButton = createControl("BUTTON", "设为只读", BS_OWNERDRAW|WS_TABSTOP, 642, 211, 140, 42, hwnd, ID_USB_READONLY)
	setFont(usbReadonlyButton, buttonFont)
	usbBlockButton = createControl("BUTTON", "禁用 USB 存储", BS_OWNERDRAW|WS_TABSTOP, 792, 211, 140, 42, hwnd, ID_USB_BLOCK)
	setFont(usbBlockButton, buttonFont)

	h = createControl("STATIC", "网络适配器", SS_NOPREFIX, 44, 333, 250, 28, hwnd, ID_NET_HEADING)
	setFont(h, sectionFont)
	h = createControl("STATIC", "选择本机网卡后，可单独启用或禁用；默认路由网卡会额外提醒", SS_NOPREFIX, 44, 361, 780, 22, hwnd, ID_NET_HELPER)
	setFont(h, smallFont)

	h = createControl("STATIC", "选择网卡", SS_NOPREFIX, 44, 401, 76, 24, hwnd, ID_NET_LABEL)
	setFont(h, bodyFont)
	netComboHwnd = createControl("COMBOBOX", "", CBS_DROPDOWNLIST|CBS_OWNERDRAWFIXED|CBS_HASSTRINGS|CBS_NOINTEGRALHEIGHT|WS_VSCROLL|WS_TABSTOP, 132, 392, 648, 238, hwnd, ID_NET_COMBO)
	setFont(netComboHwnd, bodyFont)
	send(netComboHwnd, CB_SETITEMHEIGHT, 0, 34)
	send(netComboHwnd, CB_SETITEMHEIGHT, ^uintptr(0), 36)
	procSetWindowTheme.Call(netComboHwnd, uintptr(unsafe.Pointer(u16("Explorer"))), 0)
	netRefreshButton = createControl("BUTTON", "刷新列表", BS_OWNERDRAW|WS_TABSTOP, 800, 391, 132, 42, hwnd, ID_NET_REFRESH)
	setFont(netRefreshButton, buttonFont)

	h = createControl("STATIC", "连接状态", SS_NOPREFIX, 64, 465, 110, 18, hwnd, ID_NET_STATE_LABEL)
	setFont(h, tinyFont)
	h = createControl("STATIC", "链路速率", SS_NOPREFIX, 252, 465, 110, 18, hwnd, ID_NET_SPEED_LABEL)
	setFont(h, tinyFont)
	h = createControl("STATIC", "MAC 地址", SS_NOPREFIX, 416, 465, 110, 18, hwnd, ID_NET_MAC_LABEL)
	setFont(h, tinyFont)
	h = createControl("STATIC", "默认路由", SS_NOPREFIX, 650, 465, 110, 18, hwnd, ID_NET_ROUTE_LABEL)
	setFont(h, tinyFont)

	netStateHwnd = createControl("STATIC", "正在读取...", SS_NOPREFIX, 64, 487, 158, 26, hwnd, ID_NET_STATE)
	setFont(netStateHwnd, valueFont)
	netSpeedHwnd = createControl("STATIC", "-", SS_NOPREFIX, 252, 487, 126, 26, hwnd, ID_NET_SPEED)
	setFont(netSpeedHwnd, valueFont)
	netMacHwnd = createControl("STATIC", "-", SS_NOPREFIX, 416, 487, 200, 26, hwnd, ID_NET_MAC)
	setFont(netMacHwnd, valueFont)
	netRouteHwnd = createControl("STATIC", "-", SS_NOPREFIX, 650, 487, 218, 26, hwnd, ID_NET_ROUTE)
	setFont(netRouteHwnd, valueFont)
	netDescHwnd = createControl("STATIC", "正在读取网卡...", SS_NOPREFIX, 64, 529, 814, 22, hwnd, ID_NET_DESC)
	setFont(netDescHwnd, smallFont)
	netDetailHwnd = netDescHwnd

	h = createControl("STATIC", "提示：禁用当前联网网卡会立即断网，远程操作时请谨慎。", SS_NOPREFIX, 44, 574, 590, 24, hwnd, ID_NET_HINT)
	setFont(h, smallFont)
	netEnableButton = createControl("BUTTON", "启用选中网卡", BS_OWNERDRAW|WS_TABSTOP, 656, 563, 132, 42, hwnd, ID_NET_ENABLE)
	setFont(netEnableButton, buttonFont)
	netDisableButton = createControl("BUTTON", "禁用选中网卡", BS_OWNERDRAW|WS_TABSTOP, 800, 563, 132, 42, hwnd, ID_NET_DISABLE)
	setFont(netDisableButton, buttonFont)

	statusHwnd = createControl("STATIC", "就绪", SS_NOPREFIX, 48, 635, 700, 24, hwnd, ID_STATUS)
	setFont(statusHwnd, smallFont)

	procPostMessageW.Call(hwnd, WM_APP_REFRESH, 0, 0)
}

func usbStateBrush() uintptr {
	switch usbVisualState {
	case usbStateAllow:
		return brushSuccessSoft
	case usbStateReadOnly:
		return brushWarnSoft
	case usbStateBlock:
		return brushDangerSoft
	case usbStateManaged:
		return brushPurpleSoft
	case usbStateCustom:
		return brushWarnSoft
	default:
		return brushAccentSoft
	}
}

func setUSBVisualState(state int) {
	usbVisualState = state
	for _, h := range []uintptr{mainHwnd, usbAllowButton, usbReadonlyButton, usbBlockButton, usbStatusHwnd, usbDetailHwnd} {
		if h != 0 {
			procInvalidateRect.Call(h, 0, 1)
		}
	}
}

func setNetVisualState(status string) {
	s := strings.ToLower(strings.TrimSpace(status))
	switch s {
	case "up", "connected":
		netVisualState = netStateUp
	case "disabled":
		netVisualState = netStateDisabled
	case "disconnected", "not present", "lowerlayerdown", "down":
		netVisualState = netStateDown
	default:
		netVisualState = netStateUnknown
	}
	if netStateHwnd != 0 {
		procInvalidateRect.Call(netStateHwnd, 0, 1)
	}
}

func paintRoundRect(hdc uintptr, r uiRect, radius int32, brush, pen uintptr) {
	oldBrush, _, _ := procSelectObject.Call(hdc, brush)
	oldPen, _, _ := procSelectObject.Call(hdc, pen)
	procRoundRect.Call(hdc, uintptr(r.left), uintptr(r.top), uintptr(r.right), uintptr(r.bottom), uintptr(radius), uintptr(radius))
	procSelectObject.Call(hdc, oldBrush)
	procSelectObject.Call(hdc, oldPen)
}

func paintDot(hdc uintptr, cx, cy, radius int32, color uintptr) {
	brush, _, _ := procCreateSolidBrush.Call(color)
	pen, _, _ := procCreatePen.Call(PS_SOLID, 1, color)
	oldBrush, _, _ := procSelectObject.Call(hdc, brush)
	oldPen, _, _ := procSelectObject.Call(hdc, pen)
	procEllipse.Call(hdc, uintptr(cx-radius), uintptr(cy-radius), uintptr(cx+radius), uintptr(cy+radius))
	procSelectObject.Call(hdc, oldBrush)
	procSelectObject.Call(hdc, oldPen)
	procDeleteObject.Call(brush)
	procDeleteObject.Call(pen)
}

func drawTextInRect(hdc uintptr, text string, r uiRect, font, color uintptr, flags uintptr) {
	procSetBkMode.Call(hdc, TRANSPARENT)
	procSetTextColor.Call(hdc, color)
	oldFont, _, _ := procSelectObject.Call(hdc, font)
	buf := syscall.StringToUTF16(text)
	if len(buf) > 1 {
		procDrawTextW.Call(hdc, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)-1), uintptr(unsafe.Pointer(&r)), flags)
	}
	procSelectObject.Call(hdc, oldFont)
}

func paintModernWindow(hwnd uintptr) {
	var ps paintStruct
	hdc, _, _ := procBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
	if hdc == 0 {
		return
	}
	defer procEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))

	var client uiRect
	procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&client)))

	procFillRect.Call(hdc, uintptr(unsafe.Pointer(&client)), brushPage)

	header := uiRect{0, 0, client.right, 94}
	procFillRect.Call(hdc, uintptr(unsafe.Pointer(&header)), brushWhite)

	// Product mark.
	paintRoundRect(hdc, uiRect{24, 21, 66, 63}, 12, brushAccent, penCard)
	drawTextInRect(hdc, "U", uiRect{24, 21, 66, 63}, logoFont, rgb(255, 255, 255), DT_CENTER|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)

	// Lightweight elevation badge. The process is already elevated before this window exists.
	paintRoundRect(hdc, uiRect{812, 27, 940, 59}, 16, brushAccentSoft, penDetail)
	paintDot(hdc, 828, 43, 4, rgb(37, 99, 235))
	drawTextInRect(hdc, "管理员运行中", uiRect{840, 27, 932, 59}, smallFont, rgb(29, 78, 216), DT_CENTER|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)

	sepBrush, _, _ := procCreateSolidBrush.Call(rgb(235, 240, 247))
	sep := uiRect{0, 93, client.right, 94}
	procFillRect.Call(hdc, uintptr(unsafe.Pointer(&sep)), sepBrush)

	// Card shadows and surfaces.
	paintRoundRect(hdc, uiRect{22, 112, client.right - 18, 300}, 18, brushShadow, penShadow)
	paintRoundRect(hdc, uiRect{20, 108, client.right - 20, 296}, 18, brushWhite, penCard)

	paintRoundRect(hdc, uiRect{22, 320, client.right - 18, 616}, 18, brushShadow, penShadow)
	paintRoundRect(hdc, uiRect{20, 316, client.right - 20, 612}, 18, brushWhite, penCard)

	// USB state panel.
	paintRoundRect(hdc, uiRect{44, 190, 468, 280}, 14, usbStateBrush(), penDetail)
	accentColor := rgb(37, 99, 235)
	switch usbVisualState {
	case usbStateAllow:
		accentColor = rgb(22, 163, 74)
	case usbStateReadOnly, usbStateCustom:
		accentColor = rgb(217, 119, 6)
	case usbStateBlock:
		accentColor = rgb(220, 38, 38)
	case usbStateManaged:
		accentColor = rgb(126, 34, 206)
	}
	accentBrush, _, _ := procCreateSolidBrush.Call(accentColor)
	stateBar := uiRect{44, 205, 48, 265}
	procFillRect.Call(hdc, uintptr(unsafe.Pointer(&stateBar)), accentBrush)
	procDeleteObject.Call(accentBrush)

	// Network information panel with four evenly separated metrics.
	paintRoundRect(hdc, uiRect{44, 449, client.right - 44, 558}, 14, brushDetail, penDetail)
	for _, x := range []int32{236, 400, 634} {
		line := uiRect{x, 463, x + 1, 516}
		procFillRect.Call(hdc, uintptr(unsafe.Pointer(&line)), sepBrush)
	}
	hline := uiRect{64, 520, client.right - 64, 521}
	procFillRect.Call(hdc, uintptr(unsafe.Pointer(&hline)), sepBrush)

	// Footer.
	footerTop := int32(620)
	footer := uiRect{0, footerTop, client.right, client.bottom}
	procFillRect.Call(hdc, uintptr(unsafe.Pointer(&footer)), brushFooter)
	footerSep := uiRect{0, footerTop, client.right, footerTop + 1}
	procFillRect.Call(hdc, uintptr(unsafe.Pointer(&footerSep)), sepBrush)
	paintDot(hdc, 32, 647, 4, rgb(34, 197, 94))
	drawTextInRect(hdc, "v"+appVersion, uiRect{client.right - 110, 629, client.right - 32, 659}, tinyFont, rgb(148, 163, 184), DT_CENTER|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)

	procDeleteObject.Call(sepBrush)
}

func buttonVisual(id uint32, pressed, hot, disabled bool) (bg, border, text uintptr) {
	if disabled {
		return rgb(241, 245, 249), rgb(226, 232, 240), rgb(100, 116, 139)
	}

	selected := (id == ID_USB_ALLOW && usbVisualState == usbStateAllow) ||
		(id == ID_USB_READONLY && usbVisualState == usbStateReadOnly) ||
		(id == ID_USB_BLOCK && usbVisualState == usbStateBlock)
	if selected {
		switch id {
		case ID_USB_ALLOW:
			if pressed {
				return rgb(21, 128, 61), rgb(21, 128, 61), rgb(255, 255, 255)
			}
			return rgb(22, 163, 74), rgb(22, 163, 74), rgb(255, 255, 255)
		case ID_USB_READONLY:
			if pressed {
				return rgb(180, 83, 9), rgb(180, 83, 9), rgb(255, 255, 255)
			}
			return rgb(217, 119, 6), rgb(217, 119, 6), rgb(255, 255, 255)
		case ID_USB_BLOCK:
			if pressed {
				return rgb(185, 28, 28), rgb(185, 28, 28), rgb(255, 255, 255)
			}
			return rgb(220, 38, 38), rgb(220, 38, 38), rgb(255, 255, 255)
		}
	}

	switch id {
	case ID_NET_ENABLE:
		if pressed {
			return rgb(29, 78, 216), rgb(29, 78, 216), rgb(255, 255, 255)
		}
		if hot {
			return rgb(29, 78, 216), rgb(29, 78, 216), rgb(255, 255, 255)
		}
		return rgb(37, 99, 235), rgb(37, 99, 235), rgb(255, 255, 255)
	case ID_USB_BLOCK, ID_NET_DISABLE:
		if pressed {
			return rgb(254, 226, 226), rgb(248, 113, 113), rgb(153, 27, 27)
		}
		if hot {
			return rgb(254, 242, 242), rgb(248, 113, 113), rgb(185, 28, 28)
		}
		return rgb(255, 249, 249), rgb(252, 165, 165), rgb(185, 28, 28)
	case ID_USB_ALLOW:
		if hot {
			return rgb(240, 253, 244), rgb(134, 239, 172), rgb(21, 128, 61)
		}
	case ID_USB_READONLY:
		if hot {
			return rgb(255, 251, 235), rgb(253, 186, 116), rgb(180, 83, 9)
		}
	case ID_NET_REFRESH:
		if hot {
			return rgb(239, 246, 255), rgb(147, 197, 253), rgb(29, 78, 216)
		}
	}
	if pressed {
		return rgb(241, 245, 249), rgb(148, 163, 184), rgb(30, 41, 59)
	}
	return rgb(255, 255, 255), rgb(203, 213, 225), rgb(51, 65, 85)
}

func buttonText(id uint32) string {
	switch id {
	case ID_USB_ALLOW:
		return "允许读写"
	case ID_USB_READONLY:
		return "设为只读"
	case ID_USB_BLOCK:
		return "禁用 USB 存储"
	case ID_NET_REFRESH:
		return "刷新列表"
	case ID_NET_ENABLE:
		return "启用选中网卡"
	case ID_NET_DISABLE:
		return "禁用选中网卡"
	}
	return ""
}

func drawModernButton(dis *drawItemStruct) {
	if dis == nil || dis.CtlType != ODT_BUTTON {
		return
	}
	pressed := dis.itemState&ODS_SELECTED != 0
	hot := dis.itemState&ODS_HOTLIGHT != 0
	disabled := dis.itemState&ODS_DISABLED != 0
	bgColor, borderColor, textColor := buttonVisual(dis.CtlID, pressed, hot, disabled)

	bg, _, _ := procCreateSolidBrush.Call(bgColor)
	pen, _, _ := procCreatePen.Call(PS_SOLID, 1, borderColor)
	oldBrush, _, _ := procSelectObject.Call(dis.hDC, bg)
	oldPen, _, _ := procSelectObject.Call(dis.hDC, pen)
	procRoundRect.Call(dis.hDC,
		uintptr(dis.rcItem.left), uintptr(dis.rcItem.top), uintptr(dis.rcItem.right), uintptr(dis.rcItem.bottom),
		12, 12)
	procSelectObject.Call(dis.hDC, oldBrush)
	procSelectObject.Call(dis.hDC, oldPen)
	procDeleteObject.Call(bg)
	procDeleteObject.Call(pen)

	drawTextInRect(dis.hDC, buttonText(dis.CtlID), dis.rcItem, buttonFont, textColor, DT_CENTER|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)
}

func adapterDotColor(status string) uintptr {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "up", "connected":
		return rgb(22, 163, 74)
	case "disabled":
		return rgb(220, 38, 38)
	case "disconnected", "not present", "lowerlayerdown", "down":
		return rgb(148, 163, 184)
	default:
		return rgb(100, 116, 139)
	}
}

func drawModernCombo(dis *drawItemStruct) {
	if dis == nil || dis.CtlType != ODT_COMBOBOX {
		return
	}

	isEdit := dis.itemState&ODS_COMBOBOXEDIT != 0
	selected := dis.itemState&ODS_SELECTED != 0 && !isEdit
	bgColor := rgb(255, 255, 255)
	if selected {
		bgColor = rgb(239, 246, 255)
	}
	bg, _, _ := procCreateSolidBrush.Call(bgColor)
	procFillRect.Call(dis.hDC, uintptr(unsafe.Pointer(&dis.rcItem)), bg)
	procDeleteObject.Call(bg)

	if dis.itemID == ^uint32(0) || int(dis.itemID) >= len(adapters) {
		return
	}
	a := adapters[int(dis.itemID)]

	centerY := (dis.rcItem.top + dis.rcItem.bottom) / 2
	paintDot(dis.hDC, dis.rcItem.left+16, centerY, 4, adapterDotColor(a.Status))

	nameRect := uiRect{dis.rcItem.left + 30, dis.rcItem.top, dis.rcItem.right - 112, dis.rcItem.bottom}
	label := fmt.Sprintf("%s   ·   接口 %d", a.Name, a.IfIndex)
	drawTextInRect(dis.hDC, label, nameRect, bodyFont, rgb(30, 41, 59), DT_LEFT|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)

	if a.IsActive {
		routeRect := uiRect{dis.rcItem.right - 108, dis.rcItem.top, dis.rcItem.right - 10, dis.rcItem.bottom}
		drawTextInRect(dis.hDC, "正在使用", routeRect, tinyFont, rgb(22, 163, 74), DT_CENTER|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)
	} else if a.IsDefault {
		routeRect := uiRect{dis.rcItem.right - 108, dis.rcItem.top, dis.rcItem.right - 10, dis.rcItem.bottom}
		drawTextInRect(dis.hDC, "默认路由", routeRect, tinyFont, rgb(37, 99, 235), DT_CENTER|DT_VCENTER|DT_SINGLELINE|DT_NOPREFIX)
	}

	if !isEdit {
		lineBrush, _, _ := procCreateSolidBrush.Call(rgb(241, 245, 249))
		line := uiRect{dis.rcItem.left + 12, dis.rcItem.bottom - 1, dis.rcItem.right - 8, dis.rcItem.bottom}
		procFillRect.Call(dis.hDC, uintptr(unsafe.Pointer(&line)), lineBrush)
		procDeleteObject.Call(lineBrush)
	}
}

func usbStateTextColor() uintptr {
	switch usbVisualState {
	case usbStateAllow:
		return rgb(21, 128, 61)
	case usbStateReadOnly:
		return rgb(180, 83, 9)
	case usbStateBlock:
		return rgb(185, 28, 28)
	case usbStateManaged:
		return rgb(126, 34, 206)
	case usbStateCustom:
		return rgb(180, 83, 9)
	default:
		return rgb(71, 85, 105)
	}
}

func netStateTextColor() uintptr {
	switch netVisualState {
	case netStateUp:
		return rgb(21, 128, 61)
	case netStateDisabled:
		return rgb(185, 28, 28)
	case netStateDown:
		return rgb(100, 116, 139)
	default:
		return rgb(51, 65, 85)
	}
}

func staticBrushAndColor(hwndCtrl uintptr) (brush, color uintptr) {
	id, _, _ := procGetDlgCtrlID.Call(hwndCtrl)
	switch int(id) {
	case ID_STATUS:
		return brushFooter, rgb(100, 116, 139)
	case ID_USB_STATUS:
		return usbStateBrush(), usbStateTextColor()
	case ID_USB_DETAIL, ID_USB_STATE_LABEL:
		return usbStateBrush(), rgb(100, 116, 139)
	case ID_NET_STATE:
		return brushDetail, netStateTextColor()
	case ID_NET_SPEED, ID_NET_MAC, ID_NET_ROUTE:
		return brushDetail, rgb(30, 41, 59)
	case ID_NET_DESC:
		return brushDetail, rgb(100, 116, 139)
	case ID_NET_STATE_LABEL, ID_NET_SPEED_LABEL, ID_NET_MAC_LABEL, ID_NET_ROUTE_LABEL:
		return brushDetail, rgb(148, 163, 184)
	case ID_HEADER_SUBTITLE, ID_USB_HELPER, ID_NET_HELPER, ID_NET_HINT:
		return brushWhite, rgb(100, 116, 139)
	default:
		return brushWhite, rgb(15, 23, 42)
	}
}

func modernWndProc(hwnd uintptr, message uint32, wParam, lParam uintptr) (bool, uintptr) {
	switch message {
	case WM_CREATE:
		createModernUI(hwnd)
		return true, 0
	case WM_PAINT:
		paintModernWindow(hwnd)
		return true, 0
	case WM_ERASEBKGND:
		return true, 1
	case WM_DRAWITEM:
		dis := (*drawItemStruct)(unsafe.Pointer(lParam))
		if dis == nil {
			return false, 0
		}
		switch dis.CtlType {
		case ODT_BUTTON:
			drawModernButton(dis)
			return true, 1
		case ODT_COMBOBOX:
			drawModernCombo(dis)
			return true, 1
		}
		return false, 0
	case WM_CTLCOLORSTATIC:
		hdc := wParam
		brush, color := staticBrushAndColor(lParam)
		procSetBkMode.Call(hdc, TRANSPARENT)
		procSetTextColor.Call(hdc, color)
		return true, brush
	}
	return false, 0
}
