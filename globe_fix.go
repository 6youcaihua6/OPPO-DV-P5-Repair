//go:build windows

package main

import "math"

// The globe ends at y=53. The old description started at y=51 and
// its opaque STATIC background covered the bottom two rows on first paint.
const headerDescriptionY = 58

func globeGeometry(w, h int32, scale float64) (pad, icon int32) {
	pad = int32(math.Max(2, math.Round(2*scale)))
	icon = int32(math.Round(32 * scale))
	available := w - 2*pad
	if h-2*pad < available {
		available = h - 2*pad
	}
	if icon > available {
		icon = available
	}
	return
}

func (g *GUI) customCursorTarget(target uintptr) bool {
	return target != 0 && (target == g.hwnd || target == g.globeHwnd || target == g.languagePopup || target == g.progressHwnd)
}

func (g *GUI) resetGlobeVisuals() {
	g.globePhase = 0
	g.globePressed = false
	if g.globeHwnd != 0 {
		pKillTimer.Call(g.globeHwnd, 301)
		pInvalidate.Call(g.globeHwnd, 0, 0)
	}
}
