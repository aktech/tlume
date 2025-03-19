package main

import (
	"encoding/json"
	"fmt"
	"github.com/schollz/progressbar/v3"
	"io"
	"log"
	"os"
	"path/filepath"
)

func main() {
	log.SetOutput(os.Stdout)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	if len(os.Args) != 2 {
		log.Fatal("Usage: tlume <machine_name>")
	}

	machineName := os.Args[1]
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

	lumeConfig := map[string]interface{}{
		"os":                osValue,
		"cpuCount":          tartConfig["cpuCount"],
		"macAddress":        tartConfig["macAddress"],
		"hardwareModel":     tartConfig["hardwareModel"],
		"memorySize":        tartConfig["memorySize"],
		"machineIdentifier": tartConfig["ecid"],
		"display":           width + "x" + height,
		"diskSize":          91268055040, // Default disk size, update if needed
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
