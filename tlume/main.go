package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/schollz/progressbar/v3"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
)

func main() {
	log.SetOutput(os.Stdout)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	// Define command line flags
	var diskSizeStr string
	flag.StringVar(&diskSizeStr, "disk-size", "85GB", "Disk size with unit suffix: e.g., 100GB, 2TB (default: 85GB)")

	// Parse flags but keep the positional arguments
	flag.Parse()
	args := flag.Args()

	if len(args) != 1 {
		log.Fatal("Usage: tlume [options] <machine_name>\n\nOptions:\n  -disk-size STRING   Disk size with unit (e.g., 100GB, 2TB) (default: 85GB)")
	}

	// Parse disk size string to get the value in bytes
	diskSizeBytes, err := parseDiskSize(diskSizeStr)
	if err != nil {
		log.Fatalf("Error parsing disk size: %v", err)
	}

	machineName := args[0]
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal("Error: Unable to determine home directory", err)
	}

	lumeConfigPath := filepath.Join(homeDir, ".lume", machineName, "config.json")
	tartConfigPath := filepath.Join(homeDir, ".tart", "vms", machineName, "config.json")

	lumeMachinePath := filepath.Join(homeDir, ".lume", machineName)
	tartMachinePath := filepath.Join(homeDir, ".tart", "vms", machineName)

	log.Println("Copying machine from Tart to Lume...")
	if err := copyDirWithProgress(tartMachinePath, lumeMachinePath); err != nil {
		log.Fatal("Error copying machine from Tart to Lume:", err)
	}

	log.Println("Reading Tart config...")
	tartData, err := os.ReadFile(tartConfigPath)
	if err != nil {
		log.Fatal("Error reading Tart config:", err)
	}

	var tartConfig map[string]interface{}
	if err := json.Unmarshal(tartData, &tartConfig); err != nil {
		log.Fatal("Error parsing Tart config:", err)
	}

	// Safely extract display width and height as numeric values and convert to string
	display := tartConfig["display"].(map[string]interface{})
	width := fmt.Sprintf("%.0f", display["width"].(float64))
	height := fmt.Sprintf("%.0f", display["height"].(float64))

	// Convert Tart config format to Lume config format (Strict Schema)
	// Convert "darwin" to "macOS" for the os field
	osValue := tartConfig["os"]
	if osValue == "darwin" {
		osValue = "macOS"
	}

	// Log the disk size being used
	sizeInGB := float64(diskSizeBytes) / (1024 * 1024 * 1024)
	log.Printf("Setting disk size to %.2f GB (%d bytes) - required for Lume config as Tart doesn't specify disk size",
		sizeInGB, diskSizeBytes)

	lumeConfig := map[string]interface{}{
		"os":                osValue,
		"cpuCount":          tartConfig["cpuCount"],
		"macAddress":        tartConfig["macAddress"],
		"hardwareModel":     tartConfig["hardwareModel"],
		"memorySize":        tartConfig["memorySize"],
		"machineIdentifier": tartConfig["ecid"],
		"display":           width + "x" + height,
		"diskSize":          diskSizeBytes, // Using the converted bytes value
	}

	updatedLumeData, err := json.MarshalIndent(lumeConfig, "", "  ")
	if err != nil {
		log.Fatal("Error encoding updated Lume config:", err)
	}

	if err := os.WriteFile(lumeConfigPath, updatedLumeData, 0644); err != nil {
		log.Fatal("Error writing updated Lume config:", err)
	}

	log.Println("Lume config updated successfully!")
	log.Printf("Successfully copied machine from %s to %s and updated config\n", tartMachinePath, lumeMachinePath)
}

// parseDiskSize parses a disk size string like "100GB" or "2TB" into bytes
func parseDiskSize(sizeStr string) (int64, error) {
	// Regular expression to match number followed by optional unit (B, KB, MB, GB, TB)
	re := regexp.MustCompile(`^(\d+(?:\.\d+)?)([KMGT]?B)?$`)
	matches := re.FindStringSubmatch(sizeStr)

	if matches == nil {
		return 0, fmt.Errorf("invalid disk size format: %s (examples of valid formats: 100GB, 2TB, 500)", sizeStr)
	}

	// Parse the numeric part
	value, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0, err
	}

	// Convert to bytes based on the unit
	var multiplier float64 = 1
	if len(matches) > 2 && matches[2] != "" {
		switch matches[2] {
		case "B":
			multiplier = 1
		case "KB":
			multiplier = 1024
		case "MB":
			multiplier = 1024 * 1024
		case "GB":
			multiplier = 1024 * 1024 * 1024
		case "TB":
			multiplier = 1024 * 1024 * 1024 * 1024
		default:
			// If no unit specified, assume GB
			multiplier = 1024 * 1024 * 1024
		}
	} else {
		// If no unit specified, assume GB
		multiplier = 1024 * 1024 * 1024
	}

	return int64(value * multiplier), nil
}

// copyDirWithProgress copies the contents of one directory to another with a real-time progress bar
func copyDirWithProgress(src string, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	dirEntries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	var totalSize int64
	for _, entry := range dirEntries {
		info, err := os.Stat(filepath.Join(src, entry.Name()))
		if err == nil && !entry.IsDir() {
			totalSize += info.Size()
		}
	}

	bar := progressbar.DefaultBytes(totalSize, "Copying...")

	for _, entry := range dirEntries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDirWithProgress(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFileWithProgress(srcPath, dstPath, bar); err != nil {
				return err
			}
		}
	}

	return nil
}

// copyFileWithProgress copies a file from src to dst and updates progress bar in real-time
func copyFileWithProgress(src string, dst string, bar *progressbar.ProgressBar) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()
	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	buf := make([]byte, 1024*1024) // 1MB buffer for large file copying
	for {
		n, err := srcFile.Read(buf)
		if n > 0 {
			_, writeErr := dstFile.Write(buf[:n])
			if writeErr != nil {
				return writeErr
			}
			bar.Add(n)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}

	return nil
}
