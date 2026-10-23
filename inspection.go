//go:build windows

package main

import (
	"fmt"
	"strings"
)

// A completed inspection has an explicit finding, independent of I/O success.
type inspectionResult struct{ Status, ZH, EN string }

func (r *inspectionResult) Error() string { return r.ZH + " / " + r.EN }

func inspectReport(r Report) *inspectionResult {
	result := func(status, zh, en string) *inspectionResult { return &inspectionResult{status, zh, en} }
	if r.VideoNum != 1 || r.Vid.Codec != "hevc" {
		return result("inspectUnsupported", "不属于 P5 修复范围：需要单个 HEVC 视频轨。", "Out of P5 repair scope: exactly one HEVC video track is required.")
	}
	if r.DV == nil {
		return result("inspectUnsupported", "不属于 P5 修复范围：未检测到 Dolby Vision 配置。", "Out of P5 repair scope: no Dolby Vision configuration was detected.")
	}
	if r.DV.Profile != 5 {
		return result("inspectUnsupported", fmt.Sprintf("不属于 P5 修复范围：检测到 DV P%d，不是 P5。", r.DV.Profile), fmt.Sprintf("Out of P5 repair scope: detected DV P%d, not P5.", r.DV.Profile))
	}
	var zh, en []string
	if r.DV.RPUPresent != 1 || r.DV.BLPresent != 1 || r.DV.ELPresent != 0 {
		zh = append(zh, "P5 配置标志异常（要求 BL/RPU=1、EL=0）")
		en = append(en, "unexpected P5 configuration flags (requires BL/RPU=1, EL=0)")
	}
	if !r.HVCCValid {
		zh = append(zh, "hvcC 结构不合法或数据不完整")
		en = append(en, "invalid or incomplete hvcC structure")
	}
	if !r.Complete || r.VPS <= 0 || r.SPS <= 0 || r.PPS <= 0 {
		zh = append(zh, fmt.Sprintf("VPS/SPS/PPS 不完整（%d/%d/%d）", r.VPS, r.SPS, r.PPS))
		en = append(en, fmt.Sprintf("incomplete VPS/SPS/PPS (%d/%d/%d)", r.VPS, r.SPS, r.PPS))
	}
	if r.RateNum <= 0 || r.RateDen <= 0 {
		zh = append(zh, "无法确认有效帧率")
		en = append(en, "valid frame rate could not be determined")
	}
	if len(zh) > 0 {
		return result("inspectIssue", "发现问题："+strings.Join(zh, "；")+"。这不是正常通过；可查看报告判断是否尝试修复。", "Issues found: "+strings.Join(en, "; ")+". This is not a pass; review the report before attempting repair.")
	}
	return result("inspectPass", "P5 头信息结构通过；仅检查元数据和参数头，未验证完整解码或 OPPO/电视实播。", "P5 header structure passed; only metadata and parameter headers were inspected. Full decoding and OPPO/TV playback were not verified.")
}

func writeInspection(src, root, reportDir string, r Report, log func(string), reportLanguages ...string) error {
	finding := inspectReport(r)
	zh, en := reportStart(src, root, r)
	zh = append(zh, "检查类型：只读结构自检", fmt.Sprintf("hvcC 合法：%t；VPS/SPS/PPS 数量：%d/%d/%d", r.HVCCValid, r.VPS, r.SPS, r.PPS))
	en = append(en, "Mode: Read-only structural inspection", fmt.Sprintf("hvcC valid: %t; VPS/SPS/PPS counts: %d/%d/%d", r.HVCCValid, r.VPS, r.SPS, r.PPS))
	if r.DV != nil {
		zh = append(zh, fmt.Sprintf("DV 标志：Profile=%d, BL=%d, RPU=%d, EL=%d", r.DV.Profile, r.DV.BLPresent, r.DV.RPUPresent, r.DV.ELPresent))
		en = append(en, fmt.Sprintf("DV flags: Profile=%d, BL=%d, RPU=%d, EL=%d", r.DV.Profile, r.DV.BLPresent, r.DV.RPUPresent, r.DV.ELPresent))
	}
	if r.HVCCError != "" {
		zh = append(zh, "hvcC 详情："+r.HVCCError)
		en = append(en, "hvcC details: structural parsing failed or extradata length did not match the probe record.")
	}
	zh = append(zh, "结论："+finding.ZH, "检查范围：未扫描整条码流，未验证画面色彩、完整解码、实际 Dolby Vision 播放或音画同步。")
	en = append(en, "Finding: "+finding.EN, "Scope: the entire bitstream was not scanned. Color correctness, full decoding, actual Dolby Vision playback and A/V sync were not verified.")
	path := nextNameInDir(src, "_DV_Fix_Report", ".txt", reportDir)
	language := "en"
	if len(reportLanguages) > 0 {
		language = reportLanguages[0]
	}
	if e := txtFile(path, localizedReport(language, src, root, "inspect", "", r, nil, nil, "")); e != nil {
		return fmt.Errorf("%s: %w", pair("无法保存自检报告", "Cannot save inspection report"), e)
	}
	log(event("outcome", event("@"+finding.Status), "", pair(finding.ZH, finding.EN)))
	log(event("report", path))
	return finding
}
