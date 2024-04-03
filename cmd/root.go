package cmd

import (
	"dji-automerge/internal/util"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/vitali-fedulov/images4"
	"os"
	"path/filepath"
	"strings"
)

const (
	Mp4BinaryFileName = "mp4_merge-linux64"

	JoinedSuffix = "_joined"
)

var Input string
var Output string

var TmpDir = "./tmp"

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "dji-automerge",
	Short: "Automatically join video parts.",
	Long:  `Automatically join video parts.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		input := Input
		output := Output

		if len(input) <= 0 {
			input, err = os.Getwd()
			if err != nil {
				return err
			}
		}
		input, err = filepath.Abs(input)
		if err != nil {
			return err
		}

		if len(output) <= 0 {
			output = input
		}
		output, err = filepath.Abs(output)
		if err != nil {
			return err
		}

		TmpDir = input + "/tmp"

		outputPath := input

		_, err = createTmpDir(TmpDir)

		if err != nil {
			return err
		}

		inputPath := input
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

		fmt.Printf("Joining %v videos\n", len(matchingVideos))
		err = joinVideos(matchingVideos, outputPath)
		if err != nil {
			return err
		}

		fmt.Printf("Moving source files to %v\n", outputPath)
		err = moveSourceFiles(matchingVideos, outputPath)
		if err != nil {
			return err
		}

		// cleanup
		_, err = removeTmpDir(TmpDir)

		return err
	},
}

func removeTmpDir(path string) (string, error) {
	return util.ExecCommand("rm", "-r", path)
}

func createTmpDir(dir string) (string, error) {
	return util.ExecCommand("mkdir", "-p", dir)
}

func moveSourceFiles(groups []VideoGroup, path string) error {
	sourcesPath := path + "/Sources"

	if len(groups) <= 0 {
		return nil
	}

	err := os.MkdirAll(sourcesPath, os.ModePerm)
	if err != nil {
		return err
	}

	for _, group := range groups {
		for _, part := range group.Parts {
			// move file to path
			targetPath := sourcesPath + "/" + filepath.Base(part.Path)

			err = os.Rename(part.Path, targetPath)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func joinVideos(videos []VideoGroup, path string) error {
	fmt.Printf("Joining videos: %v", videos)
	for _, group := range videos {
		outputFileName := group.Parts[0].Path
		outputFileName = outputFileName[:len(outputFileName)-4] + "_merged.mp4"

		err := mergeVideos(outputFileName, group.Parts)
		if err != nil {
			return err
		}
	}
	return nil
}

func mergeVideos(name string, parts []VideoData) error {
	// check if mp4-merge exists
	pathToMp4Merge := filepath.Join(TmpDir, Mp4BinaryFileName)
	_, err := util.ExecCommand("which", pathToMp4Merge)
	if err != nil {
		fmt.Printf("mp4-merge not found, downloading...")
		err = downloadMp4Merge(pathToMp4Merge)
		if err != nil {
			return err
		}
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

//
//func compareImageFiles(frame1Path string, frame2Path string) (int64, error) {
//	f1, err := readImageFile(frame1Path)
//	if err != nil {
//		return 0, err
//	}
//	f2, err := readImageFile(frame2Path)
//	if err != nil {
//		return 0, err
//	}
//
//	return fastCompare(f1.(*image.RGBA), f2.(*image.RGBA))
//}
//
//func fastCompare(img1, img2 *image.RGBA) (int64, error) {
//	if img1.Bounds() != img2.Bounds() {
//		return 0, fmt.Errorf("image bounds not equal: %+v, %+v", img1.Bounds(), img2.Bounds())
//	}
//
//	accumError := int64(0)
//
//	for i := 0; i < len(img1.Pix); i++ {
//		accumError += int64(sqDiffUInt8(img1.Pix[i], img2.Pix[i]))
//	}
//
//	return int64(math.Sqrt(float64(accumError))), nil
//}
//
//func sqDiffUInt8(x, y uint8) uint64 {
//	d := uint64(x) - uint64(y)
//	return d * d
//}
//
//func readImageFile(frame string) (image.Image, error) {
//	reader, err := os.Open(frame)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer reader.Close()
//	m, _, err := image.Decode(reader)
//	if err != nil {
//		return nil, err
//	}
//	return m, nil
//}

func getLastFrame(file string) (string, error) {
	// get filename of targetPath
	filename := filepath.Base(file)
	filename = filename + ".lastFrame"
	filename = filename + ".png"

	targetPath := TmpDir + "/" + filename

	_, err := util.ExecCommand("ffmpeg", "-sseof", "-0.3", "-i", file, "-vsync", "0", "-q:v", "31", "-update", "true", targetPath)
	return targetPath, err
}

func getFirstFrame(file string) (string, error) {
	// get filename of targetPath
	filename := filepath.Base(file)
	filename = filename + ".firstFrame"
	filename = filename + ".png"

	targetPath := TmpDir + "/" + filename

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

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the RootCmd.
func Execute() {
	RootCmd.PersistentFlags().StringVarP(
		&Input,
		"input", "i",
		"./",
		"Input directory",
	)

	RootCmd.PersistentFlags().StringVarP(
		&Output,
		"output", "o",
		"./",
		"Output directory",
	)

	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {

}
