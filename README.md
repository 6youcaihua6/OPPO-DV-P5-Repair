# OPPO DV P5 Repair

**0.1BETA · Windows x64**

[Download the Windows app](https://github.com/6youcaihua6/OPPO-DV-P5-Repair/releases/tag/v0.1.0-beta.1) · [中文说明](#中文说明)

## What problem does it address?

This project was created for a specific playback problem: **some Dolby Vision Profile 5 MKV files play correctly in Infuse on Apple TV, Kodi or CoreELEC, but appear as SDR BT.709 with purple and green colors on an OPPO player—even after installing the [oppo-dv-mkv](https://github.com/fandangos/oppo-dv-mkv) mod.**

OPPO DV P5 Repair brings **FFmpeg and MKVToolNix** together in a Windows app to make the file-processing steps available with one click. It aims to resolve this problem for affected files without re-encoding the picture or overwriting the original.

## Thanks to the original author

This project would not exist without the work of Brazilian developer **[Vitor DallAcqua / fandangos](https://github.com/fandangos)** and his **[oppo-dv-mkv](https://github.com/fandangos/oppo-dv-mkv)** project, which provides Dolby Vision MKV playback support on the player side. His work made this project possible.

Our app is a separate Windows utility for preparing affected video files for that playback setup. It does not install the player mod, and is not an official release from its author, OPPO or Dolby.

Thanks also to the **[FFmpeg](https://ffmpeg.org/)** and **[MKVToolNix](https://mkvtoolnix.download/)** developers. These open-source tools do the underlying media processing; this project puts the workflow, checks and batch handling into one interface.

## How can it help?

A video can contain the picture data yet still lack some of the information a particular player needs to read it correctly. Different playback apps handle these files differently, which is why the same file may work in one setup and fail in another.

This app uses **FFmpeg to copy out the original video**, then **MKVToolNix to put it into a new MKV**, rebuilding the relevant playback information from data still present in the file. It brings over the audio, subtitles, chapters and attachments, and preserves the original video timing.

In plain terms: **it reorganizes the file so the player has the information it needs, while keeping the encoded picture.** This can help when incomplete video configuration is the cause. It cannot recover data that is genuinely missing or fix every cause of purple/green playback.

The app checks the file before processing and checks the result afterward. P5 files with complete configuration are skipped. If the output fails a check, it is marked as failed rather than presented as a successful repair.

## Download and use

1. Open [Releases](https://github.com/6youcaihua6/OPPO-DV-P5-Repair/releases/tag/v0.1.0-beta.1) and download **`OPPO_DV_P5_Repair_0.1BETA_Windows_x64.zip`**. Extract it and run the EXE. GitHub's `Source code` downloads are for building the app yourself.
2. Install or extract FFmpeg and MKVToolNix using the links below. The app tries to locate them automatically; select their executables under **Tool paths** if necessary.
3. Drag in an MKV or a folder. Subfolders are included, and files are processed one at a time.
4. Use **Inspect only** to check a file, then **Repair + verify** for files that need repair and are within the supported scope.
5. Each batch gets a new result folder: **Videos** contains output media and **Reports** contains inspection reports. Reports use the interface language selected when the batch starts.

Original files are preserved, and outputs retain the source file's modification time. Allow enough free disk space for temporary files when processing large videos.

### Required tools

| Tool | What to download |
| --- | --- |
| FFmpeg | [Windows builds](https://www.gyan.dev/ffmpeg/builds/): choose the release essentials ZIP. The required files are `bin/ffmpeg.exe` and `bin/ffprobe.exe`. |
| MKVToolNix | [Official downloads](https://mkvtoolnix.download/downloads.html#windows): choose the Windows 64-bit installer or portable package. It must include `mkvmerge.exe`, `mkvextract.exe` and `mkvpropedit.exe`. |

Keep the full tool folders so required libraries remain available. The Microsoft Store edition of MKVToolNix is not suitable for this app's command-line calls. Neither tool is bundled with this release.

## Scope and current status

- Intended for **some Dolby Vision Profile 5 MKV files** used with the player mod above. It does not convert other Dolby Vision profiles to P5.
- Files with multiple main video tracks are rejected to avoid dropping video tracks. An attached cover image is treated separately.
- A report marked as passed means the implemented file checks passed. It does **not** confirm correct colors, full decoding, actual Dolby Vision playback or perfect A/V sync on your equipment. Check the beginning, middle and end on your player and display.
- This is a **public beta**. The release passed 29 top-level automated tests, including synthetic-media tests using real FFmpeg and MKVToolNix. Full desktop interaction and OPPO playback acceptance remain incomplete. See [TESTING.md](TESTING.md).

## Build and feedback

On Windows with a compatible Go toolchain, run `./build.ps1`; output goes to `dist/`. The current release was built with Go 1.27.3 for Windows amd64.

When reporting a problem, include Windows/tool versions, the selected mode, error text and relevant report details. Remove private file paths. Please distinguish a processing error from a problem seen when playing the output.

No open-source license has been selected for this initial publication. The third-party tools have their own licenses; public visibility of this repository does not by itself grant a general license to modify or redistribute this project's code and assets.

---

## 中文说明

### 这个软件解决什么问题？

这个项目针对的是一种具体情况：**部分 Dolby Vision Profile 5 的 MKV 视频，在 Apple TV（ATV）的 Infuse、Kodi 或 CoreELEC 上能正常播放，但 OPPO 安装 [oppo-dv-mkv](https://github.com/fandangos/oppo-dv-mkv) 插件后，播放这些文件时仍显示为 SDR BT.709，画面偏紫、偏绿。**

OPPO DV P5 Repair 借助 **FFmpeg 和 MKVToolNix** 两套开源工具，把处理过程整合成 Windows 下的一键操作，尝试修复这类文件的兼容问题。处理时不重新压制画面，也不覆盖原文件。

### 感谢原作者

首先感谢巴西开发者 **[Vitor DallAcqua / fandangos](https://github.com/fandangos)** 及其 **[oppo-dv-mkv](https://github.com/fandangos/oppo-dv-mkv)** 项目。他在播放器端实现的杜比视界 MKV 播放支持，是本项目诞生的基础。**没有他的工作，就不会有这个项目。**

本软件是在这一使用场景下，为处理部分有问题的视频文件而开发的独立 Windows 工具，不包含或安装播放器插件，也不是原作者、OPPO 或 Dolby 的官方软件。

同时感谢 **[FFmpeg](https://ffmpeg.org/)** 和 **[MKVToolNix](https://mkvtoolnix.download/)** 的开发者。底层的视频处理由这些开源工具完成，本项目负责把操作流程、自检和批处理整合到一个界面中。

### 为什么这样处理可能有用？

视频里有画面数据，不代表文件中供播放器读取的说明信息就是完整的。不同软件对这类文件的处理方式不同，所以同一个文件，在一套设备上能播，换到另一套设备上就可能出问题。

本软件先用 **FFmpeg 取出原视频**，再用 **MKVToolNix 重新保存为一个新的 MKV**，利用文件中仍然存在的数据，补全相关的播放信息。同时带回音轨、字幕、章节和附件，并保留原视频的播放时间信息。

通俗地说，就是：**画面本身不重新压制，把文件重新整理一遍，让播放器能读到需要的信息。** 如果原因正是这部分信息不完整，这种处理就可能改善播放兼容性。但它不能找回真正丢失的数据，也不能解决所有偏紫、偏绿的问题。

软件会在处理前检查文件，处理后再检查结果。相关信息已经完整的 P5 文件会直接跳过；输出检查失败时会明确标记，不会当作修复成功。

### 下载和使用

1. 在 [Releases](https://github.com/6youcaihua6/OPPO-DV-P5-Repair/releases/tag/v0.1.0-beta.1) 下载 **`OPPO_DV_P5_Repair_0.1BETA_Windows_x64.zip`**，解压后运行 EXE。页面上的 `Source code` 是源码，不是直接运行的程序。
2. 下载并安装或解压下方两套工具。软件会尝试自动查找；找不到时，在“工具路径”中选择对应 EXE。
3. 拖入一个 MKV，或整个文件夹。子文件夹中的 MKV 也会加入，多个文件按顺序处理。
4. 可先点 **“只读自检”** 查看情况；文件需要处理且属于支持范围时，再点 **“一键修复 + 自检”**。
5. 每批自动建立独立结果文件夹：**Videos** 放输出视频，**Reports** 放自检报告。报告采用任务开始时选择的界面语言。

原文件保留，输出文件保留原文件的修改时间。处理大体积视频时会产生临时文件，请预留足够磁盘空间。

### 需要下载的工具

| 工具 | 下载与选择 |
| --- | --- |
| FFmpeg | [Windows 下载页](https://www.gyan.dev/ffmpeg/builds/)，选择 release essentials ZIP，解压后使用 `bin/ffmpeg.exe` 和 `bin/ffprobe.exe`。 |
| MKVToolNix | [官方下载页](https://mkvtoolnix.download/downloads.html#windows)，选择 Windows 64-bit 安装版或便携版，需有 `mkvmerge.exe`、`mkvextract.exe`、`mkvpropedit.exe`。 |

请保留完整工具文件夹，避免只复制 EXE 导致运行库缺失。MKVToolNix 微软商店版不适用于本程序的命令行调用。发布包不附带这两套工具。

### 支持范围与当前状态

- 面向安装上述播放器插件后，**部分 P5 MKV 文件**出现的播放问题，不用于将其他杜比视界格式转换成 P5。
- 有多条主视频轨的文件会拒绝自动修复，避免丢失视频轨；封面附件单独识别，不算主视频轨。
- 报告显示通过，只代表软件执行的文件检查通过。**不代表画面颜色、完整解码、实机杜比视界播放或音画同步已经验证。** 请在自己的播放器和显示设备上抽查开头、中间、结尾。
- 当前为**公开测试版**。发布版本有 29 项顶层自动测试通过，包含使用真实 FFmpeg/MKVToolNix 的合成媒体测试；尚未完成全部桌面交互和 OPPO 实机播放验收。详见 [测试记录](TESTING.md)。

### 编译与反馈

在 Windows 上使用兼容的 Go 工具链运行 `./build.ps1`，程序输出到 `dist/`。当前发布版使用 Go 1.27.3 构建，适用于 Windows x64。

反馈问题时，请提供 Windows 和工具版本、处理模式、错误信息及报告相关内容，并去掉私人路径。请区分“处理时报错”和“播放输出文件时异常”。

本项目首次发布尚未选择开源许可证。第三方工具遵循各自的许可证；仓库公开本身不等于授予对本项目代码和素材的一般修改、再分发许可。
