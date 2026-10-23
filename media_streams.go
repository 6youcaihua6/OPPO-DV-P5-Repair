//go:build windows

package main

import "fmt"

func countMediaStreams(r *Report, streams []Stream) {
	for _, s := range streams {
		switch s.Type {
		case "video":
			if s.Disposition["attached_pic"] == 1 {
				r.AttachedPicNum++
				continue
			}
			r.VideoNum++
			if r.VideoNum == 1 {
				r.Vid = s
			}
		case "audio":
			r.AudioNum++
		case "subtitle":
			r.SubNum++
		}
	}
}

// Use the selected real video stream's absolute index, not v:0, which can
// designate a cover image in files where attached pictures come first.
func videoMap(r Report) string { return fmt.Sprintf("0:%d", r.Vid.Index) }
