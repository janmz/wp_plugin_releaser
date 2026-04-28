package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

// Cache for executable paths
var (
	exePathCache     = make(map[string]*string)
	exePathCacheLock sync.RWMutex
)

func findSVGFiles(updatesDir string) ([]string, error) {
	files, err := os.ReadDir(updatesDir)
	if err != nil {
		return nil, err
	}

	var svgFiles []string
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(strings.ToLower(file.Name()), ".svg") {
			svgFiles = append(svgFiles, file.Name())
		}
	}
	return svgFiles, nil
}

func checkSVGFilesChanged(workDir string) ([]string, error) {
	updatesDir := filepath.Join(workDir, "Updates")

	if _, err := os.Stat(filepath.Join(workDir, ".git")); err == nil {
		changedFiles, err := getChangedFiles(workDir)
		if err == nil {
			var changedSVGFiles []string
			for _, file := range changedFiles {
				if strings.HasSuffix(strings.ToLower(file), ".svg") {
					filename := filepath.Base(file)
					changedSVGFiles = append(changedSVGFiles, filename)
				}
			}
			return changedSVGFiles, nil
		}
	}

	svgFiles, err := findSVGFiles(updatesDir)
	if err != nil {
		return nil, err
	}
	return svgFiles, nil
}

func lookUpExe(name string, windowsRelativePath string) *string {
	cacheKey := name
	if runtime.GOOS == "windows" && windowsRelativePath != "" {
		cacheKey = name + ":" + windowsRelativePath
	}

	exePathCacheLock.RLock()
	if cached, ok := exePathCache[cacheKey]; ok {
		exePathCacheLock.RUnlock()
		return cached
	}
	exePathCacheLock.RUnlock()

	exeName := name
	if runtime.GOOS == "windows" {
		exeName = name + ".exe"
	}

	path, err := exec.LookPath(exeName)
	if err == nil {
		exePathCacheLock.Lock()
		exePathCache[cacheKey] = &path
		exePathCacheLock.Unlock()
		return &path
	}

	if runtime.GOOS == "windows" && windowsRelativePath != "" {
		programFiles := os.Getenv("ProgramFiles")
		if programFiles == "" {
			programFiles = "C:\\Program Files"
		}

		if strings.Contains(windowsRelativePath, "*") {
			globPattern := filepath.Join(programFiles, windowsRelativePath, exeName)
			matches, err := filepath.Glob(globPattern)
			if err == nil && len(matches) > 0 {
				fullPath := matches[0]
				exePathCacheLock.Lock()
				exePathCache[cacheKey] = &fullPath
				exePathCacheLock.Unlock()
				return &fullPath
			}
		} else {
			fullPath := filepath.Join(programFiles, windowsRelativePath, exeName)
			if _, err := os.Stat(fullPath); err == nil {
				exePathCacheLock.Lock()
				exePathCache[cacheKey] = &fullPath
				exePathCacheLock.Unlock()
				return &fullPath
			}
		}
	}

	exePathCacheLock.Lock()
	exePathCache[cacheKey] = nil
	exePathCacheLock.Unlock()
	return nil
}

func convertSVGToPNG(updatesDir string, svgFiles []string) error {
	execPathInkscape := lookUpExe("inkscape", "inkscape\\bin")
	execPathImageMagick := lookUpExe("magick", "ImageMagick-*")

	if execPathInkscape == nil && execPathImageMagick == nil {
		logAndPrint(t("error.svg_converter_missing"))
		logAndPrint("Skipping SVG to PNG conversion. Please install ImageMagick (magick) or Inkscape.")
		return nil
	}

	var converter func(string, string, []int, []int) error
	if execPathInkscape != nil {
		converter = convertSingleSVGWithInkscape
	} else {
		converter = convertSingleSVGWithImageMagick
	}

	for _, svgFile := range svgFiles {
		svgPath := filepath.Join(updatesDir, svgFile)

		squareSizes := []int{128, 256}
		wideSizes := [][]int{{772, 250}, {1544, 500}}

		filename := strings.ToLower(filepath.Base(svgFile))
		isLikelyLogo := strings.Contains(filename, "logo") || strings.Contains(filename, "icon")
		isLikelyBanner := strings.Contains(filename, "banner")

		if isLikelyLogo {
			for _, size := range squareSizes {
				if err := converter(svgPath, updatesDir, []int{size, size}, nil); err != nil {
					return err
				}
			}
		} else if isLikelyBanner {
			for _, dims := range wideSizes {
				if err := converter(svgPath, updatesDir, dims, nil); err != nil {
					return err
				}
			}
		} else {
			// For files that are neither banner nor logo/icon, generate exactly one
			// PNG with height 1024px and auto width (preserve aspect ratio).
			if execPathInkscape != nil {
				if err := convertSingleSVGHeight1024WithInkscape(svgPath, updatesDir); err != nil {
					return err
				}
			} else {
				if err := convertSingleSVGHeight1024WithImageMagick(svgPath, updatesDir); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func convertSingleSVGHeight1024WithImageMagick(svgPath string, outputDir string) error {
	baseName := strings.TrimSuffix(filepath.Base(svgPath), ".svg")
	baseName = strings.TrimSuffix(baseName, ".SVG")

	outputPath := filepath.Join(outputDir, fmt.Sprintf("%s-h1024.png", baseName))

	execPath := lookUpExe("magick", "ImageMagick-*")
	if execPath == nil {
		return fmt.Errorf("magick executable not found")
	}

	// Height-only resize: keep aspect ratio, auto width.
	cmd := exec.Command(*execPath, "-background", "transparent", "-resize", "x1024", svgPath, outputPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to convert %s: %v", svgPath, err)
	}

	logAndPrint(fmt.Sprintf("Converted: %s -> %s", filepath.Base(svgPath), filepath.Base(outputPath)))
	return nil
}

func convertSingleSVGHeight1024WithInkscape(svgPath string, outputDir string) error {
	baseName := strings.TrimSuffix(filepath.Base(svgPath), ".svg")
	baseName = strings.TrimSuffix(baseName, ".SVG")

	outputPath := filepath.Join(outputDir, fmt.Sprintf("%s-h1024.png", baseName))

	execPath := lookUpExe("inkscape", "inkscape\\bin")
	if execPath == nil {
		return fmt.Errorf("inkscape executable not found")
	}

	// Specify only height to preserve aspect ratio.
	cmd := exec.Command(*execPath, "--export-filename", outputPath, "--export-height", "1024", svgPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to convert %s: %v", svgPath, err)
	}

	logAndPrint(fmt.Sprintf("Converted: %s -> %s", filepath.Base(svgPath), filepath.Base(outputPath)))
	return nil
}

func convertSingleSVGWithImageMagick(svgPath string, outputDir string, squareSize []int, wideSize []int) error {
	baseName := strings.TrimSuffix(filepath.Base(svgPath), ".svg")
	baseName = strings.TrimSuffix(baseName, ".SVG")

	var outputPath string
	var resizeArg string

	if len(squareSize) == 2 {
		outputPath = filepath.Join(outputDir, fmt.Sprintf("%s-%dx%d.png", baseName, squareSize[0], squareSize[1]))
		resizeArg = fmt.Sprintf("%dx%d", squareSize[0], squareSize[1])
	} else if len(wideSize) == 2 {
		outputPath = filepath.Join(outputDir, fmt.Sprintf("%s-%dx%d.png", baseName, wideSize[0], wideSize[1]))
		resizeArg = fmt.Sprintf("%dx%d", wideSize[0], wideSize[1])
	} else {
		return fmt.Errorf("invalid size parameters")
	}

	execPath := lookUpExe("magick", "ImageMagick-*")
	if execPath == nil {
		return fmt.Errorf("magick executable not found")
	}

	cmd := exec.Command(*execPath, "-background", "transparent", "-resize", resizeArg, svgPath, outputPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to convert %s: %v", svgPath, err)
	}

	logAndPrint(fmt.Sprintf("Converted: %s -> %s", filepath.Base(svgPath), filepath.Base(outputPath)))
	return nil
}

func convertSingleSVGWithInkscape(svgPath string, outputDir string, squareSize []int, wideSize []int) error {
	baseName := strings.TrimSuffix(filepath.Base(svgPath), ".svg")
	baseName = strings.TrimSuffix(baseName, ".SVG")

	var outputPath string
	var width, height string

	if len(squareSize) == 2 {
		outputPath = filepath.Join(outputDir, fmt.Sprintf("%s-%dx%d.png", baseName, squareSize[0], squareSize[1]))
		width = strconv.Itoa(squareSize[0])
		height = strconv.Itoa(squareSize[1])
	} else if len(wideSize) == 2 {
		outputPath = filepath.Join(outputDir, fmt.Sprintf("%s-%dx%d.png", baseName, wideSize[0], wideSize[1]))
		width = strconv.Itoa(wideSize[0])
		height = strconv.Itoa(wideSize[1])
	} else {
		return fmt.Errorf("invalid size parameters")
	}

	execPath := lookUpExe("inkscape", "inkscape\\bin")
	if execPath == nil {
		return fmt.Errorf("inkscape executable not found")
	}

	cmd := exec.Command(*execPath, "--export-filename", outputPath, "--export-width", width, "--export-height", height, svgPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to convert %s: %v", svgPath, err)
	}

	logAndPrint(fmt.Sprintf("Converted: %s -> %s", filepath.Base(svgPath), filepath.Base(outputPath)))
	return nil
}

func uniqueStrings(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	var out []string
	for _, s := range items {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

func fileExists(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}

func expectedPNGPathsForSVG(updatesDir string, svgFilename string) []string {
	filename := strings.ToLower(filepath.Base(svgFilename))
	baseName := strings.TrimSuffix(filepath.Base(svgFilename), ".svg")
	baseName = strings.TrimSuffix(baseName, ".SVG")

	squareSizes := []int{128, 256}
	wideSizes := [][]int{{772, 250}, {1544, 500}}

	isLikelyLogo := strings.Contains(filename, "logo") || strings.Contains(filename, "icon")
	isLikelyBanner := strings.Contains(filename, "banner")

	var out []string
	if isLikelyLogo {
		for _, size := range squareSizes {
			out = append(out, filepath.Join(updatesDir, fmt.Sprintf("%s-%dx%d.png", baseName, size, size)))
		}
		return out
	}
	if isLikelyBanner {
		for _, dims := range wideSizes {
			out = append(out, filepath.Join(updatesDir, fmt.Sprintf("%s-%dx%d.png", baseName, dims[0], dims[1])))
		}
		return out
	}

	// Unknown type: only one height-based export.
	return []string{filepath.Join(updatesDir, fmt.Sprintf("%s-h1024.png", baseName))}
}

func findSVGsWithMissingOrStalePNGs(updatesDir string, svgFiles []string) []string {
	var out []string
	for _, svgFile := range svgFiles {
		svgPath := filepath.Join(updatesDir, filepath.Base(svgFile))
		svgInfo, err := os.Stat(svgPath)
		if err != nil {
			continue
		}
		paths := expectedPNGPathsForSVG(updatesDir, svgFile)
		if len(paths) == 0 {
			continue
		}
		for _, p := range paths {
			pngInfo, err := os.Stat(p)
			if err != nil {
				out = append(out, svgFile)
				break
			}
			if pngInfo.ModTime().Before(svgInfo.ModTime()) {
				out = append(out, svgFile)
				break
			}
		}
	}
	return uniqueStrings(out)
}

func filterExistingSVGsInUpdates(updatesDir string, svgFiles []string) []string {
	var out []string
	for _, svgFile := range svgFiles {
		if fileExists(filepath.Join(updatesDir, filepath.Base(svgFile))) {
			out = append(out, filepath.Base(svgFile))
		}
	}
	return uniqueStrings(out)
}

func processSVGFiles(workDir string, updateInfo *UpdateInfo) error {
	updatesDir := filepath.Join(workDir, "Updates")

	if _, err := os.Stat(updatesDir); os.IsNotExist(err) {
		return nil
	}

	changedSVGFiles, err := checkSVGFilesChanged(workDir)
	if err != nil {
		return err
	}
	changedSVGFiles = filterExistingSVGsInUpdates(updatesDir, changedSVGFiles)

	svgFilesToConvert := changedSVGFiles
	if len(svgFilesToConvert) == 0 {
		allSVGFiles, err := findSVGFiles(updatesDir)
		if err != nil {
			return err
		}
		allSVGFiles = filterExistingSVGsInUpdates(updatesDir, allSVGFiles)
		svgFilesToConvert = findSVGsWithMissingOrStalePNGs(updatesDir, allSVGFiles)
		if len(svgFilesToConvert) > 0 {
			logAndPrint("SVGs changed: none detected; forcing conversion because PNGs are missing or stale")
		}
	}

	if len(svgFilesToConvert) == 0 {
		return nil
	}

	logAndPrint(t("log.svg_converting"))
	logAndPrint(fmt.Sprintf("Found %d SVG file(s) to convert", len(svgFilesToConvert)))

	for _, svgFile := range svgFilesToConvert {
		logAndPrint(fmt.Sprintf("SVG convert candidate: %s", svgFile))
		svgPath := filepath.Join(updatesDir, filepath.Base(svgFile))
		svgInfo, svgErr := os.Stat(svgPath)
		for _, p := range expectedPNGPathsForSVG(updatesDir, svgFile) {
			pngInfo, pngErr := os.Stat(p)
			if pngErr == nil {
				if svgErr == nil && pngInfo.ModTime().Before(svgInfo.ModTime()) {
					logAndPrint(fmt.Sprintf("Stale PNG target: %s", filepath.Base(p)))
				}
				continue
			}
			logAndPrint(fmt.Sprintf("Missing PNG target: %s", filepath.Base(p)))
		}
	}

	if err := convertSVGToPNG(updatesDir, svgFilesToConvert); err != nil {
		return err
	}

	logAndPrint(t("log.svg_converted"))
	return nil
}

