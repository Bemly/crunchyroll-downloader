package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// mergeEverything merges audio, video and subtitles in a single MKV container
func mergeEverything(videoFile, audioFile, subsFile, outputFile string, subtitlesLang *string, info EpisodeInfo) {
	args := []string{
		"-i", videoFile,
		"-i", audioFile,
	}

	if subsFile != "" {
		args = append(args,
			"-i", subsFile,
			"-c:s", "copy",
			"-metadata:s:s:0", fmt.Sprintf("title=%s", languageNames[*subtitlesLang]),
		)
	}

	args = append(args,
		"-c:v", "copy", "-c:a", "copy",
		"-metadata:g", "title="+fmt.Sprintf("S%02vE%02v - %s", info.EpisodeMetadata.SeasonNumber, info.EpisodeMetadata.EpisodeNumber, info.Title),
		"-metadata:g", "show="+info.EpisodeMetadata.SeriesTitle,
		"-metadata:g", "track="+fmt.Sprintf("%v", info.EpisodeMetadata.EpisodeNumber),
		"-metadata:g", "season_number="+fmt.Sprintf("%v", info.EpisodeMetadata.EpisodeNumber),
		outputFile,
	)

	cmd := exec.Command("./ffmpeg.exe", args...)
	if err := cmd.Run(); err != nil {
		panic(err)
	}

	// Remove stuff
	_ = os.Remove(videoFile)
	_ = os.Remove(audioFile)
	_ = os.Remove(subsFile)

	fmt.Printf("\nDownload finished! Output file: %s\n\n", outputFile)
}

// mergeEverythingBemly merges audio, video and subtitles in a single MP4 container
func mergeEverythingBemly(videoFile, audioFile, subsFile, outputFile string, subtitlesLang *string, info EpisodeInfo) error {
	// 基础输入：视频和音频
	args := []string{
		"-i", videoFile,
		"-i", audioFile,
	}

	// 如果有外部字幕文件，作为第三个输入
	if subsFile != "" {
		args = append(args, "-i", subsFile)
	}

	// 映射你的特定编码参数
	args = append(args,
		"-map", "0:v:0", // 取第一个输入的视频
		"-map", "1:a:0", // 取第二个输入的音频
	)

	if subsFile != "" {
		args = append(args, "-map", "2:s:0") // 取第三个输入的字幕
	}

	args = append(args,
		// 视频编码设置: AMD AV1 硬件编码
		"-c:v", "av1_amf",
		"-usage", "transcoding",
		"-quality", "quality",
		"-preset", "quality",
		"-rc", "qvbr",
		"-qvbr_quality_level", "32",
		"-aq_mode", "caq", // 关键：开启 CAQ (代替报错的 vbaq)
		"-preanalysis", "true", // 开启预分析
		"-pa_lookahead_buffer_depth", "40", // 扫描视野 40 帧
		"-pa_taq_mode", "2", // 时域自适应量化，番剧静态画面大杀器
		"-async_depth", "41", // 配合 lookahead，榨干 7600S 性能
		"-g", "480", // 10秒一个关键帧，优化归档体积

		// 音频编码设置: ALAC 无损
		"-c:a", "alac",
		"-c:s", "mov_text",

		// 语言元数据映射
		"-metadata:s:v:0", "language=chi",
		"-metadata:s:a:0", "language=jpn",
		"-metadata:s:s:0", "language=zho",
	)

	// 剧集元数据映射
	titleTag := fmt.Sprintf("S%02vE%02v - %s", info.EpisodeMetadata.SeasonNumber, info.EpisodeMetadata.EpisodeNumber, info.Title)
	args = append(args,
		"-metadata", "title="+titleTag,
		"-metadata", "show="+info.EpisodeMetadata.SeriesTitle,
		"-metadata", "season_number="+fmt.Sprintf("%v", info.EpisodeMetadata.SeasonNumber),
		"-metadata", "episode_sort="+fmt.Sprintf("%v", info.EpisodeMetadata.EpisodeNumber),
		"-metadata", "episode_id="+fmt.Sprintf("S%02vE%02v", info.EpisodeMetadata.SeasonNumber, info.EpisodeMetadata.EpisodeNumber),
		outputFile,
	)

	cmd := exec.Command("./ffmpeg.exe", args...)

	// 重新编码非常耗时，建议加上进度显示
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("FFmpeg 运行出错: %v\n", err)
		return fmt.Errorf("FFmpeg 运行失败 (集数 %v): %w", info.EpisodeMetadata.EpisodeNumber, err)
	}

	absVideo, _ := filepath.Abs(videoFile)
	absAudio, _ := filepath.Abs(audioFile)
	absSubs, _ := filepath.Abs(subsFile)

	// 清理原文件
	fmt.Printf("\n请自行删除源文件: \nrm %s\n rm %s\nrm %s", absVideo, absAudio, absSubs)
	fmt.Printf("\n转码并合成完成！输出文件: %s\n", outputFile)

	return nil
}
