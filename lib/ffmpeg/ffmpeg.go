package ffmpeg

import (
	"fmt"
	"os"
	"os/exec"
	"path"
)

type FMHelper struct{}

func NewFMHelper() *FMHelper {
	return &FMHelper{}
}

func (ff *FMHelper) ConvertSegmentsToMp4(
	inputFiles []string,
	outputFile string,
	outputPath string,
) error {
	concatFile, err := createTempConcatFile(inputFiles, outputPath)
	if err != nil {
		return err
	}
	combinedTsFile, err := createCombinedTsFile(concatFile, outputPath)
	if err != nil {
		return err
	}
	return createMp4FromTs(combinedTsFile, outputFile)
}

func createTempConcatFile(
	inputFiles []string,
	outputPath string,
) (string, error) {
	concatFilePath := path.Join(outputPath, "concatFile.txt")
	file, err := os.Create(concatFilePath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	for _, val := range inputFiles {
		file.WriteString(fmt.Sprintf("file '%v'\n", val))
	}
	return concatFilePath, nil
}

func executeCommand(command string) error {
	_, err := exec.Command("bash", "-c", command).Output()
	return err
}

func createCombinedTsFile(
	concatFile string,
	outputPath string,
) (string, error) {
	combinedTsFilePath := path.Join(outputPath, "combined.ts")
	return combinedTsFilePath, executeCommand(
		fmt.Sprintf("ffmpeg -f concat -safe 0 -i %v -c copy %v", concatFile, combinedTsFilePath),
	)
}

func createMp4FromTs(
	combinedTsFile string,
	outputMp4File string,
) error {
	return executeCommand(
		fmt.Sprintf("ffmpeg -i %v -acodec copy -vcodec copy %v", combinedTsFile, outputMp4File),
	)
}
