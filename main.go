package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
)

const ENVIRONMENTDIR = "/home/modules/environments"

type Config struct {
	Hostname string `json:"hostname"`
	Username string `json:"username"`
	FlakeDir string `json:"flakeDir" env:"FLAKE_DIR"`
	Profile string `json:"profile" env:"PROFILE"` 		// For later
	DesktopEnvironment string `json:"desktopEnvironment" env:"DESKTOP_ENVIRONMENT"`
	Theme string `json:"theme" env:"THEME"`
	EnabledModules []string `json:"enabledModules" env:"ENABLED_MODULES"`
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func setup() string {
	userDir, err := os.UserConfigDir(); 
	if err != nil {
		log.Fatal(err)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	if _, err := os.Stat(userDir + "/nxm/session.json"); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(userDir + "/nxm", os.ModePerm); err != nil {
				log.Fatal(err)
			}
			if _, err := os.Create(userDir + "/nxm/session.json"); err != nil {
				log.Fatal(err)
			}
			var config Config
			config.Username = os.Getenv("USER")
			if config.Hostname, err = os.Hostname(); err != nil {
				log.Fatal(err)
			}
			config.FlakeDir = homeDir + "/nixos-config"
			writeJson(config, userDir + "/nxm/session.json")
		} else {
			log.Fatal(err)
		}
	}
	return userDir + "/nxm/session.json"
}

func readJson(config *Config, file string) {
	data, err := ioutil.ReadFile(file)

	if err != nil {
		log.Fatal(err)
	}
	err = json.Unmarshal(data, config)

	if err != nil {
		log.Fatal(err)
	}
}

func writeJson(config Config, file string) {
	data, err := json.MarshalIndent(config, "", "    ")	

	if err != nil {
		log.Fatal(err)
	}

	err = ioutil.WriteFile(file, data, 0644)

	if err != nil {
		log.Fatal(err)
	}
}

func help() {
	fmt.Println("Help NXM:\nUsage\tnxm arg")
}


func setEnvVars(config Config) {
    v := reflect.ValueOf(config)
    t := reflect.TypeOf(config) 

    for i := 0; i < v.NumField(); i++ {   
        field := v.Field(i)                 
        tag := t.Field(i).Tag.Get("env") 

        if tag != "" && field.CanInterface() {
            os.Setenv(tag, fmt.Sprintf("%v", field.Interface()))
        }
    }
}

func runCmd(cmd *exec.Cmd) {
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	err := cmd.Run()
	
	if err != nil {
		log.Fatal(err)
	}
}

func userSwitchConfig(config *Config) {

	cmd := exec.Command("home-manager", "switch", "--flake", config.FlakeDir + "#" + config.Username, "--impure")
	runCmd(cmd)
}

func systemSwitchConfig(config *Config) {
	hostname := strings.ToLower(config.Hostname)
	cmd := exec.Command("nixos-rebuild", "switch", "--flake", config.FlakeDir + "#" + hostname, "--impure", "--use-remote-sudo")
	runCmd(cmd)
}

func switchConfig(config *Config, systemSwitch ...bool) {

	isSystemSwitch := false
	if len(systemSwitch) > 0 {
		fmt.Println(systemSwitch[0])
		isSystemSwitch = systemSwitch[0]
	}

	fmt.Println("Setting Env Vars...")
	setEnvVars(*config)
	fmt.Println("Done!")

	if len(os.Args) > 2 {
		if strings.ToLower(os.Args[2]) == "all" {
			fmt.Println("Switching System Configuration")
			systemSwitchConfig(config)
		}
	}

	if isSystemSwitch {
		fmt.Println("Switching System Config")
		systemSwitchConfig(config)
	}

	fmt.Println("Switching to new home-manager config")
	userSwitchConfig(config)
}

func switchDesktop(config *Config) {
	if len(os.Args) < 3 {
		log.Fatal("Please specify a desktop environment to switch to.")
	}
	config.DesktopEnvironment = os.Args[2]
	switchConfig(config, true)
	
}

func updateFlake(config *Config) {
	fmt.Println("Updating Flake")
	cmd := exec.Command("nix", "flake", "update", "--flake", config.FlakeDir)
	runCmd(cmd)
	fmt.Println("Done!")
	switchConfig(config, true)
}

func getModules(config *Config) []string {
	files, err := filepath.Glob(config.FlakeDir + ENVIRONMENTDIR + "/*.nix")
	if err != nil {
		log.Fatal(err)
	}
	return files
}
func modulesList(config *Config) {

	files := getModules(config)

	moduleList := "================\n  Modules List\n================\n\nModule\t\tStatus\n------------------------"

	FileLoop:
	for _, file := range files {
		
		if filepath.Base(file) == "default.nix" { 			// Man, I really need to fix this.
			continue FileLoop
		}

		for _, module := range config.EnabledModules {
			if filepath.Base(file) == module {
				fileModule := strings.TrimSuffix(filepath.Base(file), ".nix")
				moduleList += "\n" + strings.ToUpper(fileModule[:1]) + fileModule[1:] + "\t\tEnabled"
				continue FileLoop
			}
		}
		fileModule := strings.TrimSuffix(filepath.Base(file), ".nix")
		moduleList += "\n" + strings.ToUpper(fileModule[:1]) + fileModule[1:] + "\t\tDisabled"
	}

	fmt.Println(moduleList)
}

func checkModules(files []string) []string {
	if len(os.Args) < 4 {
		fmt.Println("Please specify one or more environment modules to enable. See:\nnxm module list\nFor a list of modules.")
		os.Exit(1)
	}
	var modulesChecked []string

	ArgumentLoop:
	for _, argumentModule := range os.Args[3:] {
		for _, module := range files {
			if strings.ToLower(argumentModule) + ".nix" == filepath.Base(module) {
				modulesChecked = append(modulesChecked, filepath.Base(module))
				continue ArgumentLoop
			}
		}
		log.Fatal("\"" + argumentModule + "\" Does not exist")
	}

	if len(os.Args) - 3 != len(modulesChecked) {
		log.Fatal("One or more of the module names are invalid")
	}

	return modulesChecked
}

// TODO: Add enable all command.
func enableModules(config *Config, sessionDir string) {
	files := getModules(config)
	modulesEnable := checkModules(files)

	fmt.Println("Enabling the following modules:")
	fmt.Println(modulesEnable)

	// Check to make sure they aren't already enabled.
	for _, module := range modulesEnable {
		if !contains(config.EnabledModules, module) {
			config.EnabledModules = append(config.EnabledModules, module)
		} else {
			fmt.Println(module + " already enabled, skipping...")
		}
	}

	writeJson(*config, sessionDir)
	fmt.Println("Setting Env Vars...")
	setEnvVars(*config)
	fmt.Println("\n\nReloading home-manager...")
	userSwitchConfig(config)
}


// TODO: Add disable all command.
func disableModule(config *Config, sessionDir string){
	files := getModules(config)
	modulesDisable := checkModules(files)

	fmt.Println("Disabling the following modules:")
	fmt.Println(modulesDisable)

	// Check to make sure they aren't already disabled.
	ModuleLoop:
	for _, module := range modulesDisable {
		for i, enabledModule := range config.EnabledModules {
			if enabledModule == module {
				config.EnabledModules = append(config.EnabledModules[:i], config.EnabledModules[i+1:]...)
				continue ModuleLoop
			} else {
				fmt.Println(module + " already disabled, skipping...")
			}
		}
	}

	writeJson(*config, sessionDir)
	fmt.Println("Setting Env Vars...")
	setEnvVars(*config)
	fmt.Println("\n\nReloading home-manager...")
	userSwitchConfig(config)
}

func modules(config *Config, sessionDir string) {
	if len(os.Args) < 3 {
		// TODO: Show info about commands with module prefix
	}

	switch os.Args[2] {
		case "list":
			modulesList(config)
		case "enable":
			enableModules(config, sessionDir)
		case "disable":
			// TODO: Disable modules
			disableModule(config, sessionDir)
	}
}

func main() {
	sessionDir := setup()
	var config Config
	readJson(&config, sessionDir)
	if len(os.Args) > 1 {
		switch os.Args[1] {
			case "help":
				help()
			case "switch":
				switchConfig(&config)
			case "update":
				updateFlake(&config)
			case "module":
				modules(&config, sessionDir)
			case "desktop":
				switchDesktop(&config)

				// Only if sucsessfull we clear the theme (set it do DE default) and write the config.
				config.Theme = ""
				writeJson(config, sessionDir)
		}
	}
}
