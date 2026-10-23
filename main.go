//go:build windows

// OPPO Dolby Vision MKV Repair — single-file, native Windows GUI (Go stdlib only).
// Build: GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="-H=windowsgui -s -w" -o OPPO_DV_Repair.exe .
package main

import (
	"bufio"
	"bytes"
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image/png"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"
)

//go:embed icon.ico
var appIcon []byte

//go:embed ui_sprites.png
var uiSprites []byte

const (
	WM_PAINT          = 0x000F
	WM_ERASEBKGND     = 0x0014
	WM_TIMER          = 0x0113
	WM_SETICON        = 0x0080
	WM_CTLCOLORSTATIC = 0x0138
	WM_CTLCOLOREDIT   = 0x0133
	WM_CTLCOLORBTN    = 0x0135
	DT_CENTER         = 0x00000001
	DT_VCENTER        = 0x00000004
	DT_SINGLELINE     = 0x00000020
	title             = "OPPO Dolby Vision MKV Repair 0.1BETA"
	WM_DESTROY        = 0x0002
	WM_CLOSE          = 0x0010
	WM_COMMAND        = 0x0111
	WM_LBUTTONUP      = 0x0202
	WM_LBUTTONDOWN    = 0x0201
	WM_ACTIVATE       = 0x0006
	WM_KILLFOCUS      = 0x0008
	WA_INACTIVE       = 0
	WS_POPUP          = 0x80000000
	WS_EX_TOOLWINDOW  = 0x00000080
	SW_SHOWNORMAL     = 1
	SWP_SHOWWINDOW    = 0x0040
	SWP_NOZORDER      = 0x0004
	SWP_NOACTIVATE    = 0x0010
	WM_MOUSEMOVE      = 0x0200
	WM_KEYDOWN        = 0x0100
	SW_HIDE           = 0
	DT_LEFT           = 0
	SRCCOPY           = 0x00CC0020
	WM_SETFONT        = 0x0030
	WM_APP            = 0x8000
	WM_DROPFILES      = 0x0233
	WM_DPICHANGED     = 0x02E0
	ID_LANGUAGE       = 118
	CBS_DROPDOWNLIST  = 0x0003
	CB_ADDSTRING      = 0x0143
	CB_GETCURSEL      = 0x0147
	CB_SETCURSEL      = 0x014E
	CBN_SELCHANGE     = 1
	ID_PATH_ADD       = 116
	ID_OUTPUT_BROWSE  = 117
	EN_KILLFOCUS      = 0x0200
	WM_DISPLAYCHANGE  = 0x007E
	// Fixed-size top-level window: title bar + close/minimize, no sizing frame or maximize.
	WS_FIXEDWINDOW       = 0x00CA0000
	WS_VISIBLE           = 0x10000000
	WS_CHILD             = 0x40000000
	WS_TABSTOP           = 0x00010000
	WS_BORDER            = 0x00800000
	WS_VSCROLL           = 0x00200000
	ES_READONLY          = 0x800
	ES_MULTILINE         = 0x4
	ES_AUTOVSCROLL       = 0x40
	WS_EX_CLIENTEDGE     = 0x200
	EM_SETSEL            = 0xB1
	EM_REPLACESEL        = 0xC2
	EM_SCROLLCARET       = 0xB7
	SW_SHOW              = 5
	OFN_EXPLORER         = 0x00080000
	OFN_ALLOWMULTISELECT = 0x00000200
	OFN_FILEMUSTEXIST    = 0x1000
	OFN_PATHMUSTEXIST    = 0x800
	OFN_NOCHANGEDIR      = 0x8
	ID_ADD               = 101
	ID_CLEAR             = 102
	ID_REPAIR            = 103
	ID_EXTRACT           = 104
	ID_INSPECT           = 105
	ID_CANCEL            = 106
	ID_FFMPEG_BROWSE     = 111
	ID_FFPROBE_BROWSE    = 112
	ID_MKV_BROWSE        = 113
	ID_EXTRACT_BROWSE    = 114
)

var (
	user32                  = syscall.NewLazyDLL("user32.dll")
	kernel32                = syscall.NewLazyDLL("kernel32.dll")
	gdi32                   = syscall.NewLazyDLL("gdi32.dll")
	comdlg32                = syscall.NewLazyDLL("comdlg32.dll")
	pCreateMenu             = user32.NewProc("CreatePopupMenu")
	pAppendMenu             = user32.NewProc("AppendMenuW")
	pTrackMenu              = user32.NewProc("TrackPopupMenuEx")
	pDestroyMenu            = user32.NewProc("DestroyMenu")
	pRegister               = user32.NewProc("RegisterClassExW")
	pCreate                 = user32.NewProc("CreateWindowExW")
	pDefProc                = user32.NewProc("DefWindowProcW")
	pGetMessage             = user32.NewProc("GetMessageW")
	pTranslate              = user32.NewProc("TranslateMessage")
	pDispatch               = user32.NewProc("DispatchMessageW")
	pShow                   = user32.NewProc("ShowWindow")
	pUpdate                 = user32.NewProc("UpdateWindow")
	pSend                   = user32.NewProc("SendMessageW")
	pPost                   = user32.NewProc("PostMessageW")
	pSetText                = user32.NewProc("SetWindowTextW")
	pGetText                = user32.NewProc("GetWindowTextW")
	pGetTextLen             = user32.NewProc("GetWindowTextLengthW")
	pEnable                 = user32.NewProc("EnableWindow")
	pDestroy                = user32.NewProc("DestroyWindow")
	pPostQuit               = user32.NewProc("PostQuitMessage")
	pMessageBox             = user32.NewProc("MessageBoxW")
	pGetModule              = kernel32.NewProc("GetModuleHandleW")
	pDiskFree               = kernel32.NewProc("GetDiskFreeSpaceExW")
	pGetOpen                = comdlg32.NewProc("GetOpenFileNameW")
	pFont                   = gdi32.NewProc("CreateFontW")
	pDpi                    = user32.NewProc("SetProcessDPIAware")
	pDpiCtx                 = user32.NewProc("SetProcessDpiAwarenessContext")
	pGetDpi                 = user32.NewProc("GetDpiForSystem")
	pGetMetric              = user32.NewProc("GetSystemMetrics")
	pGetRect                = user32.NewProc("GetWindowRect")
	pFromRect               = user32.NewProc("MonitorFromRect")
	pMonitorInfo            = user32.NewProc("GetMonitorInfoW")
	pMove                   = user32.NewProc("MoveWindow")
	pDeleteFont             = gdi32.NewProc("DeleteObject")
	pBeginPaint             = user32.NewProc("BeginPaint")
	pEndPaint               = user32.NewProc("EndPaint")
	pInvalidate             = user32.NewProc("InvalidateRect")
	pSetTimer               = user32.NewProc("SetTimer")
	pKillTimer              = user32.NewProc("KillTimer")
	pFillRect               = user32.NewProc("FillRect")
	pCreateBrush            = gdi32.NewProc("CreateSolidBrush")
	pSetBkMode              = gdi32.NewProc("SetBkMode")
	pSetTextColor           = gdi32.NewProc("SetTextColor")
	pSetBkColor             = gdi32.NewProc("SetBkColor")
	pDrawText               = user32.NewProc("DrawTextW")
	pStretchDIBits          = gdi32.NewProc("StretchDIBits")
	pGetClientRect          = user32.NewProc("GetClientRect")
	pSetWindowPos           = user32.NewProc("SetWindowPos")
	pSetFocus               = user32.NewProc("SetFocus")
	pSetForeground          = user32.NewProc("SetForegroundWindow")
	pSelectObject           = gdi32.NewProc("SelectObject")
	pCreateCompatibleDC     = gdi32.NewProc("CreateCompatibleDC")
	pCreateCompatibleBitmap = gdi32.NewProc("CreateCompatibleBitmap")
	pDeleteDC               = gdi32.NewProc("DeleteDC")
	pBitBlt                 = gdi32.NewProc("BitBlt")
	pIconFromResource       = user32.NewProc("CreateIconFromResourceEx")
	shell32                 = syscall.NewLazyDLL("shell32.dll")
	pAcceptDrops            = shell32.NewProc("DragAcceptFiles")
	pDragQuery              = shell32.NewProc("DragQueryFileW")
	pDragFinish             = shell32.NewProc("DragFinish")
	app                     GUI
)

type PAINTSTRUCT struct {
	Hdc       uintptr
	Erase     int32
	Paint     RECT
	Restore   int32
	IncUpdate int32
	Reserved  [32]byte
}

func rgb(r, g, b byte) uintptr { return uintptr(uint32(r) | uint32(g)<<8 | uint32(b)<<16) }
func fill(hdc uintptr, r RECT, color uintptr) {
	brush, _, _ := pCreateBrush.Call(color)
	pFillRect.Call(hdc, uintptr(unsafe.Pointer(&r)), brush)
	pDeleteFont.Call(brush)
}
func paintProgress(hwnd uintptr) {
	var ps PAINTSTRUCT
	hdc, _, _ := pBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
	if hdc == 0 {
		return
	}
	defer pEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
	w := int32(math.Round(895 * app.scale))
	h := int32(math.Round(39 * app.scale))
	fill(hdc, RECT{0, 0, w, h}, rgb(19, 28, 40))
	margin := int32(math.Max(2, math.Round(3*app.scale)))
	fill(hdc, RECT{margin, margin, w - margin, h - margin}, rgb(40, 52, 65))
	progress := atomic.LoadInt32(&app.progress)
	if progress < 0 {
		progress = 0
	}
	if progress > 100 {
		progress = 100
	}
	inner := w - 2*margin
	width := int32(int64(inner) * int64(progress) / 100)
	block := int32(math.Max(3, math.Round(8*app.scale)))
	for x := int32(0); x < width; x += block {
		t := float64(x) / math.Max(1, float64(inner))
		var r, g, b byte
		if t < 0.5 {
			u := t * 2
			r = byte(242 - uint8(u*174))
			g = byte(190 + uint8(u*29))
			b = byte(63 + uint8(u*67))
		} else {
			u := (t - 0.5) * 2
			r = byte(68 - uint8(u*9))
			g = byte(219 - uint8(u*73))
			b = byte(130 + uint8(u*100))
		}
		right := x + block - 1
		if right > width {
			right = width
		}
		fill(hdc, RECT{margin + x, margin, margin + right, h - margin}, rgb(r, g, b))
	}
	pSetBkMode.Call(hdc, 1)
	pSetTextColor.Call(hdc, rgb(255, 255, 255))
	if app.font != 0 {
		old, _, _ := pSelectObject.Call(hdc, app.font)
		defer pSelectObject.Call(hdc, old)
	}
	label := fmt.Sprintf("%d%%", progress)
	rc := RECT{0, 0, w, h}
	pDrawText.Call(hdc, uintptr(unsafe.Pointer(wide(label))), ^uintptr(0), uintptr(unsafe.Pointer(&rc)), DT_CENTER|DT_VCENTER|DT_SINGLELINE)
}

// Embedded, hand-drawn PNG atlas; each country has actual flag pixel art.
// Sprite 0 is the globe (32x32), sprites 1-12 are national flags (48x30).
type sprite struct {
	w, h int32
	bgra []byte
}

var flagSprites []sprite

type BITMAPINFOHEADER struct {
	Size                         uint32
	Width, Height                int32
	Planes, BitCount             uint16
	Compression, SizeImage       uint32
	XPelsPerMeter, YPelsPerMeter int32
	ClrUsed, ClrImportant        uint32
}

func initFlagSprites() {
	img, err := png.Decode(bytes.NewReader(uiSprites))
	if err != nil {
		return
	}
	flagSprites = make([]sprite, 13)
	for k := 0; k < 13; k++ {
		w, h := 48, 30
		ox, oy := 8, k*40+5
		if k == 0 {
			w, h = 32, 32
			ox, oy = 16, 4
		}
		pixels := make([]byte, w*h*4)
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				r, g, b, a := img.At(ox+x, oy+y).RGBA()
				j := (y*w + x) * 4
				// Alpha blend against same dark fill as the popup/globe button.
				alpha := float64(a) / 65535
				pixels[j+0] = byte(float64(b>>8)*alpha + float64(25)*(1-alpha))
				pixels[j+1] = byte(float64(g>>8)*alpha + float64(34)*(1-alpha))
				pixels[j+2] = byte(float64(r>>8)*alpha + float64(47)*(1-alpha))
				pixels[j+3] = 0
			}
		}
		flagSprites[k] = sprite{int32(w), int32(h), pixels}
	}
}
func spriteDraw(hdc uintptr, index int, x, y, w, h int32) {
	if index < 0 || index >= len(flagSprites) {
		return
	}
	sp := flagSprites[index]
	if len(sp.bgra) == 0 {
		return
	}
	hdr := BITMAPINFOHEADER{Size: 40, Width: sp.w, Height: -sp.h, Planes: 1, BitCount: 32}
	pStretchDIBits.Call(hdc, uintptr(x), uintptr(y), uintptr(w), uintptr(h), 0, 0, uintptr(sp.w), uintptr(sp.h), uintptr(unsafe.Pointer(&sp.bgra[0])), uintptr(unsafe.Pointer(&hdr)), 0, SRCCOPY)
	runtime.KeepAlive(sp.bgra)
}
func paintGlobe(hwnd uintptr) {
	var ps PAINTSTRUCT
	hdc, _, _ := pBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
	if hdc == 0 {
		return
	}
	defer pEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
	var client RECT
	if ok, _, _ := pGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&client))); ok == 0 {
		return
	}
	w, h := client.Right-client.Left, client.Bottom-client.Top
	pad, ico := globeGeometry(w, h, app.scale)
	if ico <= 0 {
		return
	}
	fill(hdc, RECT{0, 0, w, h}, rgb(90, 125, 141))
	fill(hdc, RECT{pad, pad, w - pad, h - pad}, rgb(25, 34, 47))
	offset := int32(0)
	if app.globePressed {
		offset = int32(math.Max(1, math.Round(float64(app.globePhase)*app.scale)))
	}
	if offset > 0 {
		fill(hdc, RECT{pad, pad, w - pad, h - pad}, rgb(17, 26, 36))
		fill(hdc, RECT{pad + 1, pad + offset, w - pad - 1, h - pad}, rgb(25, 34, 47))
	}
	spriteDraw(hdc, 0, (w-ico)/2, (h-ico)/2+offset, ico, ico)
}
func paintLangPopup(hwnd uintptr) {
	var ps PAINTSTRUCT
	hdc, _, _ := pBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
	if hdc == 0 {
		return
	}
	defer pEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
	var rc RECT
	pGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&rc)))
	w, h := rc.Right, rc.Bottom
	memdc, _, _ := pCreateCompatibleDC.Call(hdc)
	if memdc == 0 {
		return
	}
	bmp, _, _ := pCreateCompatibleBitmap.Call(hdc, uintptr(w), uintptr(h))
	if bmp == 0 {
		pDeleteDC.Call(memdc)
		return
	}
	oldBmp, _, _ := pSelectObject.Call(memdc, bmp)
	defer func() { pSelectObject.Call(memdc, oldBmp); pDeleteFont.Call(bmp); pDeleteDC.Call(memdc) }()
	paintDC := memdc
	fill(paintDC, RECT{0, 0, w, h}, rgb(119, 143, 158))
	edge := int32(math.Max(1, math.Round(app.scale)))
	fill(paintDC, RECT{edge, edge, w - edge, h - edge}, rgb(25, 34, 47))
	if app.font != 0 {
		old, _, _ := pSelectObject.Call(paintDC, app.font)
		defer pSelectObject.Call(paintDC, old)
	}
	for i, item := range languageList {
		y := int32(math.Round(float64(8+i*28) * app.scale))
		rh := int32(math.Round(28 * app.scale))
		if app.menuHover == i {
			fill(paintDC, RECT{edge, y, w - edge, y + rh}, rgb(43, 69, 92))
		}
		if app.language == item.code {
			selW := int32(math.Round(3 * app.scale))
			fill(paintDC, RECT{edge + 2, y + 2, edge + 2 + selW, y + rh - 2}, rgb(77, 217, 146))
		}
		fx := int32(math.Round(12 * app.scale))
		fw := int32(math.Round(39 * app.scale))
		fh := int32(math.Round(24 * app.scale))
		spriteDraw(paintDC, i+1, fx, y+(rh-fh)/2, fw, fh)
		pSetBkMode.Call(paintDC, 1)
		pSetTextColor.Call(paintDC, rgb(231, 239, 247))
		tx := int32(math.Round(60 * app.scale))
		textRect := RECT{tx, y, w - int32(5*app.scale), y + rh}
		pDrawText.Call(paintDC, uintptr(unsafe.Pointer(wide(item.name))), ^uintptr(0), uintptr(unsafe.Pointer(&textRect)), DT_VCENTER|DT_SINGLELINE)
	}
	pBitBlt.Call(hdc, 0, 0, uintptr(w), uintptr(h), memdc, 0, 0, SRCCOPY)
}
func (g *GUI) setProgress(v int) {
	if atomic.LoadInt32(&g.progressBlocked) != 0 && v != 0 {
		return
	}
	if v < 0 {
		v = 0
	}
	if v > 100 {
		v = 100
	}
	atomic.StoreInt32(&g.progress, int32(v))
	if atomic.LoadInt32(&g.progressBlocked) != 0 {
		atomic.StoreInt32(&g.progress, 0)
	}
	g.enqueue(func() {
		if g.progressHwnd != 0 {
			pInvalidate.Call(g.progressHwnd, 0, 0)
		}
	})
}
func (g *GUI) stageProgress(ctx context.Context, path string, expected int64, from, to int, stop <-chan struct{}) {
	if expected <= 0 {
		return
	}
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			st, e := os.Stat(path)
			if e != nil {
				continue
			}
			frac := math.Min(0.995, float64(st.Size())/float64(expected))
			v := from + int(float64(to-from)*frac)
			g.setProgress(v)
		}
	}
}

type POINT struct{ X, Y int32 }
type MSG struct {
	Hwnd           uintptr
	Message        uint32
	_              uint32
	WParam, LParam uintptr
	Time           uint32
	Pt             POINT
	LPrivate       uint32
	_              uint32
}
type WNDCLASSEX struct {
	Size, Style                        uint32
	WndProc                            uintptr
	ClsExtra, WndExtra                 int32
	Instance, Icon, Cursor, Background uintptr
	MenuName, ClassName                *uint16
	IconSm                             uintptr
}
type OPENFILENAME struct {
	Size                         uint32
	HwndOwner, Instance          uintptr
	Filter, CustomFilter         *uint16
	MaxCustomFilter, FilterIndex uint32
	File                         *uint16
	MaxFile                      uint32
	FileTitle                    *uint16
	MaxFileTitle                 uint32
	InitialDir, Title            *uint16
	Flags                        uint32
	FileOffset, FileExtension    uint16
	DefExt                       *uint16
	CustData, Hook               uintptr
	TemplateName                 *uint16
	Reserved                     uintptr
	ReservedValue, FlagsEx       uint32
}

// Remember tool locations even if the EXE is upgraded or moved.
// Settings are written to %LOCALAPPDATA%\OPPO_DV_Repair\settings.json.
type Settings struct {
	Tools    map[string]string `json:"tools"`
	Output   string            `json:"output"`
	Language string            `json:"language,omitempty"`
}

func settingsPath() string {
	root := os.Getenv("LOCALAPPDATA")
	if root == "" {
		if base, err := os.UserConfigDir(); err == nil {
			root = base
		}
	}
	if root == "" {
		root = os.TempDir()
	}
	return filepath.Join(root, "OPPO_DV_Repair", "settings.json")
}
func loadSettings() (Settings, error) {
	var cfg Settings
	data, err := os.ReadFile(settingsPath())
	if os.IsNotExist(err) {
		return Settings{Tools: map[string]string{}}, nil
	}
	if err != nil {
		return cfg, err
	}
	if err = json.Unmarshal(data, &cfg); err != nil {
		return Settings{Tools: map[string]string{}}, err
	}
	if cfg.Tools == nil {
		cfg.Tools = map[string]string{}
	}
	return cfg, nil
}
func (g *GUI) saveSettings() {
	if !g.initDone {
		return
	}
	cfg := Settings{Tools: make(map[string]string), Output: strings.TrimSpace(getText(g.outputEdit)), Language: g.language}
	for name, h := range g.tools {
		cfg.Tools[name] = strings.TrimSpace(getText(h))
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err == nil {
		err = os.MkdirAll(filepath.Dir(settingsPath()), 0700)
	}
	if err == nil {
		err = os.WriteFile(settingsPath(), data, 0600)
	}
	if err != nil {
		g.appendLog(pair("[设置保存失败] ", "[Could not save settings] ") + err.Error())
	}
}

type RECT struct{ Left, Top, Right, Bottom int32 }
type MONITORINFO struct {
	Size    uint32
	Monitor RECT
	Work    RECT
	Flags   uint32
}

func currentWorkRect(r RECT) (RECT, bool) {
	m, _, _ := pFromRect.Call(uintptr(unsafe.Pointer(&r)), 2) // MONITOR_DEFAULTTONEAREST
	if m == 0 {
		return RECT{}, false
	}
	info := MONITORINFO{Size: uint32(unsafe.Sizeof(MONITORINFO{}))}
	ok, _, _ := pMonitorInfo.Call(m, uintptr(unsafe.Pointer(&info)))
	return info.Work, ok != 0
}
func screenScale(w, h int, dpi uint32) float64 {
	if w <= 0 || h <= 0 {
		return 1
	}
	if dpi < 96 {
		dpi = 96
	}
	s := math.Max(float64(dpi)/96, math.Min(float64(w)/1920, float64(h)/1080))
	if s > 2.5 {
		s = 2.5
	}
	usable := math.Min((float64(w)-64)/940, (float64(h)-100)/940)
	if usable > 0 && s > usable {
		s = usable
	}
	if s < 1 {
		s = 1
	}
	return math.Round(s*20) / 20
}

type LangChoice struct{ code, flag, name string }

var languageList = []LangChoice{
	{"zh-CN", "CN", "简体中文"},
	{"zh-TW", "TW", "繁體中文"},
	{"en", "US", "English"},
	{"ja", "JP", "日本語"},
	{"ko", "KR", "한국어"},
	{"fr", "FR", "Français"},
	{"de", "DE", "Deutsch"},
	{"it", "IT", "Italiano"},
	{"es", "ES", "Español"},
	{"pt", "PT", "Português"},
	{"ru", "RU", "Русский"},
	{"ar", "SA", "العربية"},
}
var languages = map[string]map[string]string{
	"zh-CN": {"headline": "[ P5 REPAIR ]  OPPO · DOLBY VISION  v3.0", "globe": "🌐 语言", "description": "无损提取HEVC → 重建VPS/SPS/PPS → 检查Dolby Vision；原文件不覆盖", "addTitle": "① 添加片源（支持拖入MKV，或粘贴完整路径）", "add": "添加 MKV…", "paste": "添加下方路径", "clear": "清空列表", "outputTitle": "② 指定导出目录（留空 = 原 MKV 所在目录）", "folder": "选择文件夹…", "outputNote": "临时HEVC、修复版MKV、时间戳、自检报告全部写入此目录。", "toolsTitle": "③ 工具路径（优先自动检测；也可手动指定）", "browse": "浏览…", "actions": "④ 一键处理", "repair": "一键修复 + 自检", "extract": "仅导出原始 HEVC", "inspect": "只读自检", "cancel": "取消", "logTitle": "[ LOG ]  运行日志", "ready": "READY  //  请选择文件并开始处理", "settingsError": "设置读取失败", "finished": "完成：%d / %d 项。", "cancelled": "已取消。", "processing": "处理中 %d/%d：%s", "cancelling": "正在取消当前命令……"},
	"zh-TW": {"headline": "[ P5 REPAIR ]  OPPO · DOLBY VISION  v3.0", "globe": "🌐 語言", "description": "無損擷取HEVC → 重建VPS/SPS/PPS → 檢查Dolby Vision；不覆蓋原檔", "addTitle": "① 加入影片（支援拖曳MKV或貼上完整路徑）", "add": "加入 MKV…", "paste": "加入下方路徑", "clear": "清空清單", "outputTitle": "② 指定輸出資料夾（留空 = 原檔所在資料夾）", "folder": "選擇資料夾…", "outputNote": "暫存HEVC、修復MKV、時間戳記及檢查報告存放於此。", "toolsTitle": "③ 工具路徑（自動偵測，亦可手動指定）", "browse": "瀏覽…", "actions": "④ 開始處理", "repair": "一鍵修復 + 檢查", "extract": "僅匯出 HEVC", "inspect": "唯讀檢查", "cancel": "取消", "logTitle": "[ LOG ]  執行記錄", "ready": "READY  //  請選擇檔案", "settingsError": "讀取設定失敗", "finished": "完成：%d / %d 項。", "cancelled": "已取消。", "processing": "處理中 %d/%d：%s", "cancelling": "正在取消……"},
	"en":    {"headline": "[ P5 REPAIR ]  OPPO · DOLBY VISION  v3.0", "globe": "🌐 Language", "description": "Lossless HEVC extraction → VPS/SPS/PPS rebuild → Dolby Vision check; originals untouched", "addTitle": "① Add MKV files (drag & drop or paste full paths)", "add": "Add MKV…", "paste": "Add pasted path", "clear": "Clear list", "outputTitle": "② Output folder (blank = source folder)", "folder": "Browse folder…", "outputNote": "Temporary HEVC, repaired MKV, timestamps and reports are saved here.", "toolsTitle": "③ Tool paths (auto-detect or browse manually)", "browse": "Browse…", "actions": "④ Process", "repair": "Repair + verify", "extract": "Extract HEVC only", "inspect": "Inspect only", "cancel": "Cancel", "logTitle": "[ LOG ]  Activity", "ready": "READY  //  Select files to begin", "settingsError": "Could not read settings", "finished": "Completed: %d / %d files.", "cancelled": "Cancelled.", "processing": "Processing %d/%d: %s", "cancelling": "Cancelling current task…"},
	"ja":    {"headline": "[ P5 REPAIR ]  OPPO · DOLBY VISION  v3.0", "globe": "🌐 言語", "description": "HEVCを無劣化抽出 → VPS/SPS/PPSを再構築 → Dolby Visionを検査（原本は保持）", "addTitle": "① MKVを追加（ドラッグ＆ドロップ／パス貼り付け）", "add": "MKVを追加…", "paste": "パスを追加", "clear": "一覧を消去", "outputTitle": "② 出力先（空欄なら元のフォルダー）", "folder": "フォルダー…", "outputNote": "一時HEVC、修復MKV、タイムスタンプ、検査結果を保存します。", "toolsTitle": "③ ツールの場所（自動検出または手動指定）", "browse": "参照…", "actions": "④ 実行", "repair": "修復＋検証", "extract": "HEVCのみ抽出", "inspect": "検査のみ", "cancel": "キャンセル", "logTitle": "[ LOG ]  実行ログ", "ready": "READY  //  ファイルを選択してください", "settingsError": "設定の読み込みに失敗", "finished": "完了：%d / %d 件。", "cancelled": "キャンセルしました。", "processing": "処理中 %d/%d：%s", "cancelling": "処理をキャンセルしています…"},
	"ko":    {"headline": "[ P5 REPAIR ]  OPPO · DOLBY VISION  v3.0", "globe": "🌐 언어", "description": "무손실 HEVC 추출 → VPS/SPS/PPS 재구성 → Dolby Vision 검사; 원본 보존", "addTitle": "① MKV 추가 (드래그 또는 경로 붙여넣기)", "add": "MKV 추가…", "paste": "경로 추가", "clear": "목록 지우기", "outputTitle": "② 출력 폴더 (빈칸 = 원본 폴더)", "folder": "폴더 선택…", "outputNote": "임시 HEVC, 복구 MKV, 타임스탬프 및 보고서 저장 위치.", "toolsTitle": "③ 도구 경로 (자동 검색 또는 수동 지정)", "browse": "찾아보기…", "actions": "④ 실행", "repair": "복구 + 검사", "extract": "HEVC만 추출", "inspect": "검사만", "cancel": "취소", "logTitle": "[ LOG ]  실행 기록", "ready": "READY  //  파일을 선택하세요", "settingsError": "설정 읽기 실패", "finished": "완료: %d / %d개.", "cancelled": "취소됨.", "processing": "처리 중 %d/%d: %s", "cancelling": "취소 중…"},
	"fr":    {"headline": "[ P5 REPAIR ]  OPPO · DOLBY VISION  v3.0", "globe": "🌐 Langue", "description": "Extraction HEVC sans perte → reconstruction VPS/SPS/PPS → contrôle Dolby Vision", "addTitle": "① Ajouter des MKV (glisser-déposer ou coller un chemin)", "add": "Ajouter MKV…", "paste": "Ajouter chemin", "clear": "Vider la liste", "outputTitle": "② Dossier de sortie (vide = dossier source)", "folder": "Choisir dossier…", "outputNote": "HEVC temporaires, MKV réparés, horodatages et rapports ici.", "toolsTitle": "③ Chemins des outils (détection auto ou sélection)", "browse": "Parcourir…", "actions": "④ Traitement", "repair": "Réparer + vérifier", "extract": "Extraire HEVC", "inspect": "Inspecter", "cancel": "Annuler", "logTitle": "[ LOG ]  Journal", "ready": "PRÊT  //  Sélectionnez des fichiers", "settingsError": "Erreur de configuration", "finished": "Terminé : %d / %d fichiers.", "cancelled": "Annulé.", "processing": "Traitement %d/%d : %s", "cancelling": "Annulation en cours…"},
	"de":    {"headline": "[ P5 REPAIR ]  OPPO · DOLBY VISION  v3.0", "globe": "🌐 Sprache", "description": "HEVC verlustfrei extrahieren → VPS/SPS/PPS erneuern → Dolby Vision prüfen", "addTitle": "① MKV hinzufügen (ziehen oder Pfad einfügen)", "add": "MKV hinzufügen…", "paste": "Pfad hinzufügen", "clear": "Liste leeren", "outputTitle": "② Ausgabeordner (leer = Quellordner)", "folder": "Ordner wählen…", "outputNote": "Temporäre HEVC, reparierte MKV, Zeitstempel und Berichte hier.", "toolsTitle": "③ Werkzeugpfade (automatisch oder manuell)", "browse": "Durchsuchen…", "actions": "④ Verarbeitung", "repair": "Reparieren + prüfen", "extract": "Nur HEVC exportieren", "inspect": "Nur prüfen", "cancel": "Abbrechen", "logTitle": "[ LOG ]  Protokoll", "ready": "BEREIT  //  Dateien auswählen", "settingsError": "Einstellungen fehlerhaft", "finished": "Fertig: %d / %d Dateien.", "cancelled": "Abgebrochen.", "processing": "Verarbeitung %d/%d: %s", "cancelling": "Wird abgebrochen…"},
	"it":    {"headline": "[ P5 REPAIR ]  OPPO · DOLBY VISION  v3.0", "globe": "🌐 Lingua", "description": "Estrazione HEVC senza perdita → ricostruzione VPS/SPS/PPS → verifica Dolby Vision", "addTitle": "① Aggiungi MKV (trascina o incolla il percorso)", "add": "Aggiungi MKV…", "paste": "Aggiungi percorso", "clear": "Svuota elenco", "outputTitle": "② Cartella output (vuota = cartella originale)", "folder": "Scegli cartella…", "outputNote": "HEVC temporanei, MKV riparati, timestamp e rapporti qui.", "toolsTitle": "③ Percorsi strumenti (ricerca automatica o manuale)", "browse": "Sfoglia…", "actions": "④ Elaborazione", "repair": "Ripara + verifica", "extract": "Estrai solo HEVC", "inspect": "Solo verifica", "cancel": "Annulla", "logTitle": "[ LOG ]  Registro", "ready": "PRONTO  //  Seleziona i file", "settingsError": "Errore impostazioni", "finished": "Completati: %d / %d file.", "cancelled": "Annullato.", "processing": "Elaborazione %d/%d: %s", "cancelling": "Annullamento…"},
	"es":    {"headline": "[ P5 REPAIR ]  OPPO · DOLBY VISION  v3.0", "globe": "🌐 Idioma", "description": "Extraer HEVC sin pérdidas → reconstruir VPS/SPS/PPS → verificar Dolby Vision", "addTitle": "① Añadir MKV (arrastrar o pegar ruta)", "add": "Añadir MKV…", "paste": "Añadir ruta", "clear": "Vaciar lista", "outputTitle": "② Carpeta de salida (vacía = carpeta original)", "folder": "Elegir carpeta…", "outputNote": "HEVC temporal, MKV reparado, marcas de tiempo e informes aquí.", "toolsTitle": "③ Rutas de herramientas (detección automática o manual)", "browse": "Examinar…", "actions": "④ Procesar", "repair": "Reparar + verificar", "extract": "Solo extraer HEVC", "inspect": "Solo inspeccionar", "cancel": "Cancelar", "logTitle": "[ LOG ]  Registro", "ready": "LISTO  //  Selecciona archivos", "settingsError": "Error de configuración", "finished": "Completados: %d / %d archivos.", "cancelled": "Cancelado.", "processing": "Procesando %d/%d: %s", "cancelling": "Cancelando…"},
	"pt":    {"headline": "[ P5 REPAIR ]  OPPO · DOLBY VISION  v3.0", "globe": "🌐 Idioma", "description": "Extrair HEVC sem perdas → reconstruir VPS/SPS/PPS → verificar Dolby Vision", "addTitle": "① Adicionar MKV (arrastar ou colar caminho)", "add": "Adicionar MKV…", "paste": "Adicionar caminho", "clear": "Limpar lista", "outputTitle": "② Pasta de saída (vazio = pasta original)", "folder": "Escolher pasta…", "outputNote": "HEVC temporário, MKV reparado, timestamps e relatórios aqui.", "toolsTitle": "③ Caminhos das ferramentas (auto ou manual)", "browse": "Procurar…", "actions": "④ Processar", "repair": "Reparar + verificar", "extract": "Extrair HEVC", "inspect": "Apenas inspecionar", "cancel": "Cancelar", "logTitle": "[ LOG ]  Registo", "ready": "PRONTO  //  Selecione ficheiros", "settingsError": "Erro nas definições", "finished": "Concluídos: %d / %d ficheiros.", "cancelled": "Cancelado.", "processing": "A processar %d/%d: %s", "cancelling": "A cancelar…"},
	"ru":    {"headline": "[ P5 REPAIR ]  OPPO · DOLBY VISION  v3.0", "globe": "🌐 Язык", "description": "Извлечение HEVC без потерь → восстановление VPS/SPS/PPS → проверка Dolby Vision", "addTitle": "① Добавить MKV (перетащить или вставить путь)", "add": "Добавить MKV…", "paste": "Добавить путь", "clear": "Очистить список", "outputTitle": "② Папка вывода (пусто = папка исходника)", "folder": "Выбрать папку…", "outputNote": "Временный HEVC, MKV, метки времени и отчёты сохраняются здесь.", "toolsTitle": "③ Пути к утилитам (автопоиск или вручную)", "browse": "Обзор…", "actions": "④ Обработка", "repair": "Исправить + проверить", "extract": "Извлечь HEVC", "inspect": "Только проверка", "cancel": "Отмена", "logTitle": "[ LOG ]  Журнал", "ready": "ГОТОВО  //  Выберите файлы", "settingsError": "Ошибка настроек", "finished": "Завершено: %d / %d файлов.", "cancelled": "Отменено.", "processing": "Обработка %d/%d: %s", "cancelling": "Отмена операции…"},
	"ar":    {"headline": "[ P5 REPAIR ]  OPPO · DOLBY VISION  v3.0", "globe": "🌐 اللغة", "description": "استخراج HEVC دون فقد → إعادة بناء VPS/SPS/PPS → فحص Dolby Vision", "addTitle": "① إضافة ملفات MKV (سحب أو لصق المسار)", "add": "إضافة MKV…", "paste": "إضافة المسار", "clear": "مسح القائمة", "outputTitle": "② مجلد الإخراج (فارغ = مجلد المصدر)", "folder": "اختيار المجلد…", "outputNote": "يُحفظ هنا HEVC المؤقت وMKV المُصلح والتوقيتات والتقارير.", "toolsTitle": "③ مسارات الأدوات (اكتشاف تلقائي أو تحديد يدوي)", "browse": "استعراض…", "actions": "④ المعالجة", "repair": "إصلاح + فحص", "extract": "استخراج HEVC", "inspect": "فحص فقط", "cancel": "إلغاء", "logTitle": "[ LOG ]  سجل التشغيل", "ready": "جاهز  //  اختر الملفات", "settingsError": "تعذر قراءة الإعدادات", "finished": "اكتمل: %d / %d ملفات.", "cancelled": "تم الإلغاء.", "processing": "معالجة %d/%d: %s", "cancelling": "جارٍ الإلغاء…"},
}

func tr(lang, key string) string {
	if m, ok := languages[lang]; ok {
		if v, ok := m[key]; ok {
			return v
		}
	}
	if v, ok := languages["en"][key]; ok {
		return v
	}
	return key
}

type LayoutItem struct {
	hwnd       uintptr
	x, y, w, h int
}
type GUI struct {
	arrowCursor                    uintptr
	logRecords                     []logRecord
	statusRaw                      string
	lastResults                    []BatchResult
	progressBlocked                int32
	closeAfterCancel               bool
	scale                          float64
	automatic                      float64
	layouts                        []LayoutItem
	pathEdit, outputEdit           uintptr
	initDone                       bool
	hwnd                           uintptr
	instance                       uintptr
	btn                            map[int]uintptr
	tools                          map[string]uintptr
	files                          []string
	filesEdit, logEdit, statusEdit uintptr
	progressHwnd                   uintptr
	globeHwnd, languagePopup       uintptr
	menuShown                      bool
	globePhase                     int
	globePressed                   bool
	menuHover                      int
	progress                       int32
	darkBrush                      uintptr
	editBrush                      uintptr
	font                           uintptr
	working                        bool
	cancel                         context.CancelFunc
	pending                        []func()
	pendingMu                      sync.Mutex
	language                       string
	labels                         map[string]uintptr
}

func wide(s string) *uint16        { p, _ := syscall.UTF16PtrFromString(s); return p }
func pix(s float64, v int) uintptr { return uintptr(int(math.Round(s * float64(v)))) }
func (g *GUI) widget(class, text string, style uint32, x, y, w, h int, id int) uintptr {
	r, _, _ := pCreate.Call(0, uintptr(unsafe.Pointer(wide(class))), uintptr(unsafe.Pointer(wide(text))), uintptr(style|WS_CHILD|WS_VISIBLE), pix(g.scale, x), pix(g.scale, y), pix(g.scale, w), pix(g.scale, h), g.hwnd, uintptr(id), g.instance, 0)
	if r != 0 {
		g.layouts = append(g.layouts, LayoutItem{r, x, y, w, h})
		if g.font != 0 {
			pSend.Call(r, WM_SETFONT, g.font, 1)
		}
	}
	return r
}
func (g *GUI) label(text string, x, y, w, h int) { g.widget("STATIC", text, 0, x, y, w, h, 0) }
func (g *GUI) trLabel(key string, x, y, w, h int) {
	g.labels[key] = g.widget("STATIC", tr(g.language, key), 0, x, y, w, h, 0)
}
func (g *GUI) trButton(key string, id, x, y, w, h int) { g.button(tr(g.language, key), id, x, y, w, h) }
func (g *GUI) button(text string, id, x, y, w, h int) {
	g.btn[id] = g.widget("BUTTON", text, WS_TABSTOP, x, y, w, h, id)
}
func (g *GUI) edit(text string, readOnly bool, x, y, w, h int) uintptr {
	s := uint32(WS_BORDER | WS_TABSTOP)
	if readOnly {
		s |= ES_READONLY
	}
	return g.widget("EDIT", text, s, x, y, w, h, 0)
}
func (g *GUI) multiline(x, y, w, h int) uintptr {
	return g.widget("EDIT", "", uint32(WS_BORDER|WS_VSCROLL|ES_MULTILINE|ES_AUTOVSCROLL|ES_READONLY), x, y, w, h, 0)
}
func setText(hwnd uintptr, s string) { pSetText.Call(hwnd, uintptr(unsafe.Pointer(wide(s)))) }
func getText(hwnd uintptr) string {
	n, _, _ := pGetTextLen.Call(hwnd)
	b := make([]uint16, n+2)
	pGetText.Call(hwnd, uintptr(unsafe.Pointer(&b[0])), uintptr(len(b)))
	return syscall.UTF16ToString(b)
}
func (g *GUI) appendLog(s string) { g.appendLocalizedLog(s) }
func (g *GUI) enqueue(fn func()) {
	g.pendingMu.Lock()
	g.pending = append(g.pending, fn)
	g.pendingMu.Unlock()
	pPost.Call(g.hwnd, WM_APP+1, 0, 0)
}
func (g *GUI) flush() {
	g.pendingMu.Lock()
	todo := g.pending
	g.pending = nil
	g.pendingMu.Unlock()
	for _, fn := range todo {
		fn()
	}
}
func (g *GUI) log(s string)    { g.enqueue(func() { g.appendLog(s) }) }
func (g *GUI) status(s string) { g.enqueue(func() { g.setStatusNow(s) }) }
func (g *GUI) busy(b bool) {
	g.working = b
	ids := []int{ID_ADD, ID_PATH_ADD, ID_CLEAR, ID_REPAIR, ID_EXTRACT, ID_INSPECT, ID_FFMPEG_BROWSE, ID_FFPROBE_BROWSE, ID_MKV_BROWSE, ID_EXTRACT_BROWSE, ID_OUTPUT_BROWSE}
	for _, id := range ids {
		v := uintptr(1)
		if b {
			v = 0
		}
		pEnable.Call(g.btn[id], v)
	}
	v := uintptr(0)
	if b {
		v = 1
	}
	pEnable.Call(g.btn[ID_CANCEL], v)
}
func (g *GUI) refreshFiles() {
	if len(g.files) == 0 {
		setText(g.filesEdit, tr(g.language, "empty"))
		return
	}
	var items []string
	for i, f := range g.files {
		items = append(items, fmt.Sprintf("%d. %s", i+1, f))
	}
	setText(g.filesEdit, strings.Join(items, "\r\n"))
}

// Run the OS picker out-of-process. A former GetOpenFileNameW call directly
// inside the Win32 WndProc callback could terminate the whole GUI on some PCs.
// PowerShell 5.1 and Windows Forms are part of standard Windows 10/11.
// Return paths encoded as base64 UTF-8 lines, so Chinese/Japanese paths are safe.
func chooseFiles(exe, multi bool) ([]string, error) {
	filter := "MKV 文件 (*.mkv)|*.mkv|所有文件 (*.*)|*.*"
	heading := "选择 MKV 文件（可多选）"
	if exe {
		filter = "EXE 程序 (*.exe)|*.exe|所有文件 (*.*)|*.*"
		heading = "选择工具程序 EXE"
	}
	multiValue := "$true"
	if !multi {
		multiValue = "$false"
	}
	script := fmt.Sprintf(`
$ErrorActionPreference = 'Stop'
try {
  Add-Type -AssemblyName System.Windows.Forms
  [System.Windows.Forms.Application]::EnableVisualStyles()
  $picker = New-Object System.Windows.Forms.OpenFileDialog
  $picker.Title = '%s'
  $picker.Filter = '%s'
  $picker.Multiselect = %s
  $picker.CheckFileExists = $true
  $picker.RestoreDirectory = $true
  $owner = New-Object System.Windows.Forms.Form
  $owner.Text = 'OPPO DV Repair - File Picker'
  $owner.Size = New-Object System.Drawing.Size(1, 1)
  $owner.StartPosition = [System.Windows.Forms.FormStartPosition]::CenterScreen
  $owner.ShowInTaskbar = $false
  $owner.TopMost = $true
  $owner.Opacity = 0
  $owner.Show()
  $picked = $picker.ShowDialog($owner)
  $owner.Close()
  $owner.Dispose()
  if ($picked -eq [System.Windows.Forms.DialogResult]::OK) {
    foreach ($item in @($picker.FileNames)) {
      [Console]::WriteLine([Convert]::ToBase64String([System.Text.Encoding]::UTF8.GetBytes($item)))
    }
  }
  $picker.Dispose()
} catch {
  [Console]::Error.WriteLine($_.Exception.ToString())
  exit 2
}`, heading, filter, multiValue)
	// -EncodedCommand is UTF-16LE, avoids quote escaping and execution policy.
	utf16script := syscall.StringToUTF16(script)
	raw := make([]byte, len(utf16script)*2)
	for i, v := range utf16script {
		binary.LittleEndian.PutUint16(raw[i*2:], v)
	}
	if len(raw) >= 2 {
		raw = raw[:len(raw)-2]
	}
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-STA", "-EncodedCommand", base64.StdEncoding.EncodeToString(raw))
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	stdout, err := cmd.Output()
	if err != nil {
		reason := err.Error()
		if ee, ok := err.(*exec.ExitError); ok {
			reason += " / " + string(ee.Stderr)
		}
		return nil, fmt.Errorf("文件选择器启动失败：%s", truncate(reason, 400))
	}
	var results []string
	for _, line := range strings.Split(strings.TrimSpace(string(stdout)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		b, e := base64.StdEncoding.DecodeString(line)
		if e != nil {
			return nil, fmt.Errorf("文件选择器返回了无效内容：%w", e)
		}
		candidate := string(b)
		if candidate != "" {
			results = append(results, candidate)
		}
	}
	return results, nil
}

// Folder browser runs out-of-process, just like MKV file selection. A failure
// cannot bring down the Win32 message callback, and users can paste a path.
func chooseFolder(initial string) (string, error) {
	initialB64 := base64.StdEncoding.EncodeToString([]byte(initial))
	script := fmt.Sprintf(`
$ErrorActionPreference = 'Stop'
try {
  Add-Type -AssemblyName System.Windows.Forms
  [System.Windows.Forms.Application]::EnableVisualStyles()
  $picker = New-Object System.Windows.Forms.FolderBrowserDialog
  $picker.Description = '选择修复输出目录（临时HEVC和最终MKV都保存在此）'
  $picker.ShowNewFolderButton = $true
  $start = [System.Text.Encoding]::UTF8.GetString([Convert]::FromBase64String('%s'))
  if ($start -and [System.IO.Directory]::Exists($start)) { $picker.SelectedPath = $start }
  $owner = New-Object System.Windows.Forms.Form
  $owner.Text = 'OPPO DV Repair - Output Folder'
  $owner.Size = New-Object System.Drawing.Size(1, 1)
  $owner.StartPosition = [System.Windows.Forms.FormStartPosition]::CenterScreen
  $owner.ShowInTaskbar = $false
  $owner.TopMost = $true
  $owner.Opacity = 0
  $owner.Show()
  $picked = $picker.ShowDialog($owner)
  $owner.Close()
  $owner.Dispose()
  if ($picked -eq [System.Windows.Forms.DialogResult]::OK) {
    [Console]::WriteLine([Convert]::ToBase64String([System.Text.Encoding]::UTF8.GetBytes($picker.SelectedPath)))
  }
  $picker.Dispose()
} catch {
  [Console]::Error.WriteLine($_.Exception.ToString())
  exit 2
}`, initialB64)
	u := syscall.StringToUTF16(script)
	raw := make([]byte, len(u)*2)
	for i, v := range u {
		binary.LittleEndian.PutUint16(raw[i*2:], v)
	}
	raw = raw[:len(raw)-2]
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-STA", "-EncodedCommand", base64.StdEncoding.EncodeToString(raw))
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	out, err := cmd.Output()
	if err != nil {
		reason := err.Error()
		if ee, ok := err.(*exec.ExitError); ok {
			reason += " / " + string(ee.Stderr)
		}
		return "", fmt.Errorf("目录选择器失败：%s", truncate(reason, 450))
	}
	encoded := strings.TrimSpace(string(out))
	if encoded == "" {
		return "", nil
	}
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("目录选择器返回了无效路径：%w", err)
	}
	return string(decoded), nil
}

func showErr(owner uintptr, s string) {
	pMessageBox.Call(owner, uintptr(unsafe.Pointer(wide(renderMessage(app.language, s)))), uintptr(unsafe.Pointer(wide("OPPO DV MKV Repair"))), 0x10)
}
func autoScale() float64 {
	w, _, _ := pGetMetric.Call(0)
	h, _, _ := pGetMetric.Call(1)
	dpi := uint32(96)
	if e := pGetDpi.Find(); e == nil {
		if n, _, _ := pGetDpi.Call(); n >= 96 && n <= 768 {
			dpi = uint32(n)
		}
	}
	return screenScale(int(w), int(h), dpi)
}
func makeFont(scale float64) uintptr {
	name := wide("Segoe UI")
	f, _, _ := pFont.Call(uintptr(int32(-math.Round(17*scale))), 0, 0, 0, 400, 0, 0, 0, 1, 0, 0, 0, 0, uintptr(unsafe.Pointer(name)))
	return f
}

// A DPI or display resolution change can resize the *fixed* window automatically;
// the user cannot drag the border or maximize it. Preserve the screen location.
func (g *GUI) setScale(scale float64, suggested *RECT) {
	if scale < 0.85 {
		scale = 0.85
	}
	if scale > 3 {
		scale = 3
	}
	pos := RECT{Left: 60, Top: 40, Right: 1000, Bottom: 980}
	if g.hwnd != 0 {
		pGetRect.Call(g.hwnd, uintptr(unsafe.Pointer(&pos)))
	}
	if suggested != nil {
		pos = *suggested
	}
	changed := math.Abs(g.scale-scale) >= 0.001
	if changed {
		g.scale = scale
		oldFont := g.font
		g.font = makeFont(scale)
		for _, item := range g.layouts {
			pMove.Call(item.hwnd, pix(scale, item.x), pix(scale, item.y), pix(scale, item.w), pix(scale, item.h), 1)
			pSend.Call(item.hwnd, WM_SETFONT, g.font, 1)
		}
		if oldFont != 0 {
			pDeleteFont.Call(oldFont)
		}
		refreshSponsorCaptionFont()
	}
	winW := int(math.Round(scale * 940))
	winH := int(math.Round(scale * 940))
	work, ok := currentWorkRect(pos)
	x, y := int(pos.Left), int(pos.Top)
	if ok {
		maxX := int(work.Right) - winW
		maxY := int(work.Bottom) - winH
		if x > maxX {
			x = maxX
		}
		if y > maxY {
			y = maxY
		}
		if x < int(work.Left) {
			x = int(work.Left)
		}
		if y < int(work.Top) {
			y = int(work.Top)
		}
	}
	pMove.Call(g.hwnd, uintptr(x), uintptr(y), uintptr(winW), uintptr(winH), 1)
	if changed && g.initDone {
		g.closeLanguageMenu()
		pInvalidate.Call(g.globeHwnd, 0, 0)
		pInvalidate.Call(g.languagePopup, 0, 0)
	}
}
func (g *GUI) addPaths(paths []string) {
	g.lastResults = nil
	files, issues := collectMKVs(paths, g.files)
	g.files = append(g.files, files...)
	g.refreshFiles()
	for _, issue := range issues {
		g.appendLog(issue)
	}
	g.appendLog(event("@addedFiles", strconv.Itoa(len(files)), strconv.Itoa(len(g.files))))
	if len(files) == 0 {
		g.appendLog(event("@noNewFiles"))
	}
}
func (g *GUI) setup() {
	g.font = makeFont(g.scale)
	g.btn = map[int]uintptr{}
	g.tools = map[string]uintptr{}
	g.labels = map[string]uintptr{}
	cfg, cfgErr := loadSettings()
	g.language = cfg.Language
	if _, ok := languages[g.language]; !ok {
		g.language = "zh-CN"
	}
	g.trLabel("headline", 20, 14, 570, 34)
	// Genuine pixel-art globe, not a Unicode emoji or native combo box.
	g.globeHwnd = g.widget("OPPO_P5_GLOBE", "", WS_TABSTOP|0x04000000, 872, 10, 43, 43, ID_LANGUAGE)
	// Custom drawn flag list: not dependent on Windows Emoji or system flag fonts.
	// A real OWNED TOP-LEVEL WS_POPUP window (never a child control).
	// Child menus were clipped/overpainted by EDIT controls on first launch.
	cls := wide("OPPO_P5_LANGPOP")
	g.languagePopup, _, _ = pCreate.Call(
		WS_EX_TOOLWINDOW, uintptr(unsafe.Pointer(cls)), 0,
		WS_POPUP|WS_BORDER, 0, 0, pix(g.scale, 236), pix(g.scale, 348),
		g.hwnd, 0, g.instance, 0)
	if g.languagePopup == 0 {
		showErr(g.hwnd, "无法创建独立语言菜单窗口")
	}
	pShow.Call(g.languagePopup, SW_HIDE)
	g.menuShown = false
	g.menuHover = -1
	g.trLabel("description", 20, headerDescriptionY, 895, 23)
	g.trLabel("addTitle", 20, 85, 895, 25)
	g.trButton("add", ID_ADD, 20, 116, 145, 35)
	g.trButton("paste", ID_PATH_ADD, 175, 116, 160, 35)
	g.trButton("clear", ID_CLEAR, 345, 116, 135, 35)
	g.pathEdit = g.edit("", false, 20, 160, 895, 30)
	g.filesEdit = g.multiline(20, 200, 895, 94)
	g.trLabel("outputTitle", 20, 302, 895, 25)
	g.outputEdit = g.edit(cfg.Output, false, 20, 330, 765, 32)
	g.trButton("folder", ID_OUTPUT_BROWSE, 796, 330, 119, 32)
	g.trLabel("outputNote", 20, 370, 895, 24)
	g.trLabel("toolsTitle", 20, 405, 895, 24)
	items := []struct {
		name  string
		id, y int
		label string
	}{
		{"ffmpeg", ID_FFMPEG_BROWSE, 435, "FFmpeg"},
		{"ffprobe", ID_FFPROBE_BROWSE, 472, "FFprobe"},
		{"mkvmerge", ID_MKV_BROWSE, 509, "MKVmerge"},
		{"mkvextract", ID_EXTRACT_BROWSE, 546, "MKVextract"},
	}
	for _, t := range items {
		g.label(t.label, 22, t.y+4, 108, 25)
		value := strings.TrimSpace(cfg.Tools[t.name])
		if value == "" {
			value = findTool(t.name)
		}
		g.tools[t.name] = g.edit(value, false, 130, t.y, 660, 31)
		g.trButton("browse", t.id, 800, t.y, 115, 31)
	}
	g.trLabel("actions", 20, 589, 895, 25)
	g.trButton("repair", ID_REPAIR, 20, 620, 215, 45)
	g.trButton("extract", ID_EXTRACT, 245, 620, 208, 45)
	g.trButton("inspect", ID_INSPECT, 463, 620, 163, 45)
	g.trButton("cancel", ID_CANCEL, 636, 620, 110, 45)
	setupSponsor(g)
	g.progressHwnd = g.widget("OPPO_P5_PROGRESS", "", 0, 20, 675, 895, 39, 0)
	g.statusEdit = g.widget("STATIC", tr(g.language, "ready"), 0, 20, 717, 895, 24, 0)
	g.trLabel("logTitle", 20, 751, 895, 24)
	g.logEdit = g.multiline(20, 779, 895, 111)
	g.busy(false)
	g.refreshFiles()
	g.appendLog("OPPO P5 Repair " + version + " | Windows x64")
	g.appendLog(event("@startup"))
	g.initDone = true
	pAcceptDrops.Call(g.hwnd, 1)
	if cfgErr != nil {
		g.appendLog(event("@settingsError") + ": " + cfgErr.Error())
	}
}
func (g *GUI) switchLanguage(code string) {
	if _, ok := languages[code]; !ok {
		return
	}
	g.language = code
	for key, hwnd := range g.labels {
		if key == "sponsorCaption" {
			continue
		}
		setText(hwnd, tr(code, key))
	}
	keys := map[int]string{ID_ADD: "add", ID_PATH_ADD: "paste", ID_CLEAR: "clear", ID_OUTPUT_BROWSE: "folder", ID_FFMPEG_BROWSE: "browse", ID_FFPROBE_BROWSE: "browse", ID_MKV_BROWSE: "browse", ID_EXTRACT_BROWSE: "browse", ID_REPAIR: "repair", ID_EXTRACT: "extract", ID_INSPECT: "inspect", ID_CANCEL: "cancel"}
	for id, key := range keys {
		setText(g.btn[id], tr(code, key))
	}
	if g.languagePopup != 0 {
		pInvalidate.Call(g.languagePopup, 0, 0)
	}
	g.refreshLog()
	if g.statusRaw != "" {
		g.setStatusNow(g.statusRaw)
	} else {
		g.setStatusNow(event("@ready"))
	}
	if len(g.lastResults) > 0 {
		g.renderResults(g.lastResults)
	} else {
		g.refreshFiles()
	}
	g.saveSettings()
	refreshSponsorLanguage()
}
func (g *GUI) toggleLanguageMenu() {
	if g.languagePopup == 0 {
		return
	}
	g.menuShown = !g.menuShown
	if g.menuShown {
		g.menuHover = -1
		// Position in SCREEN coordinates under the pixel globe button.
		var globe RECT
		pGetRect.Call(g.globeHwnd, uintptr(unsafe.Pointer(&globe)))
		menuW, menuH := int32(math.Round(236*g.scale)), int32(math.Round(348*g.scale))
		x, y := globe.Right-menuW, globe.Bottom+int32(math.Round(4*g.scale))
		if work, ok := currentWorkRect(globe); ok {
			if x < work.Left {
				x = work.Left
			}
			if x+menuW > work.Right {
				x = work.Right - menuW
			}
			if y+menuH > work.Bottom {
				y = globe.Top - menuH - int32(math.Round(4*g.scale))
			}
			if y < work.Top {
				y = work.Top
			}
		}
		pSetWindowPos.Call(g.languagePopup, 0, uintptr(x), uintptr(y), uintptr(menuW), uintptr(menuH), SWP_SHOWWINDOW)
		pInvalidate.Call(g.languagePopup, 0, 1)
		pUpdate.Call(g.languagePopup)
		// Popup takes focus so outside click and Esc both dismiss reliably.
		pSetForeground.Call(g.languagePopup)
		pSetFocus.Call(g.languagePopup)
	} else {
		pShow.Call(g.languagePopup, SW_HIDE)
	}
}
func (g *GUI) closeLanguageMenu() {
	if !g.menuShown {
		return
	}
	g.menuShown = false
	pShow.Call(g.languagePopup, SW_HIDE)
}

func (g *GUI) onCommand(id int) {
	if id == idSponsor {
		g.openSponsor()
		return
	}
	if g.working && id != ID_CANCEL && id != ID_LANGUAGE {
		return
	}
	switch id {
	case ID_ADD:
		g.appendLog(pair("正在打开 Windows 文件选择器……", "Opening Windows file picker…"))
		picked, e := chooseFiles(false, true)
		if e != nil {
			g.appendLog(pair("[选择器错误] ", "[File picker error] ") + e.Error())
			showErr(g.hwnd, e.Error()+"\r\n\r\n你仍可直接把MKV拖进窗口，或粘贴完整路径后点击【添加下方路径】。")
		}
		g.addPaths(picked)
	case ID_OUTPUT_BROWSE:
		initial := strings.Trim(strings.TrimSpace(getText(g.outputEdit)), "\"")
		if initial == "" && len(g.files) > 0 {
			initial = filepath.Dir(g.files[0])
		}
		selected, err := chooseFolder(initial)
		if err != nil {
			g.appendLog(pair("[目录选择器错误] ", "[Folder picker error] ") + err.Error())
			showErr(g.hwnd, err.Error()+"\r\n也可以直接在输出目录框粘贴文件夹路径。")
		} else if selected != "" {
			setText(g.outputEdit, selected)
			g.saveSettings()
			g.appendLog(pair("已选择并记住输出目录：", "Output folder saved: ") + selected)
		}
	case ID_PATH_ADD:
		raw := getText(g.pathEdit)
		// Accept line-separated absolute paths; keep spaces and Unicode intact.
		g.addPaths(strings.Split(strings.ReplaceAll(raw, "\r", ""), "\n"))
	case ID_CLEAR:
		g.lastResults = nil
		g.files = nil
		g.refreshFiles()
	case ID_REPAIR:
		g.run("repair")
	case ID_EXTRACT:
		g.run("export")
	case ID_INSPECT:
		g.run("inspect")
	case ID_CANCEL:
		g.requestCancel()
	case ID_FFMPEG_BROWSE:
		g.browseTool("ffmpeg")
	case ID_FFPROBE_BROWSE:
		g.browseTool("ffprobe")
	case ID_MKV_BROWSE:
		g.browseTool("mkvmerge")
	case ID_EXTRACT_BROWSE:
		g.browseTool("mkvextract")
	}
}
func (g *GUI) browseTool(name string) {
	p, e := chooseFiles(true, false)
	if e != nil {
		g.appendLog(pair("[工具浏览错误] ", "[Tool picker error] ") + e.Error())
		showErr(g.hwnd, e.Error())
		return
	}
	if len(p) > 0 {
		setText(g.tools[name], p[0])
		// MKVToolNix installs both programs in one directory; select one and
		// fill the companion automatically, without overwriting a valid choice.
		companion := map[string]string{"mkvmerge": "mkvextract", "mkvextract": "mkvmerge", "ffmpeg": "ffprobe", "ffprobe": "ffmpeg"}
		other := companion[name]
		candidate := filepath.Join(filepath.Dir(p[0]), other+".exe")
		if other != "" {
			if _, e := os.Stat(candidate); e == nil {
				if _, resolvedErr := resolveTool(strings.TrimSpace(getText(g.tools[other]))); resolvedErr != nil {
					setText(g.tools[other], candidate)
					g.appendLog(pair("已自动找到配套工具：", "Companion tool found: ") + candidate)
				}
			}
		}
		g.saveSettings()
		g.appendLog(pair("已记住工具路径：", "Tool path saved: ") + name + " → " + p[0])
	}
}
func (g *GUI) run(mode string) {
	g.saveSettings() // include manually pasted paths, not only Browse selections
	if len(g.files) == 0 {
		showErr(g.hwnd, tr(g.language, "needFiles"))
		return
	}
	toolpaths := map[string]string{}
	for _, name := range []string{"ffmpeg", "ffprobe", "mkvmerge", "mkvextract"} {
		if mode == "inspect" && name != "ffprobe" {
			continue
		}
		if mode == "export" && name != "ffprobe" && name != "ffmpeg" {
			continue
		}
		p := strings.TrimSpace(strings.Trim(getText(g.tools[name]), "\""))
		if p == "" {
			p = findTool(name)
		}
		resolved, err := resolveTool(p)
		if err != nil {
			if name == "ffprobe" {
				showErr(g.hwnd, fmt.Sprintf(tr(g.language, "missing"), name, err))
				return
			}
			resolved = p // validate after per-file preflight, so valid files can be skipped
		}
		toolpaths[name] = resolved
	}
	destination := strings.Trim(strings.TrimSpace(getText(g.outputEdit)), "\"")
	if destination != "" {
		absolute, err := filepath.Abs(destination)
		if err != nil {
			showErr(g.hwnd, "输出路径无效："+err.Error())
			return
		}
		destination = absolute
		st, err := os.Stat(destination)
		if err != nil || !st.IsDir() {
			showErr(g.hwnd, "输出目录不存在或不是文件夹：\r\n"+destination+"\r\n请先创建文件夹。")
			return
		}
		testFile, err := os.CreateTemp(destination, ".oppo_dv_write_check_*.tmp")
		if err != nil {
			showErr(g.hwnd, "输出目录无法写入：\r\n"+destination+"\r\n"+err.Error())
			return
		}
		_ = testFile.Close()
		_ = os.Remove(testFile.Name())
	}
	sources := append([]string(nil), g.files...)
	destination, err := createRunOutput(destination, sources, time.Now())
	if err != nil {
		showErr(g.hwnd, "无法创建结果文件夹 / Cannot create output folder: "+err.Error())
		return
	}
	g.appendLog(event("output", destination))
	ctx, cancel := context.WithCancel(context.WithValue(context.Background(), reportLanguageKey{}, g.language))
	g.cancel = cancel
	g.busy(true)
	atomic.StoreInt32(&g.progressBlocked, 0)
	g.setProgress(0)
	results := make([]BatchResult, len(sources))
	for i, p := range sources {
		results[i] = BatchResult{Source: p, Status: "notStarted"}
	}
	g.showResults(results)
	g.log(event("@estimate"))
	g.appendLog(pair(fmt.Sprintf("开始处理 %d 个 MKV（%s）", len(sources), mode), fmt.Sprintf("Starting %d MKV files (%s)", len(sources), mode)))

	go func() {
		defer cancel()
		passed, skipped, failed, cancelled := 0, 0, 0, 0
		inspectionIssues, inspectionUnsupported := 0, 0
		for i, p := range sources {
			if ctx.Err() != nil {
				break
			}
			results[i].Status = "running"
			g.showResults(results)
			g.status(event("@processing", strconv.Itoa(i+1), strconv.Itoa(len(sources)), filepath.Base(p)))
			g.log(event("processingFile", filepath.Base(p)))
			start := int(float64(i) / float64(len(sources)) * 100)
			end := int(float64(i+1) / float64(len(sources)) * 100)
			g.setProgress(start)
			onStage := func(stage string, path string, expected int64, fn func() error) error {
				var from, to int
				switch stage {
				case "extract":
					from = 5
					to = 48
				case "mux":
					from = 52
					to = 93
				default:
					from = 94
					to = 97
				}
				g.setProgress(start + (end-start)*from/100)
				stop := make(chan struct{})
				done := make(chan struct{})
				go func() {
					g.stageProgress(ctx, path, expected, start+(end-start)*from/100, start+(end-start)*to/100, stop)
					close(done)
				}()
				err := func() error { defer func() { close(stop); <-done }(); return fn() }()
				if err == nil && ctx.Err() == nil {
					g.setProgress(start + (end-start)*to/100)
				}
				return err
			}
			err := isolatedProcess(ctx, p, destination, mode, toolpaths, g.log, onStage)
			var finding *inspectionResult
			switch {
			case ctx.Err() != nil && err != nil:
				results[i].Status = "cancelledFile"
				results[i].Detail = ctx.Err().Error()
				cancelled++
			case errors.As(err, &finding):
				results[i].Status = finding.Status
				results[i].Detail = pair(finding.ZH, finding.EN)
				switch finding.Status {
				case "inspectPass":
					passed++
				case "inspectIssue":
					inspectionIssues++
				case "inspectUnsupported":
					inspectionUnsupported++
				}
			case errors.Is(err, errSkipped):
				results[i].Status = "skipped"
				results[i].Detail = err.Error()
				skipped++
			case err != nil:
				results[i].Status = "failed"
				results[i].Detail = err.Error()
				failed++
			default:
				results[i].Status = "success"
				passed++
			}
			g.log(event("outcome", event("@"+results[i].Status), filepath.Base(p), results[i].Detail))
			g.showResults(results)
			if ctx.Err() == nil {
				g.setProgress(end)
			}
		}

		wasCancelled := ctx.Err() != nil
		g.enqueue(func() {
			g.busy(false)
			if wasCancelled {
				g.resetProgressVisuals()
				g.appendLog(event("reset"))
			} else {
				g.setProgress(100)
			}
			g.cancel = nil
			msg := event("@batchSummary", strconv.Itoa(passed), strconv.Itoa(skipped), strconv.Itoa(failed), strconv.Itoa(cancelled), strconv.Itoa(len(sources)-passed-skipped-failed-cancelled-inspectionIssues-inspectionUnsupported))
			if mode == "inspect" {
				msg = event("@inspectSummary", strconv.Itoa(passed), strconv.Itoa(inspectionIssues), strconv.Itoa(inspectionUnsupported), strconv.Itoa(failed), strconv.Itoa(cancelled), strconv.Itoa(len(sources)-passed-inspectionIssues-inspectionUnsupported-failed-cancelled))
			}
			if wasCancelled {
				msg = event("@cancelled") + " " + msg
			}
			g.setStatusNow(msg)
			g.appendLog(msg)
			if g.closeAfterCancel {
				pDestroy.Call(g.hwnd)
			}
		})
	}()
}
func findTool(name string) string {
	exe := name + ".exe"
	if p, e := exec.LookPath(exe); e == nil {
		return p
	}
	var dirs []string
	dirs = append(dirs, registryToolDirs()...)
	for _, key := range []string{"FFMPEG_HOME", "MKVTOOLNIX_HOME"} {
		if root := os.Getenv(key); root != "" {
			dirs = append(dirs, root, filepath.Join(root, "bin"))
		}
	}
	self, _ := os.Executable()
	dirs = append(dirs, filepath.Dir(self), filepath.Join(filepath.Dir(self), "tools"), filepath.Join(filepath.Dir(self), "tools", "ffmpeg", "bin"), filepath.Join(filepath.Dir(self), "tools", "MKVToolNix"))
	for _, ev := range []string{"ProgramFiles", "ProgramFiles(x86)", "LOCALAPPDATA"} {
		if root := os.Getenv(ev); root != "" {
			dirs = append(dirs, root, filepath.Join(root, "MKVToolNix"), filepath.Join(root, "ffmpeg", "bin"), filepath.Join(root, "FFmpeg", "bin"), filepath.Join(root, "Programs", "ffmpeg", "bin"), filepath.Join(root, "Programs", "MKVToolNix"))
		}
	}
	dirs = append(dirs, `C:\ffmpeg\bin`, `C:\tools\ffmpeg\bin`, `C:\Program Files\MKVToolNix`, `C:\Program Files (x86)\MKVToolNix`)
	for _, d := range dirs {
		p := filepath.Join(d, exe)
		if st, e := os.Stat(p); e == nil && !st.IsDir() {
			return p
		}
	}
	return ""
}
func resolveTool(p string) (string, error) {
	if p == "" {
		return "", errors.New(pair("该工具不在PATH和常见安装位置", "Tool not found in PATH or common installation locations"))
	}
	if strings.ContainsAny(p, `\/:`) {
		st, e := os.Stat(p)
		if e != nil || st.IsDir() {
			return "", errors.New(pair("路径不存在："+p, "Path not found: "+p))
		}
		return p, nil
	}
	return exec.LookPath(p)
}
func nextName(source, suffix, ext string) string {
	base := strings.TrimSuffix(source, filepath.Ext(source))
	for n := 1; n < 10000; n++ {
		s := ""
		if n > 1 {
			s = fmt.Sprintf("_%d", n)
		}
		p := base + suffix + s + ext
		if _, e := os.Stat(p); os.IsNotExist(e) {
			return p
		}
	}
	return base + suffix + "_many" + ext
}

// Noncolliding output path in the selected folder (never uses the input filename).
func nextNameInDir(src, suffix, ext, outDir string) string {
	name := strings.TrimSuffix(filepath.Base(src), filepath.Ext(src))
	if outDir == "" {
		outDir = filepath.Dir(src)
	}
	base := filepath.Join(outDir, name)
	for n := 1; n < 10000; n++ {
		postfix := ""
		if n > 1 {
			postfix = fmt.Sprintf("_%d", n)
		}
		candidate := base + suffix + postfix + ext
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
	return base + suffix + "_many" + ext
}

// Disk usage peaks with both a temporary lossless HEVC file and the finished MKV.
// This is a conservative estimate, not an exact prediction of muxing overhead.
func diskSpace(dir string) (uint64, error) {
	var free, total, totalFree uint64
	r, _, e := pDiskFree.Call(uintptr(unsafe.Pointer(wide(dir))), uintptr(unsafe.Pointer(&free)), uintptr(unsafe.Pointer(&total)), uintptr(unsafe.Pointer(&totalFree)))
	if r == 0 {
		return 0, fmt.Errorf("GetDiskFreeSpaceExW: %v", e)
	}
	return free, nil
}
func gib(n uint64) string { return fmt.Sprintf("%.2f GiB", float64(n)/(1024*1024*1024)) }
func checkOutputSpace(dest string, sourceBytes int64, mode string, log func(string)) error {
	if sourceBytes < 0 {
		return errors.New(pair("文件大小无效", "Invalid file size"))
	}
	needed := uint64(float64(sourceBytes)*2.20) + uint64(128<<20)
	if mode == "repair" {
		needed = uint64(float64(sourceBytes)*3.20) + uint64(256<<20)
	}
	free, e := diskSpace(dest)
	if e != nil {
		log(pair("无法读取目标磁盘剩余空间：", "Could not read free disk space: ") + e.Error())
		return nil
	}
	log(pair(fmt.Sprintf("磁盘剩余 %s；预计至少需要 %s。", gib(free), gib(needed)), fmt.Sprintf("Free disk space: %s; estimated minimum required: %s.", gib(free), gib(needed))))
	if free < needed {
		return errors.New(pair(fmt.Sprintf("磁盘空间不足：剩余 %s，至少需要 %s。", gib(free), gib(needed)), fmt.Sprintf("Insufficient disk space: %s free; at least %s required.", gib(free), gib(needed))))
	}
	return nil
}

func command(ctx context.Context, tool string, args ...string) (string, error) {
	c := exec.CommandContext(ctx, tool, args...)
	c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	out, e := c.CombinedOutput()
	text := string(out)
	if e != nil {
		return text, fmt.Errorf("%s: %w\n%s", filepath.Base(tool), e, truncate(text, 850))
	}
	return text, nil
}
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	r := []rune(s)
	if len(r) > n {
		r = r[len(r)-n:]
	}
	return string(r)
}

type Probe struct {
	Streams []Stream `json:"streams"`
	Format  struct {
		Duration string `json:"duration"`
	} `json:"format"`
	Chapters []json.RawMessage `json:"chapters"`
}
type Stream struct {
	Disposition map[string]int `json:"disposition"`
	Index       int            `json:"index"`
	Type        string         `json:"codec_type"`
	Codec       string         `json:"codec_name"`
	Width       int            `json:"width"`
	Height      int            `json:"height"`
	AvgFPS      string         `json:"avg_frame_rate"`
	RFPS        string         `json:"r_frame_rate"`
	Extra       string         `json:"extradata"`
	ExtraSize   int            `json:"extradata_size"`
	SideData    []DVRecord     `json:"side_data_list"`
}
type DVRecord struct {
	Type       string `json:"side_data_type"`
	Profile    int    `json:"dv_profile"`
	RPUPresent int    `json:"rpu_present_flag"`
	ELPresent  int    `json:"el_present_flag"`
	BLPresent  int    `json:"bl_present_flag"`
}
type Report struct {
	Filename                             string
	Vid                                  Stream
	DV                                   *DVRecord
	Complete                             bool
	VPS, SPS, PPS                        int
	HVCCValid                            bool
	HVCCError                            string
	VideoNum, AudioNum, SubNum, Chapters int
	AttachedPicNum                       int
	Duration                             float64
	ExtradataSize                        int
	RateNum, RateDen                     int64
}

func parseHVCC(raw []byte) (vps, sps, pps int, valid bool, err string) {
	if len(raw) < 23 {
		return 0, 0, 0, false, "hvcC不足23字节"
	}
	if raw[0] != 1 {
		return 0, 0, 0, false, "hvcC version !=1"
	}
	arrays := int(raw[22])
	pos := 23
	for a := 0; a < arrays; a++ {
		if pos+3 > len(raw) {
			return vps, sps, pps, false, "数组头截断"
		}
		t := raw[pos] & 63
		ct := int(raw[pos+1])<<8 | int(raw[pos+2])
		pos += 3
		for j := 0; j < ct; j++ {
			if pos+2 > len(raw) {
				return vps, sps, pps, false, "NAL长度截断"
			}
			l := int(raw[pos])<<8 | int(raw[pos+1])
			pos += 2
			if pos+l > len(raw) {
				return vps, sps, pps, false, "NAL内容截断"
			}
			if l < 3 || raw[pos]&0x80 != 0 || (raw[pos]>>1)&63 != t || raw[pos+1]&7 == 0 {
				return vps, sps, pps, false, "Invalid HEVC NAL header/length"
			}
			pos += l
			switch t {
			case 32:
				vps++
			case 33:
				sps++
			case 34:
				pps++
			}
		}
	}
	if pos != len(raw) {
		return vps, sps, pps, false, "数组结束位置与长度不符"
	}
	return vps, sps, pps, true, ""
}
func decodeHexdump(dump string) []byte {
	var out []byte
	scan := bufio.NewScanner(strings.NewReader(dump))
	for scan.Scan() {
		line := scan.Text()
		k := strings.IndexByte(line, ':')
		if k < 0 || k > 12 {
			continue
		}
		prefix := strings.TrimSpace(line[:k])
		if len(prefix) != 8 {
			continue
		}
		if _, e := strconv.ParseUint(prefix, 16, 32); e != nil {
			continue
		}
		body := strings.TrimLeft(line[k+1:], " \t")
		if i := strings.Index(body, "  "); i >= 0 {
			body = body[:i]
		}
		var h strings.Builder
		for _, part := range strings.Fields(body) {
			if len(part)%2 != 0 || len(part) > 16 {
				break
			}
			if _, e := hex.DecodeString(part); e != nil {
				break
			}
			h.WriteString(part)
		}
		if b, e := hex.DecodeString(h.String()); e == nil {
			out = append(out, b...)
		}
	}
	return out
}
func rateRatio(fps string) (int64, int64, error) {
	bits := strings.Split(fps, "/")
	if len(bits) != 2 {
		return 0, 0, fmt.Errorf("未知帧率%s", fps)
	}
	a, e1 := strconv.ParseInt(bits[0], 10, 64)
	b, e2 := strconv.ParseInt(bits[1], 10, 64)
	if e1 != nil || e2 != nil || a <= 0 || b <= 0 || float64(a)/float64(b) > 240 {
		return 0, 0, fmt.Errorf("帧率无效：%s", fps)
	}
	return a, b, nil
}
func probe(ctx context.Context, exe, path string) (Report, error) {
	var r Report
	r.Filename = path
	args := []string{"-v", "error", "-show_streams", "-show_format", "-show_chapters", "-show_data", "-show_entries", "stream=index,codec_type,codec_name,width,height,avg_frame_rate,r_frame_rate,extradata,extradata_size,side_data_list:stream_disposition=attached_pic:stream_side_data:format=duration:chapter=id,start_time,end_time", "-of", "json", path}
	out, e := command(ctx, exe, args...)
	if e != nil {
		return r, e
	}
	var p Probe
	if e := json.Unmarshal([]byte(out), &p); e != nil {
		return r, fmt.Errorf("ffprobe JSON解析失败：%w", e)
	}
	countMediaStreams(&r, p.Streams)
	if r.VideoNum == 0 {
		return r, errors.New(pair("未检测到主视频轨（封面不计入视频轨）", "No main video track detected (cover images are not video tracks)"))
	}
	r.Chapters = len(p.Chapters)
	r.ExtradataSize = r.Vid.ExtraSize
	r.Duration, _ = strconv.ParseFloat(p.Format.Duration, 64)
	for i := range r.Vid.SideData {
		v := &r.Vid.SideData[i]
		if v.Type == "DOVI configuration record" {
			r.DV = v
			break
		}
	}
	raw := decodeHexdump(r.Vid.Extra)
	r.VPS, r.SPS, r.PPS, r.HVCCValid, r.HVCCError = parseHVCC(raw)
	if len(raw) != r.ExtradataSize {
		r.HVCCValid = false
		r.HVCCError = fmt.Sprintf("实际十六进制数据%d字节，ffprobe报告%d字节", len(raw), r.ExtradataSize)
	}
	r.Complete = r.HVCCValid && r.VPS > 0 && r.SPS > 0 && r.PPS > 0
	a, b, e := rateRatio(r.Vid.AvgFPS)
	if e != nil {
		a, b, e = rateRatio(r.Vid.RFPS)
	}
	if e == nil {
		r.RateNum = a
		r.RateDen = b
	}
	return r, nil
}
func (r Report) summary() string {
	dv := "无DV"
	if r.DV != nil {
		dv = fmt.Sprintf("P%d / RPU标志%d", r.DV.Profile, r.DV.RPUPresent)
	}
	return fmt.Sprintf("HEVC=%s, DV=%s, hvcC=%dB, VPS/SPS/PPS=%d/%d/%d, 主视频%d 封面%d 音频%d 字幕%d 章节%d, 时长%.2fs", r.Vid.Codec, dv, r.ExtradataSize, r.VPS, r.SPS, r.PPS, r.VideoNum, r.AttachedPicNum, r.AudioNum, r.SubNum, r.Chapters, r.Duration)
}

// Compare timestamps extracted from the original and repaired Matroska track.
// mkvextract timestamps_v2 values are expressed in milliseconds.
func compareVideoTimestamps(original, repaired string) (int, float64, error) {
	a, err := os.Open(original)
	if err != nil {
		return 0, 0, err
	}
	defer a.Close()
	b, err := os.Open(repaired)
	if err != nil {
		return 0, 0, err
	}
	defer b.Close()
	sa := bufio.NewScanner(a)
	sb := bufio.NewScanner(b)
	sa.Buffer(make([]byte, 65536), 1024*1024)
	sb.Buffer(make([]byte, 65536), 1024*1024)
	count, maxDelta := 0, float64(0)
	nextValue := func(scanner *bufio.Scanner) (float64, bool, error) {
		for scanner.Scan() {
			str := strings.TrimSpace(scanner.Text())
			if str == "" || strings.HasPrefix(str, "#") {
				continue
			}
			v, e := strconv.ParseFloat(str, 64)
			if e != nil {
				return 0, false, e
			}
			return v, true, nil
		}
		if e := scanner.Err(); e != nil {
			return 0, false, e
		}
		return 0, false, nil
	}
	for {
		v1, ok1, e1 := nextValue(sa)
		v2, ok2, e2 := nextValue(sb)
		if e1 != nil {
			return count, maxDelta, e1
		}
		if e2 != nil {
			return count, maxDelta, e2
		}
		if ok1 != ok2 {
			return count, maxDelta, fmt.Errorf("视频帧数不一致：时间戳数量不同（已比较%d帧）", count)
		}
		if !ok1 {
			break
		}
		delta := math.Abs(v1 - v2)
		if delta > maxDelta {
			maxDelta = delta
		}
		count++
	}
	if count == 0 {
		return 0, 0, errors.New("无有效视频时间戳")
	}
	return count, maxDelta, nil
}
func txtFile(path, report string) error {
	f, e := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if e != nil {
		return e
	}
	defer f.Close()
	_, e = f.Write(append([]byte{0xef, 0xbb, 0xbf}, []byte(report)...))
	return e
}
func processOne(ctx context.Context, src, outDir, mode string, tools map[string]string, log func(string), onStage func(string, string, int64, func() error) error) error {
	srcStat, e := os.Stat(src)
	if e != nil {
		return e
	}
	if outDir == "" {
		outDir = filepath.Dir(src)
	}
	if outStat, e := os.Stat(outDir); e != nil || !outStat.IsDir() {
		return fmt.Errorf("%s", pair("输出文件夹无效："+outDir, "Invalid output folder: "+outDir))
	}
	log(event("output", outDir))
	before, e := probe(ctx, tools["ffprobe"], src)
	if e != nil {
		return e
	}
	log(pair("原片："+before.summary(), "Source: "+before.summaryEN()))
	videoDir, reportDir, e := outputSubdirs(outDir)
	if e != nil {
		return e
	}
	log(event("videos", videoDir))
	log(event("reports", reportDir))
	if mode == "inspect" {
		return writeInspection(src, outDir, reportDir, before, log, reportLanguage(ctx))
	}
	if before.Vid.Codec != "hevc" {
		return errors.New(pair("只支持HEVC/H.265视频", "Only HEVC/H.265 video is supported"))
	}
	if before.VideoNum != 1 {
		return errors.New(pair("为了防止丢失视频轨，不支持多视频轨MKV", "Multiple video tracks are not supported, to avoid losing tracks"))
	}
	if mode == "repair" {
		if err := preflight(before); err != nil {
			return err
		}
	}
	if mode != "inspect" {
		if err := validateProcessTools(tools, mode); err != nil {
			return err
		}
	}
	if mode != "inspect" {
		if err := checkOutputSpace(outDir, srcStat.Size(), mode, log); err != nil {
			return err
		}
	}
	if !before.Complete {
		log(event("headers"))
	}
	reportPath := nextNameInDir(src, "_DV_Fix_Report", ".txt", reportDir)
	lines, english := reportStart(src, outDir, before)
	workDir, e := os.MkdirTemp(outDir, ".oppo-p5-*")
	if e != nil {
		return e
	}
	defer os.RemoveAll(workDir)
	exportOutput := nextNameInDir(src, "_DV_Fix_HEVC", ".hevc", videoDir)
	raw := filepath.Join(workDir, "video.hevc")
	log(event("extracting"))
	var out string
	e = onStage("extract", raw, srcStat.Size()*95/100, func() error {
		var err error
		out, err = command(ctx, tools["ffmpeg"], "-hide_banner", "-nostdin", "-v", "warning", "-n", "-i", src, "-map", videoMap(before), "-c:v", "copy", "-bsf:v", "hevc_mp4toannexb", "-f", "hevc", raw)
		return err
	})
	if strings.Contains(out, "No parameter sets in the extradata") {
		log(pair("FFmpeg 检测到源 hvcC 中没有参数集。", "FFmpeg detected no parameter sets in the source hvcC."))
	}
	if e != nil {
		return e
	}
	stat, e := os.Stat(raw)
	if e != nil {
		return e
	}
	if stat.Size() <= 0 {
		return errors.New(pair("提取HEVC文件为空", "Extracted HEVC file is empty"))
	}
	log(pair(fmt.Sprintf("HEVC 提取完成：%.2f GiB", float64(stat.Size())/(1024*1024*1024)), fmt.Sprintf("HEVC extraction complete: %.2f GiB", float64(stat.Size())/(1024*1024*1024))))
	if mode == "export" {
		if e := publishOutput(ctx, raw, exportOutput, srcStat.ModTime()); e != nil {
			return e
		}
		raw = exportOutput
		log(event("media", raw))
		lines = append(lines, "模式：仅导出HEVC", "导出文件："+raw, "结论：成功无损导出码流；未验证完整解码和实际Dolby Vision播放。")
		english = append(english, "Mode: Export HEVC only", "Exported file: "+raw, "Result: HEVC bitstream exported without re-encoding; full decoding and Dolby Vision playback were not verified.")
		if e := txtFile(reportPath, localizedReport(reportLanguage(ctx), src, outDir, mode, raw, before, nil, nil, "")); e != nil {
			return e
		}
		log(event("report", reportPath))
		return nil
	}
	if before.RateNum == 0 {
		return errors.New(pair("无法确认帧率，避免音画不同步，已停止。", "Frame rate could not be confirmed; processing stopped to protect sync."))
	}
	finalOutput := nextNameInDir(src, "_DV_Fix", ".mkv", videoDir)
	output := filepath.Join(workDir, "repaired.mkv")
	// On muxing error/cancellation, remove incomplete output; preserve complete files if structural tests fail.
	muxCompleted := false
	defer func() {
		if !muxCompleted {
			_ = os.Remove(output)
		}
	}()
	// Recover original video packet PTS where possible. The mkvextract path is the most accurate for VFR.
	timestamps := filepath.Join(workDir, "timestamps.txt")
	defer os.Remove(timestamps)
	tsMode := ""
	ident, e := command(ctx, tools["mkvmerge"], "-J", src)
	id := -1
	if e == nil {
		var parsed struct {
			Tracks []struct {
				ID   int    `json:"id"`
				Type string `json:"type"`
			} `json:"tracks"`
		}
		if json.Unmarshal([]byte(ident), &parsed) == nil {
			for _, t := range parsed.Tracks {
				if t.Type == "video" {
					id = t.ID
					break
				}
			}
		}
	}
	if id < 0 {
		return errors.New("Cannot identify source video track; repair stopped")
	}
	tsOutput, tsErr := command(ctx, tools["mkvextract"], src, "timestamps_v2", fmt.Sprintf("%d:%s", id, timestamps))
	_ = tsOutput
	prior := []string{}
	if tsErr == nil {
		if st, e := os.Stat(timestamps); e == nil && st.Size() > 100 {
			prior = []string{"--timestamps", "0:" + timestamps}
			tsMode = "保留原始视频逐帧时间戳"
		}
	}
	if len(prior) == 0 {
		return errors.New("Cannot preserve original video timestamps; repair stopped safely. " + truncate(tsOutput, 250))
	}

	log(event("muxing"))
	args := append([]string{"-o", output}, prior...)
	args = append(args, raw, "--no-video", src)
	var muxout string
	muxerr := onStage("mux", output, srcStat.Size(), func() error {
		var err error
		muxout, err = command(ctx, tools["mkvmerge"], args...)
		return err
	})
	if muxerr != nil {
		// MKVToolNix uses exit code 1 for warnings. Output needs to exist to accept this case.
		var ex *exec.ExitError
		if !errors.As(muxerr, &ex) || ex.ExitCode() != 1 {
			return muxerr
		}
		log(pair("MKVmerge 警告：", "MKVmerge warning: ") + truncate(muxout, 450))
	}
	if st, e := os.Stat(output); e != nil || st.Size() == 0 {
		return errors.New(pair("输出的MKV不存在或为空", "Output MKV is missing or empty"))
	}
	muxCompleted = true
	// mkvmerge's --timestamps may accurately preserve each frame's PTS while
	// writing a rounded DefaultDuration (e.g. 42 ms instead of 1001/24000 s).
	// Update ONLY the Matroska track header. Do not re-time actual frames.
	// mkvpropedit expects the DefaultDuration in integer nanoseconds.
	frameNS := int64(math.Round(1e9 * float64(before.RateDen) / float64(before.RateNum)))
	log(pair(fmt.Sprintf("设置默认帧时长：%d ns/帧（%d/%d）", frameNS, before.RateNum, before.RateDen), fmt.Sprintf("Setting default frame duration: %d ns/frame (%d/%d)", frameNS, before.RateNum, before.RateDen)))
	propOut, propErr := command(ctx, tools["mkvpropedit"], output, "--edit", "track:v1", "--set", fmt.Sprintf("default-duration=%d", frameNS))
	if propErr != nil {
		return fmt.Errorf("%s: %w", pair("视频帧率头信息修正失败", "Could not update the video frame-rate header"), propErr)
	}
	if strings.TrimSpace(propOut) != "" {
		log("MKVpropedit: " + truncate(strings.TrimSpace(propOut), 250))
	}
	after, e := probe(ctx, tools["ffprobe"], output)
	if e != nil {
		return e
	}
	log(pair("重建："+after.summary(), "Rebuilt: "+after.summaryEN()))
	log(pair(fmt.Sprintf("帧率：原片 %s，输出 %s", before.Vid.AvgFPS, after.Vid.AvgFPS), fmt.Sprintf("Frame rate: source %s, output %s", before.Vid.AvgFPS, after.Vid.AvgFPS)))
	var ptsCheckName string
	ptsEnglish := "Frame-by-frame timestamp comparison: output extraction failed"
	ptsReportDetail := "PTS: extraction=FAIL"
	ptsVerified := false
	if tsMode == "保留原始视频逐帧时间戳" {
		outputTS := filepath.Join(workDir, "verify-timestamps.txt")
		defer os.Remove(outputTS)
		tsInfo, extractErr := command(ctx, tools["mkvextract"], output, "timestamps_v2", "0:"+outputTS)
		if extractErr != nil {
			ptsCheckName = "逐帧时间戳对照（输出提取失败）"
			log(pair("无法提取输出视频时间戳：", "Could not extract output video timestamps: ") + truncate(tsInfo, 250))
		} else {
			frameCount, maxDelta, compErr := compareVideoTimestamps(timestamps, outputTS)
			if compErr != nil {
				ptsCheckName = "逐帧时间戳对照（" + compErr.Error() + "）"
				ptsEnglish = "Frame-by-frame timestamp comparison: invalid timestamps or frame count mismatch"
				ptsReportDetail = "PTS: timestamps/frame_count=FAIL"
			} else {
				ptsVerified = maxDelta <= 1.0
				ptsEnglish = fmt.Sprintf("Frame-by-frame timestamp comparison: %d frames, maximum difference %.4f ms (limit: 1 ms)", frameCount, maxDelta)
				ptsReportDetail = fmt.Sprintf("PTS: frames=%d; max Δ=%.4f ms; Δ ≤ 1 ms", frameCount, maxDelta)
				ptsCheckName = fmt.Sprintf("逐帧时间戳对照：%d帧，最大差值%.4f毫秒（限1毫秒）", frameCount, maxDelta)
			}
			log(pair(ptsCheckName, ptsEnglish))
		}
	}
	checks := []struct {
		name string
		ok   bool
	}{{"视频仍为HEVC", after.Vid.Codec == "hevc"}, {"视频分辨率保持", before.Vid.Width == after.Vid.Width && before.Vid.Height == after.Vid.Height}, {"输出hvcC有VPS/SPS/PPS", after.Complete}, {"音轨数未减少", before.AudioNum == after.AudioNum}, {"字幕轨道数未减少", before.SubNum == after.SubNum}, {"章节数未减少", before.Chapters == after.Chapters}}
	checks = append(checks, struct {
		name string
		ok   bool
	}{"主视频轨数量保持", before.VideoNum == after.VideoNum})
	if before.AttachedPicNum > 0 {
		checks = append(checks, struct {
			name string
			ok   bool
		}{"封面数量保持", before.AttachedPicNum == after.AttachedPicNum})
	}
	if before.DV != nil {
		checks = append(checks, struct {
			name string
			ok   bool
		}{"DV Profile未变化", after.DV != nil && before.DV.Profile == after.DV.Profile})
		checks = append(checks, struct {
			name string
			ok   bool
		}{"DV RPU存在标志未变化", after.DV != nil && before.DV.RPUPresent == after.DV.RPUPresent})
	}
	if before.Duration > 0 && after.Duration > 0 {
		checks = append(checks, struct {
			name string
			ok   bool
		}{"文件时长误差≤1秒", abs(before.Duration-after.Duration) <= 1})
	}
	if before.RateNum > 0 && after.RateNum > 0 {
		checks = append(checks, struct {
			name string
			ok   bool
		}{"平均帧率标记一致", abs(float64(before.RateNum)/float64(before.RateDen)-float64(after.RateNum)/float64(after.RateDen)) < 0.02})
	}
	if ptsCheckName != "" {
		checks = append(checks, struct {
			name string
			ok   bool
		}{ptsCheckName, ptsVerified})
	}
	pass := true
	for _, c := range checks {
		if !c.ok {
			pass = false
		}
	}
	if !pass {
		finalOutput = nextNameInDir(src, "_DV_Fix_Verification_Failed", ".mkv", videoDir)
	}
	if e := publishOutput(ctx, output, finalOutput, srcStat.ModTime()); e != nil {
		return e
	}
	output = finalOutput
	lines = append(lines, "修复输出："+output, "重建结果："+after.summary(), "时间戳："+tsMode, fmt.Sprintf("帧率：原片 %s；修复后 %s；默认帧时长 %d ns", before.Vid.AvgFPS, after.Vid.AvgFPS, frameNS), "", "结构自检：")
	english = append(english, "Repaired output: "+output, "Rebuilt: "+after.summaryEN(), "Timestamps: original per-frame video timestamps preserved", fmt.Sprintf("Frame rate: original %s; output %s; default frame duration %d ns", before.Vid.AvgFPS, after.Vid.AvgFPS, frameNS), "", "Structural checks:")
	for _, c := range checks {
		mark := "PASS"
		if !c.ok {
			pass = false
			mark = "FAIL"
		}
		lines = append(lines, mark+"  "+c.name)
		label := checkEnglish[c.name]
		if c.name == ptsCheckName {
			label = ptsEnglish
		}
		english = append(english, mark+"  "+label)
	}
	result := "结构自检通过（仍需OPPO+电视实播验证）"
	if !pass {
		result = "结构自检未通过，请检查详情，不可视作修复成功"
	}
	lines = append(lines, "", result, "重要：结构自检及PTS对照不证明OPPO实播DV正常或音画同步绝对正确；请抽查开头、中间、结尾。")
	conclusion := "Structural checks passed; OPPO/TV playback still needs verification."
	if !pass {
		conclusion = "Structural checks FAILED; this output must not be treated as a successful repair."
	}
	english = append(english, "", conclusion, "Important: structural checks and PTS comparison do not prove correct OPPO Dolby Vision playback or perfect A/V sync. Check the beginning, middle and end.")
	reportChecks := make([]reportCheck, 0, len(checks))
	for _, c := range checks {
		name := localizedCheckName(reportLanguage(ctx), c.name)
		if c.name == ptsCheckName {
			name = "PTS (Δ ≤ 1 ms)"
		}
		reportChecks = append(reportChecks, reportCheck{name, c.ok})
	}
	technical := fmt.Sprintf("FPS=%s → %s; default_frame_duration=%d ns\n%s", before.Vid.AvgFPS, after.Vid.AvgFPS, frameNS, ptsReportDetail)
	if e := txtFile(reportPath, localizedReport(reportLanguage(ctx), src, outDir, mode, output, before, &after, reportChecks, technical)); e != nil {
		return e
	}
	log(event("media", output))
	log(event("report", reportPath))
	log(pair(result, conclusion))
	if !pass {
		return errors.New(pair("至少一个结构检查失败（已保留输出及报告）", "At least one structural check failed; output and report were retained"))
	}
	return nil
}
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
func crashLog(where string, r interface{}) {
	dir := filepath.Join(os.Getenv("LOCALAPPDATA"), "OPPO_DV_Repair")
	if dir == "OPPO_DV_Repair" {
		dir = os.TempDir()
	}
	_ = os.MkdirAll(dir, 0755)
	file := filepath.Join(dir, "crash.log")
	b := []byte(fmt.Sprintf("[%s] %s: %v\r\n%s\r\n", time.Now().Format(time.RFC3339), where, r, string(debug.Stack())))
	f, e := os.OpenFile(file, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if e == nil {
		_, _ = f.Write(b)
		_ = f.Close()
	}
}
func droppedFiles(drop uintptr) []string {
	n, _, _ := pDragQuery.Call(drop, ^uintptr(0), 0, 0)
	var paths []string
	for i := uintptr(0); i < n; i++ {
		l, _, _ := pDragQuery.Call(drop, i, 0, 0)
		if l == 0 || l > 32767 {
			continue
		}
		buf := make([]uint16, l+1)
		pDragQuery.Call(drop, i, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
		paths = append(paths, syscall.UTF16ToString(buf))
	}
	pDragFinish.Call(drop)
	return paths
}
func wndProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) (result uintptr) {
	defer func() {
		if r := recover(); r != nil {
			crashLog("wndProc", r)
			showErr(hwnd, "内部错误已写入 %LOCALAPPDATA%\\OPPO_DV_Repair\\crash.log")
		}
	}()
	switch msg {
	case 0x0020: // WM_SETCURSOR
		if lParam&0xffff == 1 && app.customCursorTarget(wParam) && app.arrowCursor != 0 {
			user32.NewProc("SetCursor").Call(app.arrowCursor)
			return 1
		}
	case 0x0215: // WM_CAPTURECHANGED
		if hwnd == app.globeHwnd && lParam != hwnd {
			app.resetGlobeVisuals()
			return 0
		}
	case WM_CTLCOLORSTATIC:
		pSetTextColor.Call(wParam, rgb(221, 234, 237))
		pSetBkColor.Call(wParam, rgb(23, 32, 45))
		return app.darkBrush
	case WM_CTLCOLOREDIT:
		pSetTextColor.Call(wParam, rgb(236, 247, 247))
		pSetBkColor.Call(wParam, rgb(38, 50, 63))
		return app.editBrush
	case WM_CTLCOLORBTN:
		pSetTextColor.Call(wParam, rgb(221, 234, 237))
		pSetBkColor.Call(wParam, rgb(23, 32, 45))
		return app.darkBrush
	case WM_PAINT:
		if hwnd == app.globeHwnd {
			paintGlobe(hwnd)
			return 0
		}
		if hwnd == app.languagePopup {
			paintLangPopup(hwnd)
			return 0
		}
		if hwnd == app.progressHwnd {
			paintProgress(hwnd)
			return 0
		}
	case WM_LBUTTONDOWN:
		if hwnd == app.globeHwnd {
			user32.NewProc("SetCapture").Call(hwnd)
			app.globePressed = true
			app.globePhase = 2
			pInvalidate.Call(hwnd, 0, 0)
			return 0
		}
	case WM_LBUTTONUP:
		if hwnd == app.globeHwnd {
			user32.NewProc("ReleaseCapture").Call()
			app.globePressed = true
			app.globePhase = 2
			pSetTimer.Call(hwnd, 301, 35, 0)
			app.toggleLanguageMenu()
			return 0
		}
		if hwnd == app.languagePopup {
			y := int(int16((lParam >> 16) & 0xffff))
			row := (int(float64(y)/app.scale) - 8) / 28
			if int(float64(y)/app.scale) >= 8 && row >= 0 && row < len(languageList) {
				app.switchLanguage(languageList[row].code)
			}
			app.closeLanguageMenu()
			return 0
		}
		if hwnd == app.hwnd {
			app.closeLanguageMenu()
		}
	case WM_MOUSEMOVE:
		if hwnd == app.languagePopup {
			y := int(int16((lParam >> 16) & 0xffff))
			row := (int(float64(y)/app.scale) - 8) / 28
			if row < 0 || row >= len(languageList) {
				row = -1
			}
			if app.menuHover != row {
				oldRow := app.menuHover
				app.menuHover = row
				for _, r := range []int{oldRow, row} {
					if r >= 0 {
						y0 := int32(math.Round(float64(8+r*28) * app.scale))
						y1 := int32(math.Round(float64(8+(r+1)*28) * app.scale))
						var dirty = RECT{0, y0, int32(math.Round(236 * app.scale)), y1}
						pInvalidate.Call(hwnd, uintptr(unsafe.Pointer(&dirty)), 0)
					}
				}
			}
			return 0
		}
	case WM_ACTIVATE:
		if hwnd == app.languagePopup && wParam&0xffff == WA_INACTIVE {
			// Delay until after any in-flight WM_LBUTTONUP selection event.
			pPost.Call(app.hwnd, WM_APP+2, 0, 0)
			return 0
		}
	case WM_KILLFOCUS:
		if hwnd == app.languagePopup && app.menuShown {
			pPost.Call(app.hwnd, WM_APP+2, 0, 0)
		}
	case WM_KEYDOWN:
		if wParam == 0x1B && app.menuShown {
			app.closeLanguageMenu()
			return 0
		}
	case WM_TIMER:
		if hwnd == app.globeHwnd && wParam == 301 {
			if app.globePhase > 0 {
				app.globePhase--
			} else {
				app.globePressed = false
				pKillTimer.Call(hwnd, 301)
			}
			pInvalidate.Call(hwnd, 0, 0)
			return 0
		}
	case WM_ERASEBKGND:
		if hwnd == app.globeHwnd || hwnd == app.languagePopup {
			return 1
		}
		if hwnd == app.progressHwnd {
			return 1
		}
	case WM_COMMAND:
		id := int(wParam & 0xffff)
		hi := int((wParam >> 16) & 0xffff)
		if id != ID_LANGUAGE && app.menuShown {
			app.closeLanguageMenu()
		}
		if hi == EN_KILLFOCUS && app.initDone {
			// Capture manual edits as soon as the user leaves the textbox.
			if lParam == app.outputEdit {
				app.saveSettings()
			}
			for _, edit := range app.tools {
				if lParam == edit {
					app.saveSettings()
					break
				}
			}
		} else if hi == 0 && id != 0 {
			app.onCommand(id)
		}
		return 0
	case WM_DROPFILES:
		paths := droppedFiles(wParam)
		if !app.working {
			app.addPaths(paths)
		} else {
			app.appendLog(pair("正在处理文件，暂不接受拖入。", "A task is running; dropped files cannot be added yet."))
		}
		return 0
	case WM_DPICHANGED:
		if app.initDone {
			// Windows sends a suggested rectangle on a DPI transition.
			// Use the target monitor work area, never a manual UI-scale dropdown.
			dpi := uint32(wParam & 0xffff)
			var suggested *RECT
			if lParam != 0 {
				suggested = (*RECT)(unsafe.Pointer(lParam))
			}
			if suggested != nil {
				if work, ok := currentWorkRect(*suggested); ok {
					app.automatic = screenScale(int(work.Right-work.Left), int(work.Bottom-work.Top), dpi)
				} else {
					app.automatic = autoScale()
				}
			} else {
				app.automatic = autoScale()
			}
			app.setScale(app.automatic, suggested)
		}
		return 0
	case WM_DISPLAYCHANGE:
		if app.initDone {
			app.automatic = autoScale()
			app.setScale(app.automatic, nil)
		}
		return 0
	case WM_APP + 2:
		if app.menuShown {
			app.closeLanguageMenu()
		}
		return 0
	case WM_APP + 1:
		app.flush()
		return 0
	case WM_CLOSE:
		app.saveSettings()
		if app.cancel != nil {
			app.closeAfterCancel = true
			app.requestCancel()
			return 0 // let the worker stop its child process and clean scratch files
		}
		pDestroy.Call(hwnd)
		return 0
	case WM_DESTROY:
		pPostQuit.Call(0)
		return 0
	}
	r, _, _ := pDefProc.Call(hwnd, uintptr(msg), wParam, lParam)
	return r
}
func main() {
	instance, exists, err := acquireInstance(`Local\OPPO_DV_P5_Repair_SingleInstance`)
	if err != nil {
		showErr(0, "Cannot create single-instance lock / 无法建立单实例锁: "+err.Error())
		return
	}
	defer syscall.CloseHandle(instance)
	if exists {
		activateExisting()
		return
	}

	defer func() {
		if r := recover(); r != nil {
			crashLog("main", r)
			showErr(0, "程序异常；请查看 %LOCALAPPDATA%\\OPPO_DV_Repair\\crash.log")
		}
	}()
	runtime.LockOSThread()
	if e := pDpiCtx.Find(); e == nil {
		// -4 = DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2
		r, _, _ := pDpiCtx.Call(^uintptr(3))
		if r == 0 {
			pDpi.Call()
		}
	} else {
		pDpi.Call()
	}
	app.automatic = autoScale()
	app.scale = app.automatic
	inst, _, _ := pGetModule.Call(0)
	app.arrowCursor, _, _ = user32.NewProc("LoadCursorW").Call(0, 32512)
	app.darkBrush, _, _ = pCreateBrush.Call(rgb(23, 32, 45))
	app.editBrush, _, _ = pCreateBrush.Call(rgb(38, 50, 63))
	// Register a custom pixel progress window with the same safe native WndProc.
	progressCls := wide("OPPO_P5_PROGRESS")
	progressWC := WNDCLASSEX{Size: uint32(unsafe.Sizeof(WNDCLASSEX{})), Style: 3, WndProc: syscall.NewCallback(wndProc), Instance: inst, Cursor: app.arrowCursor, Background: app.darkBrush, ClassName: progressCls}
	pRegister.Call(uintptr(unsafe.Pointer(&progressWC)))
	initFlagSprites()
	globeClass := wide("OPPO_P5_GLOBE")
	globeWC := WNDCLASSEX{Size: uint32(unsafe.Sizeof(WNDCLASSEX{})), Style: 3, WndProc: syscall.NewCallback(wndProc), Instance: inst, Cursor: app.arrowCursor, Background: app.darkBrush, ClassName: globeClass}
	if res, _, _ := pRegister.Call(uintptr(unsafe.Pointer(&globeWC))); res == 0 {
		showErr(0, "地球按钮控件初始化失败")
		return
	}
	menuClass := wide("OPPO_P5_LANGPOP")
	menuWC := WNDCLASSEX{Size: uint32(unsafe.Sizeof(WNDCLASSEX{})), Style: 3, WndProc: syscall.NewCallback(wndProc), Instance: inst, Cursor: app.arrowCursor, Background: app.darkBrush, ClassName: menuClass}
	if res, _, _ := pRegister.Call(uintptr(unsafe.Pointer(&menuWC))); res == 0 {
		showErr(0, "语言菜单初始化失败")
		return
	}
	cls := wide("OPPO_DV_REPAIR_NATIVE_GUI_30")
	wc := WNDCLASSEX{Size: uint32(unsafe.Sizeof(WNDCLASSEX{})), Style: 3, WndProc: syscall.NewCallback(wndProc), Instance: inst, Cursor: app.arrowCursor, Background: app.darkBrush, ClassName: cls}
	if res, _, _ := pRegister.Call(uintptr(unsafe.Pointer(&wc))); res == 0 {
		showErr(0, "初始化Windows图形界面失败")
		return
	}
	hwnd, _, _ := pCreate.Call(0, uintptr(unsafe.Pointer(cls)), uintptr(unsafe.Pointer(wide(title))), WS_FIXEDWINDOW, 60, 40, pix(app.scale, 940), pix(app.scale, 940), 0, 0, inst, 0)
	if hwnd == 0 {
		showErr(0, "创建窗口失败")
		return
	}
	app.hwnd = hwnd
	app.instance = inst
	// ICO is embedded into the binary and applied to the taskbar and title bar.
	if len(appIcon) > 6 {
		// ICO directory: choose largest 256px PNG image if available; CreateIconFromResourceEx handles PNG icons on Windows 10/11.
		n := int(binary.LittleEndian.Uint16(appIcon[4:6]))
		chosen := 0
		for i := 0; i < n; i++ {
			pos := 6 + i*16
			if pos+16 > len(appIcon) {
				break
			}
			if appIcon[pos] == 0 {
				chosen = i
				break
			}
		}
		pos := 6 + chosen*16
		if pos+16 <= len(appIcon) {
			sz := int(binary.LittleEndian.Uint32(appIcon[pos+8 : pos+12]))
			off := int(binary.LittleEndian.Uint32(appIcon[pos+12 : pos+16]))
			if off >= 0 && sz > 0 && off+sz <= len(appIcon) {
				icon, _, _ := pIconFromResource.Call(uintptr(unsafe.Pointer(&appIcon[off])), uintptr(sz), 1, 0x30000, 0, 0, 0)
				if icon != 0 {
					pSend.Call(hwnd, WM_SETICON, 1, icon)
					pSend.Call(hwnd, WM_SETICON, 0, icon)
				}
			}
		}
	}
	app.setup()
	pShow.Call(hwnd, SW_SHOW)
	pUpdate.Call(hwnd)
	var m MSG
	for {
		r, _, _ := pGetMessage.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		pTranslate.Call(uintptr(unsafe.Pointer(&m)))
		pDispatch.Call(uintptr(unsafe.Pointer(&m)))
	}
}
