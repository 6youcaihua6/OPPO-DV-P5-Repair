# OPPO Dolby Vision P5 MKV Repair

**0.1BETA · Windows x64**

A Windows GUI for repairing selected Dolby Vision Profile 5 MKV header issues encountered with OPPO playback setups. This is an independent community project.

Designed for file-preparation workflows with [fandangos/oppo-dv-mkv](https://github.com/fandangos/oppo-dv-mkv). This Windows application invokes FFmpeg and MKVToolNix to process media files; it does not install or contain that player-side modification.

## 中文

针对部分 Dolby Vision Profile 5 MKV 参数头不完整的问题，通过无重编码提取和重新封装尝试修复。支持文件夹拖入、顺序批处理、只读自检和 HEVC 导出。

适用背景是使用 [fandangos/oppo-dv-mkv](https://github.com/fandangos/oppo-dv-mkv) 播放器端补丁的场景。感谢该项目作者的工作。本程序在 Windows 上调用 FFmpeg 和 MKVToolNix 处理文件，不包含或安装播放器补丁；两者是独立项目。此处说明使用场景，不代表已经完成所有补丁版本和设备组合的兼容验证。

### 下载和使用

1. 在本仓库 **Releases** 下载 `OPPO_DV_P5_Repair_0.1BETA_Windows_x64.zip`，解压运行 EXE。
2. 安装或解压下方两套工具。程序尝试自动查找；未找到时在“工具路径”中选择对应 EXE。
3. 拖入 MKV 或文件夹，选择输出位置，再运行自检或修复。

### 依赖

- [FFmpeg Windows builds](https://www.gyan.dev/ffmpeg/builds/)：release essentials ZIP 中包含 `bin/ffmpeg.exe`、`bin/ffprobe.exe`。
- [MKVToolNix](https://mkvtoolnix.download/downloads.html#windows)：选择 Windows 64-bit 安装版或便携版，需包含 `mkvmerge.exe`、`mkvextract.exe`、`mkvpropedit.exe`。请保留完整工具文件夹；微软商店版不适用于本程序的命令行调用。

### 特性

- 递归添加文件夹中的 MKV、去重并逐个处理。
- 原始文件不覆盖；每批创建独立结果目录，`Videos` 存视频，`Reports` 存 TXT 报告。
- 完整 P5 参数头自动跳过修复；多条主视频轨拒绝修复，封面附件不算主视频轨。
- 保留输出修改时间，对修复输出进行结构检查和逐帧 PTS 对照。
- 12 种界面语言；报告使用任务开始时的界面语言。
- 单实例、可取消任务，以及内置的自愿赞助窗口。

### Beta 限制

目前是公开测试版。结构自检通过不代表完整解码、实际 Dolby Vision 播放或音画同步已获验证。请在 OPPO 和显示设备上检查开头、中间、结尾。程序不保证所有 P5 文件都能修复，也不用于将其他 DV Profile 转换成 P5。

本次有 29 项顶层自动测试通过，包括合成视频的真实 FFmpeg/MKVToolNix 集成检查；真实 P5 原片预检为可选测试。尚未完成完整桌面交互和 OPPO 实机播放验收。详见 [TESTING.md](TESTING.md)。

## English

Download the Windows x64 ZIP from **Releases**. Install FFmpeg and MKVToolNix using the links above, then select their executables if automatic discovery does not find them.

The app processes MKV files sequentially, preserves originals, creates separate video/report folders and offers read-only inspection. Reports use the UI language selected when the batch starts. A complete P5 header is skipped; multiple main video tracks are rejected, while attached cover images are handled separately.

This is a beta. Structural checks do not certify full decoding, correct Dolby Vision playback or A/V sync. Test the beginning, middle and end on your actual player and display.

## Build

Requires a compatible Go toolchain on Windows. The current release was built with Go 1.27.3 for Windows amd64.

```powershell
./build.ps1
```

Output is written to `dist/`. The pixel icon, language sprites and sponsor QR assets are embedded. FFmpeg and MKVToolNix are not bundled.

## Feedback

When filing an issue, include your Windows version, tool versions, selected mode, error text and relevant report details. Remove private file paths before posting. State whether the problem occurs during processing or on the playback device.

## Licensing

No open-source license has been selected for this initial publication. Public visibility alone does not grant a general license to modify or redistribute the project. Third-party product names and assets remain subject to their respective rights.
