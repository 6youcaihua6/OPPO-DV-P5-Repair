//go:build windows

package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCoverAttachmentRoundTrip(t *testing.T) {
	if os.Getenv("P5_INTEGRATION") != "1" || os.Getenv("P5_MKV_DIR") == "" {
		t.Skip("real tools not enabled")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	d := t.TempDir()
	base := filepath.Join(d, "plain.mkv")
	src := filepath.Join(d, "with-cover.mkv")
	ff := os.Getenv("P5_FFMPEG")
	fp := os.Getenv("P5_FFPROBE")
	mkv := os.Getenv("P5_MKV_DIR")
	args := []string{"-v", "error", "-nostdin", "-f", "lavfi", "-i", "testsrc2=size=128x72:rate=24", "-f", "lavfi", "-i", "sine=frequency=440:sample_rate=48000", "-map", "0:v:0"}
	for i := 0; i < 12; i++ {
		args = append(args, "-map", "1:a:0")
	}
	args = append(args, "-t", "1", "-c:v", "libx265", "-x265-params", "log-level=error:pools=1", "-c:a", "aac", base)
	if _, e := command(ctx, ff, args...); e != nil {
		t.Fatal(e)
	}
	cover := filepath.Join(d, "cover.png")
	f, e := os.Create(cover)
	if e != nil {
		t.Fatal(e)
	}
	im := image.NewRGBA(image.Rect(0, 0, 8, 8))
	im.Set(0, 0, color.RGBA{G: 255, A: 255})
	e = png.Encode(f, im)
	f.Close()
	if e != nil {
		t.Fatal(e)
	}
	if _, e = command(ctx, filepath.Join(mkv, "mkvmerge.exe"), "-o", src, "--attachment-mime-type", "image/png", "--attachment-name", "cover.png", "--attach-file", cover, base); e != nil {
		t.Fatal(e)
	}
	r, e := probe(ctx, fp, src)
	if e != nil || r.VideoNum != 1 || r.AttachedPicNum != 1 || r.AudioNum != 12 {
		t.Fatal(r.summaryEN(), e)
	}
	before, _ := os.ReadFile(src)
	beforeHash := sha256.Sum256(before)
	output := filepath.Join(d, "output")
	os.Mkdir(output, 0700)
	stage := func(_ string, _ string, _ int64, fn func() error) error { return fn() }
	tools := map[string]string{"ffmpeg": ff, "ffprobe": fp}
	if e = processOne(ctx, src, output, "export", tools, func(string) {}, stage); e != nil {
		t.Fatal(e)
	}
	rawFiles, _ := filepath.Glob(filepath.Join(output, "Videos", "*.hevc"))
	if len(rawFiles) != 1 {
		t.Fatal(rawFiles)
	}
	ts := filepath.Join(d, "timestamps.txt")
	if _, e = command(ctx, filepath.Join(mkv, "mkvextract.exe"), src, "timestamps_v2", "0:"+ts); e != nil {
		t.Fatal(e)
	}
	repaired := filepath.Join(d, "remux.mkv")
	if _, e = command(ctx, filepath.Join(mkv, "mkvmerge.exe"), "-o", repaired, "--timestamps", "0:"+ts, rawFiles[0], "--no-video", src); e != nil {
		t.Fatal(e)
	}
	after, e := probe(ctx, fp, repaired)
	if e != nil || after.VideoNum != 1 || after.AttachedPicNum != 1 || after.AudioNum != 12 {
		t.Fatal(after.summaryEN(), e)
	}
	identify, e := command(ctx, filepath.Join(mkv, "mkvmerge.exe"), "-J", repaired)
	if e != nil {
		t.Fatal(e)
	}
	var j struct {
		Attachments []struct {
			ID int `json:"id"`
		} `json:"attachments"`
	}
	if e = json.Unmarshal([]byte(identify), &j); e != nil || len(j.Attachments) != 1 {
		t.Fatal(identify, e)
	}
	extracted := filepath.Join(d, "retained-cover.png")
	if _, e = command(ctx, filepath.Join(mkv, "mkvextract.exe"), repaired, "attachments", fmt.Sprintf("%d:%s", j.Attachments[0].ID, extracted)); e != nil {
		t.Fatal(e)
	}
	expected, _ := os.ReadFile(cover)
	actual, _ := os.ReadFile(extracted)
	if sha256.Sum256(expected) != sha256.Sum256(actual) {
		t.Fatal("cover content changed")
	}
	originalAfter, _ := os.ReadFile(src)
	if sha256.Sum256(originalAfter) != beforeHash {
		t.Fatal("input modified")
	}
	t.Log("Real-tool sample: 1 main HEVC + 12 audio + 1 PNG cover; export/remux retained all audio and the identical cover, input hash unchanged. Non-DV sample; not a full P5 repair test.")
}
