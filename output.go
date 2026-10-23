//go:build windows

package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"
)

var seasonEpisode = regexp.MustCompile(`(?i)^(.*?S[0-9]{1,2})[ ._-]*E[0-9]{1,3}(?:[^0-9]|$)`)

func runOutputName(sources []string) string {
	stem := func(p string) string { return strings.TrimSuffix(filepath.Base(p), filepath.Ext(p)) }
	name := stem(sources[0])
	if len(sources) > 1 {
		match := seasonEpisode.FindStringSubmatch(name)
		sameSeason := len(match) > 1
		for _, p := range sources[1:] {
			m := seasonEpisode.FindStringSubmatch(stem(p))
			if len(m) < 2 || !sameSeason || !strings.EqualFold(m[1], match[1]) {
				sameSeason = false
				break
			}
		}
		if sameSeason {
			name = match[1]
		} else {
			parent := filepath.Dir(sources[0])
			sameParent := true
			for _, p := range sources[1:] {
				if !strings.EqualFold(filepath.Dir(p), parent) {
					sameParent = false
					break
				}
			}
			if sameParent {
				name = filepath.Base(parent)
			} else {
				name += "_Batch"
			}
		}
	}
	name = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || strings.ContainsRune(`<>:"/\|?*`, r) {
			return '_'
		}
		return r
	}, name)
	name = strings.Trim(name, " ._-")
	chars := []rune(name)
	if len(chars) > 80 {
		name = string(chars[:80])
	}
	if name == "" {
		name = "Video"
	}
	return name
}

func createRunOutput(root string, sources []string, now time.Time) (string, error) {
	if len(sources) == 0 {
		return "", errors.New("No input files")
	}
	if root == "" {
		root = filepath.Dir(sources[0])
	}
	root, e := filepath.Abs(root)
	if e != nil {
		return "", e
	}
	base := runOutputName(sources) + "_DV_Fix_" + now.Format("20060102_150405")
	for n := 1; n < 10000; n++ {
		name := base
		if n > 1 {
			name += fmt.Sprintf("_%d", n)
		}
		path := filepath.Join(root, name)
		if e := os.Mkdir(path, 0755); e == nil {
			return path, nil
		} else if !os.IsExist(e) {
			return "", e
		}
	}
	return "", errors.New("Cannot allocate a unique output folder")
}

func reportStart(src, dest string, before Report) ([]string, []string) {
	stamp := time.Now().Format("2006-01-02 15:04:05")
	zh := []string{"===== 中文日志 =====", "OPPO Dolby Vision MKV 修复器 " + version, "自检时间：" + stamp, "原始文件：" + src, "导出目录：" + dest, "原文件不做修改", "源文件：" + before.summary()}
	en := []string{"===== English log =====", "OPPO Dolby Vision MKV Repair " + version, "Inspection time: " + stamp, "Original file: " + src, "Output folder: " + dest, "The original file is not modified.", "Source: " + before.summaryEN()}
	return zh, en
}

func bilingualReport(zh, en []string) string {
	return strings.Join(zh, "\n") + "\n\n" + strings.Join(en, "\n") + "\n"
}

func (r Report) summaryEN() string {
	dv := "No Dolby Vision"
	if r.DV != nil {
		dv = fmt.Sprintf("P%d / RPU flag %d", r.DV.Profile, r.DV.RPUPresent)
	}
	return fmt.Sprintf("Codec=%s, DV=%s, hvcC=%d bytes, valid=%t, VPS/SPS/PPS=%d/%d/%d, main video=%d, attached pictures=%d, audio=%d, subtitles=%d, chapters=%d, duration=%.3fs, resolution=%dx%d, average FPS=%s", r.Vid.Codec, dv, r.ExtradataSize, r.HVCCValid, r.VPS, r.SPS, r.PPS, r.VideoNum, r.AttachedPicNum, r.AudioNum, r.SubNum, r.Chapters, r.Duration, r.Vid.Width, r.Vid.Height, r.Vid.AvgFPS)
}

var checkEnglish = map[string]string{
	"主视频轨数量保持": "Main video track count is unchanged", "封面数量保持": "Attached picture count is unchanged",
	"视频仍为HEVC": "Video remains HEVC", "视频分辨率保持": "Video resolution is unchanged", "输出hvcC有VPS/SPS/PPS": "Output hvcC contains VPS/SPS/PPS", "音轨数未减少": "Audio track count is unchanged", "字幕轨道数未减少": "Subtitle track count is unchanged", "章节数未减少": "Chapter count is unchanged", "DV Profile未变化": "Dolby Vision profile is unchanged", "DV RPU存在标志未变化": "Dolby Vision RPU presence flag is unchanged", "文件时长误差≤1秒": "Duration difference is at most 1 second", "平均帧率标记一致": "Average frame rate matches (tolerance < 0.02 fps)",
}

func outputSubdirs(root string) (videos, reports string, err error) {
	videos = filepath.Join(root, "Videos")
	reports = filepath.Join(root, "Reports")
	for _, dir := range []string{videos, reports} {
		if e := os.MkdirAll(dir, 0755); e != nil {
			return "", "", e
		}
	}
	return videos, reports, nil
}
