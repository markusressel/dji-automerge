package internal

import (
	"dji-automerge/internal/util"
	"fmt"
	"github.com/vitali-fedulov/images4"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	Mp4BinaryFileName = "mp4_merge-linux64"
	JoinedSuffix      = "_joined"
)

var (
	tmpDir = "/tmp/dji-automerge"
)

func Process(inputPath string, outputPath string) error {
	checkPrerequisites()

	tmpDir = "/tmp/dji-automerge"

	_, err := createTmpDir(tmpDir)
	if err != nil {
		return err
	}

	fmt.Printf("Searching for videos in %v\n", inputPath)
	files, err := getInputFiles(inputPath)
	if err != nil {
		return err
	}

	fmt.Printf("Found %v files\n", len(files))
	matchingVideos, err := matchInputFiles(files)
	if err != nil {
		return err
	}

	for _, group := range matchingVideos {
		fmt.Printf("Joining %v videos\n", len(group.Parts))
		err = joinVideosInGroup(group, outputPath)
		if err != nil {
			return err
		}

		fmt.Printf("Moving source files to %v\n", outputPath)
		err = moveSourceFilesInGroup(group, outputPath)
		if err != nil {
			return err
		}
	}

	// cleanup
	//_, err = removeTmpDir(tmpDir)

	return err
}

func checkPrerequisites() {
	_, err := exec.LookPath("ffmpeg")
	if err != nil {
		fmt.Println("ffmpeg not found, please install ffmpeg first.")
		os.Exit(1)
	}
}

func createTmpDir(dir string) (string, error) {
	return util.ExecCommand("mkdir", "-p", dir)
}

func removeTmpDir(path string) (string, error) {
	return util.ExecCommand("rm", "-r", path)
}

func moveSourceFilesInGroup(group VideoGroup, path string) error {
	sourcesPath := path + "/Sources"

	err := os.MkdirAll(sourcesPath, os.ModePerm)
	if err != nil {
		return err
	}

	for _, part := range group.Parts {
		// move file to path
		targetPath := sourcesPath + "/" + filepath.Base(part.Path)

		err = os.Rename(part.Path, targetPath)
		if err != nil {
			return err
		}
	}
	return nil
}

func joinVideosInGroup(group VideoGroup, path string) error {
	fmt.Printf("Joining videos: %v", group.Parts)
	outputFileName := group.Parts[0].Path
	outputFileName = outputFileName[:len(outputFileName)-4] + "_merged.mp4"

	err := mergeVideos(outputFileName, group.Parts)
	if err != nil {
		return err
	}
	return nil
}

func mergeVideos(name string, parts []VideoData) error {
	pathToMp4Merge, err := getMp4MergeBinaryPath()
	if err != nil {
		return err
	}

	partFiles := make([]string, 0)
	for _, part := range parts {
		partFiles = append(partFiles, part.Path)
	}

	var args []string
	args = append(args, "-o", name)
	args = append(args, partFiles...)

	// mp4-merge -o output.mp4 part1.mp4 part2.mp4 part3.mp4
	_, err = util.ExecCommand(pathToMp4Merge, args...)
	if err != nil {
		return err
	}
	return nil
}

func getMp4MergeBinaryPath() (string, error) {
	// check if mp4-merge exists
	mp4BinaryFileNameFromPackage := "mp4-merge"

	path, err := exec.LookPath(mp4BinaryFileNameFromPackage)
	if err != nil {
		return path, nil
	}

	pathToMp4Merge := filepath.Join(tmpDir, Mp4BinaryFileName)
	info, err := os.Stat(pathToMp4Merge)
	if err == nil && !info.IsDir() {
		return path, nil
	}

	fmt.Printf("mp4-merge not found, downloading to %v...\n", tmpDir)
	err = downloadMp4Merge(pathToMp4Merge)
	if err != nil {
		return "", err
	} else {
		path = pathToMp4Merge
	}
	return path, nil
}

func downloadMp4Merge(targetPath string) error {
	url := "https://github.com/gyroflow/mp4-merge/releases/latest/download/" + Mp4BinaryFileName
	pathToMp4Merge, err := filepath.Abs(targetPath)
	if err != nil {
		return err
	}

	_, err = util.ExecCommand("curl", "-L", "-o", pathToMp4Merge, url)
	if err != nil {
		return err
	}

	_, err = util.ExecCommand("chmod", "+x", pathToMp4Merge)
	return err
}

type VideoGroup struct {
	Parts []VideoData
}

type VideoData struct {
	Path       string
	Size       int64
	FirstFrame string
	LastFrame  string
}

func matchInputFiles(files []string) ([]VideoGroup, error) {
	// TODO: detect video parts using ffmpeg first and last frame

	result := make([]VideoGroup, 0)

	var videoDataItems []VideoData
	for _, file := range files {
		// get first and last frame of each video
		firstFrameFile, err := getFirstFrame(file)
		if err != nil {
			return nil, err
		}

		lastFrameFile, err := getLastFrame(file)
		if err != nil {
			return nil, err
		}

		// get file size
		fileInfo, err := os.Stat(file)
		videoData := VideoData{
			Path:       file,
			Size:       fileInfo.Size(),
			FirstFrame: firstFrameFile,
			LastFrame:  lastFrameFile,
		}

		videoDataItems = append(videoDataItems, videoData)
	}

	var currentVideoGroup *VideoGroup
	for i := 0; i < len(videoDataItems)-1; i++ {
		currentVideoData := videoDataItems[i]
		nextVideoData := videoDataItems[i+1]

		difference, err := compareImages(currentVideoData.LastFrame, nextVideoData.FirstFrame)
		if err != nil {
			return nil, err
		}

		if difference == 0 {
			if currentVideoGroup == nil {
				currentVideoGroup = &VideoGroup{
					Parts: []VideoData{currentVideoData, nextVideoData},
				}
			} else {
				currentVideoGroup.Parts = append(currentVideoGroup.Parts, nextVideoData)
			}
			fmt.Printf("Found match for %v and %v", currentVideoData, nextVideoData)
		} else {
			if currentVideoGroup != nil {
				result = append(result, *currentVideoGroup)
				currentVideoGroup = nil
			}
			fmt.Printf("No match!")
		}
	}
	if currentVideoGroup != nil {
		result = append(result, *currentVideoGroup)
	}

	return result, nil
}

func compareImages(imagePath1, imagePath2 string) (int64, error) {
	// Opening and decoding images. Silently discarding errors.
	img1, err := images4.Open(imagePath1)
	if err != nil {
		return -1, err
	}
	img2, err := images4.Open(imagePath2)
	if err != nil {
		return -1, err
	}

	// Icons are compact hash-like image representations.
	icon1 := images4.Icon(img1)
	icon2 := images4.Icon(img2)

	// Comparison. Images are not used directly.
	// Use func CustomSimilar for different precision.
	if images4.Similar(icon1, icon2) {
		return 0, nil
	} else {
		return 1, nil
	}
}

func getLastFrame(file string) (string, error) {
	// get filename of targetPath
	filename := filepath.Base(file)
	filename = filename + ".lastFrame"
	filename = filename + ".png"

	targetPath := tmpDir + "/" + filename

	_, err := util.ExecCommand("ffmpeg", "-sseof", "-0.3", "-i", file, "-vsync", "0", "-q:v", "31", "-update", "true", targetPath)
	return targetPath, err
}

func getFirstFrame(file string) (string, error) {
	// get filename of targetPath
	filename := filepath.Base(file)
	filename = filename + ".firstFrame"
	filename = filename + ".png"

	targetPath := tmpDir + "/" + filename

	_, err := util.ExecCommand("ffmpeg", "-i", file, "-vf", "scale=iw*sar:ih,setsar=1", "-vframes", "1", targetPath)
	return targetPath, err
}

func getInputFiles(path string) ([]string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	var result []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// ignore non-mp4 files
		if !strings.HasSuffix(strings.ToLower(entry.Name()), ".mp4") {
			continue
		}

		// ignore already joined files
		if strings.HasSuffix(strings.ToLower(entry.Name()), JoinedSuffix+".mp4") {
			continue
		}

		result = append(result, filepath.Join(path, entry.Name()))
	}

	return result, nil
}
