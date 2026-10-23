//go:build windows

package main

import (
	"math"
	"testing"
)

func TestHeaderDoesNotCoverGlobe(t *testing.T) {
	for _, s := range []float64{0.85, 1, 1.25, 1.5, 1.75, 2, 2.5, 3} {
		globeBottom := int(math.Round(10*s)) + int(math.Round(43*s))
		descriptionTop := int(math.Round(headerDescriptionY * s))
		descriptionBottom := descriptionTop + int(math.Round(23*s))
		nextTop := int(math.Round(85 * s))
		if descriptionTop <= globeBottom || descriptionBottom > nextTop {
			t.Fatal(s, globeBottom, descriptionTop, descriptionBottom, nextTop)
		}
	}
}

func TestGlobeUsesActualClientDimensions(t *testing.T) {
	for _, size := range [][2]int32{{43, 43}, {107, 106}, {108, 107}, {36, 35}, {129, 128}} {
		p, icon := globeGeometry(size[0], size[1], 2.5)
		if p < 2 || icon <= 0 || icon+2*p > size[0] || icon+2*p > size[1] {
			t.Fatal(size, p, icon)
		}
	}
}

func TestArrowCursorDoesNotOverrideEditors(t *testing.T) {
	g := GUI{hwnd: 1, globeHwnd: 2, languagePopup: 3, progressHwnd: 4, pathEdit: 5, logEdit: 6, outputEdit: 7}
	for _, h := range []uintptr{1, 2, 3, 4} {
		if !g.customCursorTarget(h) {
			t.Fatal(h)
		}
	}
	for _, h := range []uintptr{0, 5, 6, 7, 99} {
		if g.customCursorTarget(h) {
			t.Fatal("native editor cursor overridden", h)
		}
	}
}
