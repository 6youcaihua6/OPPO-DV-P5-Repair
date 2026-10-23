# OPPO Dolby Vision P5 MKV Repair

**0.1BETA · Windows x64 · [下载测试版 / Download](https://github.com/6youcaihua6/OPPO-DV-P5-Repair/releases/tag/v0.1.0-beta.1)**

有些杜比视界 P5 视频，在电脑上能打开，放到安装了 [fandangos/oppo-dv-mkv](https://github.com/fandangos/oppo-dv-mkv) 补丁的 OPPO 播放器上却播放不正常。本工具针对其中一种原因：**视频内容还在，但 MKV 中用于告诉播放器“该怎样读取这条视频”的头信息不完整。**

它借助 FFmpeg 和 MKVToolNix，把视频取出来，再重新装进一个新的 MKV，尝试补全这部分信息。整个过程不重新压制画面，原文件也会保留。

> 它处理的是部分文件的结构问题，不是所有杜比视界播放故障都能用它解决。

## 为什么重新封装可能解决问题？

可以把 MKV 理解为一个盒子：里面装着视频、音轨、字幕等内容，盒子上还记录了播放器读取这些内容所需的信息。

HEVC 视频需要一组解码参数，名字叫 **VPS、SPS、PPS**。有些文件把这些参数留在视频数据里面，但 MKV 对应的视频配置头（**hvcC**）里没有完整记录。不同播放软件、播放器对这种文件的处理方式可能不同，因此“电脑上能播”不能证明这部分头信息完整。

本工具尝试做的，是**利用文件中仍然存在的信息，重新整理封装头**。如果问题确实出在这里，重新封装后就有可能改善兼容性。

它不能凭空制造已经丢失的视频参数，也不会修补损坏画面。文件本身缺少必要数据、问题来自播放器设置或其他原因时，这条方法可能无效。

## 软件具体做了什么？

1. **先检查文件。** 用 ffprobe 读取视频编码、杜比视界 Profile，以及 hvcC 和参数头情况。已具备完整头信息的 P5 文件会跳过修复；不属于修复范围的文件会给出提示。
2. **取出原视频。** 用 FFmpeg 的视频复制模式导出 HEVC，并通过 `hevc_mp4toannexb` 整理码流格式。这一步不重新编码画面。
3. **重新装回 MKV。** 用 MKVToolNix 从导出的 HEVC 重新建立视频轨和配置头，同时从原文件带回音轨、字幕、章节及附件。
4. **保留并核对播放时间。** 提取原视频逐帧时间戳用于重新封装，修正默认帧时长信息，并对输出时间戳进行对照。无法取得必要时间戳时会停止修复。
5. **检查新文件。** 核对参数头、分辨率、DV 标志、轨道数量及时间戳等信息，写入报告。检查失败会明确标记，不能当作修复成功。

**这里的“保留画质”，指视频不经过有损重编码；不表示输出文件与原文件逐字节相同。** 重新封装会改变文件结构和部分封装信息。

技术细节可查阅 [FFmpeg 码流过滤器说明](https://ffmpeg.org/ffmpeg-bitstream-filters.html#hevc_005fmp4toannexb)与 [mkvmerge 使用说明](https://mkvtoolnix.download/doc/mkvmerge.html)。这些工具负责提取和封装；本项目把检查、工具调用、批处理和结果报告整合成 Windows 图形界面。

## 它和原作者的播放器补丁是什么关系？

[fandangos/oppo-dv-mkv](https://github.com/fandangos/oppo-dv-mkv) 在播放器端提供杜比视界 MKV 播放相关修改；本工具在 Windows 上处理部分有问题的视频文件。**两者是独立项目。** 感谢原作者的工作，本工具不包含或安装该补丁，也不代表原作者、OPPO 或 Dolby 官方。

说明该使用场景，不等于已经验证所有补丁版本、播放器和显示设备组合。

## 怎样使用？

1. 在 [Releases](https://github.com/6youcaihua6/OPPO-DV-P5-Repair/releases/tag/v0.1.0-beta.1) 下载 **`OPPO_DV_P5_Repair_0.1BETA_Windows_x64.zip`**，解压运行。`Source code` 是源码，不是可直接运行的程序。
2. 下载并安装或解压下方两套工具。程序会尝试自动查找；找不到时，在“工具路径”中选择对应 EXE。
3. 拖入一个 MKV，或整个文件夹。文件夹会包含子文件夹中的 MKV，多个文件按顺序处理。
4. 建议先点 **“只读自检”** 看检查结果；需要处理且属于修复范围时，再点 **“一键修复 + 自检”**。
5. 每批会创建独立结果文件夹：`Videos` 放输出视频，`Reports` 放报告。报告采用任务开始时选择的界面语言。

原文件不覆盖，输出保留原文件的修改时间。请预留足够磁盘空间，处理长视频时会产生较大的临时文件。

### 需要下载的工具

| 工具 | 下载与选择 |
| --- | --- |
| FFmpeg | [Windows 下载页](https://www.gyan.dev/ffmpeg/builds/)，选择 release essentials ZIP，解压后使用 `bin/ffmpeg.exe` 和 `bin/ffprobe.exe`。 |
| MKVToolNix | [官方下载页](https://mkvtoolnix.download/downloads.html#windows)，选择 Windows 64-bit 安装版或便携版，需有 `mkvmerge.exe`、`mkvextract.exe`、`mkvpropedit.exe`。 |

保留完整工具文件夹，避免只复制 EXE 导致运行库缺失。MKVToolNix 微软商店版不适用于本程序的命令行调用。发布包不附带上述工具。

## 哪些情况不要误以为已经修好？

- **头信息本来完整：** 软件会跳过，不会为了生成一个新文件而重复封装。
- **不是 P5，或有多条主视频轨：** 不在当前自动修复范围。封面附件不算主视频轨。
- **码流损坏、参数确实丢失：** 重新封装不能恢复不存在的内容。
- **报告显示通过：** 只表示已执行的结构检查通过，不代表整部视频完整解码、画面颜色、实机杜比视界播放或音画同步已经验证。请在实际设备上抽查开头、中间和结尾。

当前是公开测试版。29 项顶层自动测试通过，其中包括合成媒体的真实 FFmpeg/MKVToolNix 集成检查；尚未完成全部桌面交互和 OPPO 实机播放验收。详见 [测试记录](TESTING.md)。

## English

Some Dolby Vision P5 MKVs may play on a computer but behave differently on an OPPO player using the [fandangos/oppo-dv-mkv](https://github.com/fandangos/oppo-dv-mkv) modification. This Windows tool targets one possible cause: incomplete video configuration headers in the MKV.

The video data may still contain the HEVC parameters needed to read it, while the container header does not list them completely. This tool copies the video out with FFmpeg and remuxes it with MKVToolNix so that the video configuration can be rebuilt from available data. It does not re-encode the picture or invent missing video data.

The app checks the source first, skips P5 files with complete headers, preserves the original file, carries over non-video tracks and attachments, and reuses the original video timestamps. It then checks the output structure and compares timestamps. A failed check is reported as a failure.

Download the **Windows_x64.zip** from [Releases](https://github.com/6youcaihua6/OPPO-DV-P5-Repair/releases/tag/v0.1.0-beta.1). Install FFmpeg and MKVToolNix using the links above, select their executables if needed, and drag in your MKV files or folders. Each batch creates separate `Videos` and `Reports` folders. Reports use the UI language selected at batch start.

This project is independent of the player-side modification and does not install it. It does not convert other DV profiles to P5 or fix every playback problem. Structural checks do not certify full decoding, correct colors, actual Dolby Vision playback or A/V sync. Check the beginning, middle and end on your player and display.

## Build

Requires a compatible Go toolchain on Windows. The current release was built with Go 1.27.3 for Windows amd64.

```powershell
./build.ps1
```

Output is written to `dist/`. FFmpeg and MKVToolNix are separate dependencies.

## Feedback

提交问题时，请提供 Windows 和工具版本、处理模式、错误信息及报告相关内容，并去掉私人路径。请区分“处理时报错”和“在设备上播放异常”。

When reporting a problem, include Windows/tool versions, the selected mode, error text and relevant report details. Remove private file paths and distinguish processing failures from playback problems.

## Licensing

No open-source license has been selected for this initial publication. Public visibility alone does not grant a general license to modify or redistribute this project. Third-party product names and assets remain subject to their respective rights.
