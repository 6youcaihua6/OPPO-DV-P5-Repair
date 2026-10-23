package main

import "testing"

func TestSponsorContentFitsWindow(t *testing.T) {
	for _, size := range [][2]int32{{798, 441}, {1197, 680}, {1596, 890}, {600, 400}, {790, 300}} {
		scale := sponsorContentScale(size[0], size[1])
		if float64(798)*scale > float64(size[0])+0.01 || float64(434)*scale > float64(size[1])+0.01 {
			t.Fatalf("content exceeds client area: %v", size)
		}
	}
}
