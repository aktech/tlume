package tlume

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

// ConvertLumeToTart converts a Lume config to a Tart config
func ConvertLumeToTart(machineName string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("unable to determine home directory: %w", err)
	}

	lumeConfigPath := filepath.Join(homeDir, ".lume", machineName, "config.json")
	tartConfigPath := filepath.Join(homeDir, ".tart", machineName, "config.json")

	if _, err := os.Stat(lumeConfigPath); os.IsNotExist(err) {
		return fmt.Errorf("Lume config not found at %s", lumeConfigPath)
	}

	data, err := ioutil.ReadFile(lumeConfigPath)
	if err != nil {
		return fmt.Errorf("error reading Lume config: %w", err)
	}

	var lumeConfig map[string]interface{}
	if err := json.Unmarshal(data, &lumeConfig); err != nil {
		return fmt.Errorf("error parsing Lume config: %w", err)
	}

	display := map[string]int{
		"width":  1024,
		"height": 768,
	}
	if val, ok := lumeConfig["display"].(string); ok {
		var width, height int
		fmt.Sscanf(val, "%dx%d", &width, &height)
		display["width"] = width
		display["height"] = height
	}

	tartConfig := map[string]interface{}{
		"os":            lumeConfig["os"],
		"arch":          lumeConfig["arch"],
		"cpuCount":      lumeConfig["cpuCount"],
		"cpuCountMin":   lumeConfig["cpuCountMin"],
		"memorySize":    lumeConfig["memorySize"],
		"memorySizeMin": lumeConfig["memorySizeMin"],
		"macAddress":    lumeConfig["macAddress"],
		"hardwareModel": lumeConfig["hardwareModel"],
		"display":       display,
		"diskSize":      lumeConfig["diskSize"],
		"ecid":          lumeConfig["machineIdentifier"],
		"version":       lumeConfig["version"],
	}

	dir := filepath.Dir(tartConfigPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("error creating Tart config directory: %w", err)
	}

	tartData, err := json.MarshalIndent(tartConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("error encoding Tart config: %w", err)
	}

	if err := ioutil.WriteFile(tartConfigPath, tartData, 0644); err != nil {
		return fmt.Errorf("error writing Tart config: %w", err)
	}

	fmt.Printf("Converted Lume config from %s to Tart config at %s\n", lumeConfigPath, tartConfigPath)
	return nil
}
