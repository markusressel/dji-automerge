package internal

import (
	"dji-automerge/internal/util"
	"fmt"
	"github.com/vitali-fedulov/images4"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	Mp4BinaryFileName = "mp4_merge-linux64"
	JoinedSuffix      = "_joined"
)

var (
	// used to store temporary image files
	// and the mp4-merge binary
	tmpDir = "/tmp/dji-automerge"
)

var (
	// IconSize Image resolution of the icon is very small
	// (11x11 pixels), therefore original image details
	// are lost in downsampling, except when source images
	// have very low resolution (e.g. favicons or simple
	// logos). This is useful from the privacy perspective
	// if you are to use generated icons in a large searchable
	// database.
	IconSize = images4.IconSize

	// Cutoff value for color distance.
	colorDiff = 50
	// Cutoff coefficient for Euclidean distance (squared).
	euclCoeff = 0.2
	// Coefficient of sensitivity for Cb/Cr channels vs Y.
	chanCoeff = 2.0

	// Proportion similarity threshold (0%).
	// We expect parts of the same image to have the exact same dimensions.
	thresholdProp = 0.01

	// Euclidean distance threshold (squared) for Y-channel.
	thresholdY = float64(IconSize*IconSize) * float64(colorDiff*colorDiff) * euclCoeff
	// Euclidean distance threshold (squared) for Cb and Cr channels.
	thresholdCbCr = thresholdY * chanCoeff
)

func Process(inputPath string, outputPath string) error {
	checkPrerequisites()

	_, err := createTmpDir()
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
	cleanupTmpDir()

	return err
}

func checkPrerequisites() {
	_, err := exec.LookPath("ffmpeg")
	if err != nil {
		fmt.Println("ffmpeg not found, please install ffmpeg first.")
		os.Exit(1)
	}
}

func createTmpDir() (string, error) {
	return util.ExecCommand("mkdir", "-p", tmpDir)
}

func cleanupTmpDir() {
	// remove all PNG files in tmpDir
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if Mp4BinaryFileName == entry.Name() {
			// keep mp4-merge binary for later runs
			continue
		}

		// ignore non-png files
		if !strings.HasSuffix(strings.ToLower(entry.Name()), ".png") {
			continue
		}

		_ = os.Remove(filepath.Join(tmpDir, entry.Name()))
	}
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
	partNames := make([]string, 0)
	for _, part := range group.Parts {
		partNames = append(partNames, part.Name)
	}
	fmt.Printf("Joining videos: %v\n", partNames)

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

	pathToMp4Merge, err := exec.LookPath(mp4BinaryFileNameFromPackage)
	if err == nil {
		return pathToMp4Merge, nil
	}

	pathToMp4Merge = filepath.Join(tmpDir, Mp4BinaryFileName)
	info, err := os.Stat(pathToMp4Merge)
	if err == nil && !info.IsDir() {
		return pathToMp4Merge, nil
	}

	fmt.Printf("mp4-merge not found, downloading to %v...\n", tmpDir)
	err = downloadMp4Merge(pathToMp4Merge)
	if err != nil {
		return "", err
	} else {
		return pathToMp4Merge, nil
	}
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
	Name       string
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
		if err != nil {
			return nil, err
		}
		videoData := VideoData{
			Path:       file,
			Name:       filepath.Base(file),
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

		similarity, err := compareImages(currentVideoData.LastFrame, nextVideoData.FirstFrame)
		if err != nil {
			return nil, err
		}

		if similarity.Similar() {
			if currentVideoGroup == nil {
				currentVideoGroup = &VideoGroup{
					Parts: []VideoData{currentVideoData, nextVideoData},
				}
			} else {
				currentVideoGroup.Parts = append(currentVideoGroup.Parts, nextVideoData)
			}

			fmt.Printf("Found match for '%v' and '%v' with similarity metrics: (%v) (%v) (%v) \n", currentVideoData.Name, nextVideoData.Name, similarity.Ypercent, similarity.CbPercent, similarity.CrPercent)
		} else {
			if currentVideoGroup != nil {
				result = append(result, *currentVideoGroup)
				currentVideoGroup = nil
			}
			fmt.Printf("No match between '%v' and '%v'\n", currentVideoData.Name, nextVideoData.Name)
		}
	}
	if currentVideoGroup != nil {
		result = append(result, *currentVideoGroup)
	}

	return result, nil
}

type Similarity struct {
	PropMetric            float64
	ProportionsPercentage float64

	Y         float64
	Ypercent  float64
	Cb        float64
	CbPercent float64
	Cr        float64
	CrPercent float64
}

func compareImages(imagePath1, imagePath2 string) (Similarity, error) {
	// Opening and decoding images. Silently discarding errors.
	img1, err := images4.Open(imagePath1)
	if err != nil {
		return Similarity{}, err
	}
	img2, err := images4.Open(imagePath2)
	if err != nil {
		return Similarity{}, err
	}

	// Icons are compact hash-like image representations.
	iconA := images4.Icon(img1)
	iconB := images4.Icon(img2)

	// Comparison. Images are not used directly.
	// Use func CustomSimilar for different precision.

	propMetric := images4.PropMetric(iconA, iconB)
	proportionsPercentage := propMetric / thresholdProp

	m1, m2, m3 := images4.EucMetric(iconA, iconB)

	mp1 := m1 / thresholdY
	mp2 := m2 / thresholdCbCr
	mp3 := m3 / thresholdCbCr

	return Similarity{
		PropMetric:            propMetric,
		ProportionsPercentage: proportionsPercentage,

		Y:         m1,
		Ypercent:  mp1,
		Cb:        m2,
		CbPercent: mp2,
		Cr:        m3,
		CrPercent: mp3,
	}, nil
}

func (s Similarity) Similar() bool {
	propSimilar := s.PropMetric <= thresholdProp
	if !propSimilar {
		return false
	}
	eucSimilar := s.Y < thresholdY && // Luma as most sensitive.
		s.Cb < thresholdCbCr &&
		s.Cr < thresholdCbCr
	return eucSimilar
}

func getLastFrame(file string) (string, error) {
	// get filename of targetPath
	filename := filepath.Base(file)
	filename = filename + ".lastFrame"
	filename = filename + ".png"

	targetPath := tmpDir + "/" + filename

	_, err := os.Stat(targetPath)
	if err == nil {
		err = os.Remove(targetPath)
		if err != nil {
			return "", err
		}
	}

	sseof := 0.3
	for sseof < 3.0 {
		sseofStr := strconv.FormatFloat(sseof, 'f', -1, 64)
		sseofStr = "-" + sseofStr
		_, err = util.ExecCommand("ffmpeg", "-y", "-sseof", sseofStr, "-i", file, "-vsync", "0", "-q:v", "31", "-update", "true", targetPath)
		_, statErr := os.Stat(targetPath)
		if err != nil || statErr != nil {
			sseof += 0.2
		} else {
			break
		}
	}

	return targetPath, err
}

func getFirstFrame(file string) (string, error) {
	// get filename of targetPath
	filename := filepath.Base(file)
	filename = filename + ".firstFrame"
	filename = filename + ".png"

	targetPath := tmpDir + "/" + filename

	_, err := util.ExecCommand("ffmpeg", "-y", "-i", file, "-vf", "scale=iw*sar:ih,setsar=1", "-vframes", "1", targetPath)
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
