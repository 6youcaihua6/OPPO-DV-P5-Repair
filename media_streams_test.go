//go:build windows

package main

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"
)

func TestAttachedPicturesAreNotMainVideo(t *testing.T) {
	cover := Stream{Index: 0, Type: "video", Codec: "png", Disposition: map[string]int{"attached_pic": 1}}
	main := Stream{Index: 5, Type: "video", Codec: "hevc"}
	for _, streams := range [][]Stream{{cover, main}, {main, cover}} {
		r := Report{}
		countMediaStreams(&r, streams)
		if r.VideoNum != 1 || r.AttachedPicNum != 1 || r.Vid.Index != 5 || videoMap(r) != "0:5" {
			t.Fatal(r)
		}
		r.DV = &DVRecord{Profile: 5, BLPresent: 1, RPUPresent: 1}
		r.RateNum = 24
		r.RateDen = 1
		if e := preflight(r); e != nil {
			t.Fatal("cover must not block P5 preflight", e)
		}
		if f := inspectReport(r); f.Status != "inspectIssue" {
			t.Fatal("missing headers should be reported as an issue", f)
		}
	}
	r := Report{}
	countMediaStreams(&r, []Stream{main, cover, Stream{Index: 9, Type: "video", Codec: "hevc"}})
	if r.VideoNum != 2 || preflight(r) == nil {
		t.Fatal("real multi-video protection lost", r)
	}
	r = Report{}
	countMediaStreams(&r, []Stream{cover})
	if r.VideoNum != 0 {
		t.Fatal("cover-only input counted as main video")
	}
	r = Report{}
	countMediaStreams(&r, []Stream{main, {Index: 7, Type: "video", Codec: "png"}})
	if r.VideoNum != 2 {
		t.Fatal("unmarked PNG video was silently discarded")
	}
}

// Opt-in, read-only verification against the user's file. No output media
// or report is written and no full bitstream scan/repair is performed.
func TestRealP5CoverPreflight(t *testing.T) {
	path := os.Getenv("P5_REAL_INPUT")
	if path == "" {
		t.Skip("P5_REAL_INPUT unset")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	r, e := probe(ctx, os.Getenv("P5_FFPROBE"), path)
	if e != nil {
		t.Fatal(e)
	}
	t.Log(r.summaryEN())
	if r.VideoNum != 1 || r.AttachedPicNum != 1 || r.AudioNum != 12 || r.DV == nil || r.DV.Profile != 5 {
		t.Fatal("unexpected real input metadata")
	}
	if e := preflight(r); e != nil && !errors.Is(e, errSkipped) {
		t.Fatal("real P5 file still blocked", renderMessage("en", e.Error()))
	}
	if f := inspectReport(r); f.Status != "inspectIssue" {
		t.Fatal("missing parameter sets must be an issue, not out-of-scope", f)
	}
}
