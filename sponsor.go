package main

import (
	"bytes"
	"embed"
	"image/png"
	"math"
	"runtime"
	"syscall"
	"unsafe"
)

//go:embed sponsor_assets/*.png
var sponsorAssets embed.FS

const idSponsor = 180

var coffeeWindow, supportWindow uintptr
var coffeePhase int
var coffeeDown, sponsorRegistered bool
var sponsorQR [3]sprite
var sponsorCaptionFont uintptr

func refreshSponsorCaptionFont() {
	if coffeeWindow == 0 {
		return
	}
	old := sponsorCaptionFont
	sponsorCaptionFont = makeFont(app.scale * 0.8)
	pSend.Call(app.labels["sponsorCaption"], WM_SETFONT, sponsorCaptionFont, 1)
	if old != 0 {
		pDeleteFont.Call(old)
	}
}

func setupSponsor(g *GUI) {
	if !sponsorRegistered {
		for _, name := range []string{"P5_COFFEE", "P5_SUPPORT"} {
			wc := WNDCLASSEX{Size: uint32(unsafe.Sizeof(WNDCLASSEX{})), WndProc: syscall.NewCallback(sponsorProc), Instance: g.instance, Cursor: g.arrowCursor, Background: g.darkBrush, ClassName: wide(name)}
			pRegister.Call(uintptr(unsafe.Pointer(&wc)))
		}
		for i, name := range []string{"stripe", "wechat", "alipay"} {
			data, err := sponsorAssets.ReadFile("sponsor_assets/" + name + ".png")
			if err != nil {
				continue
			}
			im, err := png.Decode(bytes.NewReader(data))
			if err != nil {
				continue
			}
			b := im.Bounds()
			sp := sprite{w: int32(b.Dx()), h: int32(b.Dy()), bgra: make([]byte, b.Dx()*b.Dy()*4)}
			for y := 0; y < b.Dy(); y++ {
				for x := 0; x < b.Dx(); x++ {
					r, g, b, _ := im.At(x, y).RGBA()
					n := (y*int(sp.w) + x) * 4
					sp.bgra[n] = byte(b >> 8)
					sp.bgra[n+1] = byte(g >> 8)
					sp.bgra[n+2] = byte(r >> 8)
					sp.bgra[n+3] = 255
				}
			}
			sponsorQR[i] = sp
		}
		sponsorRegistered = true
	}
	coffeeWindow = g.widget("P5_COFFEE", "", WS_TABSTOP, 819, 10, 43, 43, idSponsor)
	g.labels["sponsorCaption"] = g.widget("STATIC", sponsorMessage(0), 2, 606, 24, 210, 23, 0)
	refreshSponsorCaptionFont()
}
func (g *GUI) openSponsor() {
	if supportWindow != 0 {
		pShow.Call(supportWindow, 5)
		user32.NewProc("SetForegroundWindow").Call(supportWindow)
		return
	}
	var owner RECT
	pGetRect.Call(g.hwnd, uintptr(unsafe.Pointer(&owner)))
	w, h := int32(pix(g.scale, 810)), int32(pix(g.scale, 480))
	x, y := owner.Left+(owner.Right-owner.Left-w)/2, owner.Top+(owner.Bottom-owner.Top-h)/2
	if work, ok := currentWorkRect(owner); ok {
		if w > work.Right-work.Left {
			w = work.Right - work.Left
		}
		if h > work.Bottom-work.Top {
			h = work.Bottom - work.Top
		}
		x = owner.Left + (owner.Right-owner.Left-w)/2
		y = owner.Top + (owner.Bottom-owner.Top-h)/2
		if x < work.Left {
			x = work.Left
		}
		if x+w > work.Right {
			x = work.Right - w
		}
		if y < work.Top {
			y = work.Top
		}
		if y+h > work.Bottom {
			y = work.Bottom - h
		}
	}
	supportWindow, _, _ = pCreate.Call(0, uintptr(unsafe.Pointer(wide("P5_SUPPORT"))), uintptr(unsafe.Pointer(wide(sponsorMessage(1)))), 0x00C80000, uintptr(x), uintptr(y), uintptr(w), uintptr(h), g.hwnd, 0, g.instance, 0)
	if supportWindow == 0 {
		showErr(g.hwnd, "Could not create support window")
		return
	}
	pShow.Call(supportWindow, 5)
}
func sponsorPaint(hwnd uintptr) {
	var ps PAINTSTRUCT
	dc, _, _ := pBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
	if dc == 0 {
		return
	}
	defer pEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
	var rc RECT
	pGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&rc)))
	fill(dc, rc, rgb(23, 32, 45))
	if hwnd == coffeeWindow {
		s := int32(pix(app.scale, 1))
		if s < 1 {
			s = 1
		}
		fill(dc, rc, rgb(90, 125, 141))
		fill(dc, RECT{s, s, rc.Right - s, rc.Bottom - s}, rgb(25, 34, 47))
		rows := []string{"    s   s     ", "     s s      ", "              ", "  WWWWWWWW    ", "  WccccccWHH  ", "  WccccccW HH ", "  WccccccW HH ", "  WccccccWHH  ", "   WWWWWW     ", "              ", " WWWWWWWWWW   "}
		unit := int32(pix(app.scale, 2))
		x0 := (rc.Right - 14*unit) / 2
		y0 := (rc.Bottom-11*unit)/2 + int32(coffeePhase)*s
		for y, row := range rows {
			for x, c := range row {
				var col uintptr
				switch c {
				case 'W', 'H':
					col = rgb(230, 220, 195)
				case 'c':
					col = rgb(151, 91, 51)
				case 's':
					col = rgb(167, 193, 201)
				default:
					continue
				}
				x1, y1 := x0+int32(x)*unit, y0+int32(y)*unit
				fill(dc, RECT{x1, y1, x1 + unit, y1 + unit}, col)
			}
		}
		return
	}
	pSetBkMode.Call(dc, 1)
	scale := sponsorContentScale(rc.Right, rc.Bottom)
	font := makeFont(scale)
	old, _, _ := pSelectObject.Call(dc, font)
	defer func() { pSelectObject.Call(dc, old); pDeleteFont.Call(font) }()
	sc := func(v int) int32 { return int32(pix(scale, v)) }
	text := func(t string, y0, y1 int32, col uintptr) {
		pSetTextColor.Call(dc, col)
		r := RECT{0, y0, rc.Right, y1}
		pDrawText.Call(dc, uintptr(unsafe.Pointer(wide(t))), ^uintptr(0), uintptr(unsafe.Pointer(&r)), DT_CENTER|DT_VCENTER|DT_SINGLELINE)
	}
	text(sponsorMessage(2), sc(12), sc(43), rgb(230, 239, 242))
	text(sponsorMessage(3), sc(45), sc(73), rgb(186, 205, 215))
	names := []string{"Stripe", "微信支付 / WeChat Pay", "支付宝 / Alipay"}
	colors := []uintptr{rgb(99, 91, 255), rgb(7, 185, 96), rgb(22, 119, 255)}
	for i, sp := range sponsorQR {
		x := sc(18 + i*258)
		y := sc(91)
		w := sc(244)
		fill(dc, RECT{x, y, x + w, sc(382)}, rgb(255, 255, 255))
		fill(dc, RECT{x, y, x + w, y + sc(5)}, colors[i])
		pSetTextColor.Call(dc, colors[i])
		r := RECT{x, y + sc(10), x + w, y + sc(40)}
		pDrawText.Call(dc, uintptr(unsafe.Pointer(wide(names[i]))), ^uintptr(0), uintptr(unsafe.Pointer(&r)), DT_CENTER|DT_VCENTER|DT_SINGLELINE)
		if len(sp.bgra) > 0 {
			hdr := BITMAPINFOHEADER{Size: 40, Width: sp.w, Height: -sp.h, Planes: 1, BitCount: 32}
			pStretchDIBits.Call(dc, uintptr(x+sc(12)), uintptr(y+sc(49)), uintptr(sc(220)), uintptr(sc(220)), 0, 0, uintptr(sp.w), uintptr(sp.h), uintptr(unsafe.Pointer(&sp.bgra[0])), uintptr(unsafe.Pointer(&hdr)), 0, SRCCOPY)
			runtime.KeepAlive(sp.bgra)
		}
	}
	text(sponsorMessage(4), sc(389), sc(422), rgb(186, 205, 215))
}
func sponsorProc(hwnd uintptr, msg uint32, wp, lp uintptr) uintptr {
	switch msg {
	case WM_PAINT:
		sponsorPaint(hwnd)
		return 0
	case WM_ERASEBKGND:
		return 1
	case WM_LBUTTONDOWN:
		if hwnd == coffeeWindow {
			pKillTimer.Call(hwnd, 401)
			user32.NewProc("SetFocus").Call(hwnd)
			coffeeDown = true
			coffeePhase = 2
			user32.NewProc("SetCapture").Call(hwnd)
			pInvalidate.Call(hwnd, 0, 0)
			return 0
		}
	case WM_LBUTTONUP:
		if hwnd == coffeeWindow && coffeeDown {
			var rc RECT
			pGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&rc)))
			x, y := int32(int16(lp&65535)), int32(int16(lp>>16))
			coffeeDown = false
			user32.NewProc("ReleaseCapture").Call()
			coffeePhase = 2
			pSetTimer.Call(hwnd, 401, 45, 0)
			if x >= 0 && y >= 0 && x < rc.Right && y < rc.Bottom {
				app.openSponsor()
			}
			return 0
		}
	case 0x0215:
		if hwnd == coffeeWindow {
			pKillTimer.Call(hwnd, 401)
			coffeeDown = false
			coffeePhase = 0
			pInvalidate.Call(hwnd, 0, 0)
			return 0
		}
	case WM_TIMER:
		if hwnd == coffeeWindow {
			if coffeePhase > 0 {
				coffeePhase--
			} else {
				pKillTimer.Call(hwnd, 401)
			}
			pInvalidate.Call(hwnd, 0, 0)
			return 0
		}
	case WM_KEYDOWN:
		if hwnd == coffeeWindow && (wp == 32 || wp == 13) {
			coffeePhase = 2
			pSetTimer.Call(hwnd, 401, 45, 0)
			app.openSponsor()
			return 0
		}
		if wp == 27 && hwnd == supportWindow {
			pDestroy.Call(hwnd)
			return 0
		}
	case WM_CLOSE:
		pDestroy.Call(hwnd)
		return 0
	case 2:
		if hwnd == coffeeWindow {
			pKillTimer.Call(hwnd, 401)
			coffeeWindow = 0
			if sponsorCaptionFont != 0 {
				pDeleteFont.Call(sponsorCaptionFont)
				sponsorCaptionFont = 0
			}
		}
		if hwnd == supportWindow {
			supportWindow = 0
		}
		return 0
	}
	r, _, _ := pDefProc.Call(hwnd, uintptr(msg), wp, lp)
	return r
}

// Fit the content to this window, independently of the main window's DPI.
func sponsorContentScale(width, height int32) float64 {
	return math.Max(0.01, math.Min(float64(width)/798, float64(height)/434))
}
