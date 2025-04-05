package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"reflect"
	"strings"
)

type Config struct {
	Hostname string `json:"hostname"`
	Username string `json:"username"`
	FlakeDir string `json:"flakeDir" env:"FLAKE_DIR"`
	Profile string `json:"profile" env:"PROFILE"` 		// For later
	DesktopEnvironment string `json:"desktopEnvironment" env:"DESKTOP_ENVIRONMENT"`
	Theme string `json:"theme" env:"THEME"`
	EnabledModules []string `json:"enabledModules" env:"ENABLED_MODULES"`
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

func read() int {
	if len(os.Args) < 3 {
		log.Fatal("Please give a filename")
	}
	
	var config Config
	readJson(&config, os.Args[2])
	fmt.Println(config)
	return 0
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
	cmd := exec.Command("sudo", "nixos-rebuild", "switch", "--flake", config.FlakeDir + "#" + hostname, "--impure")
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

	if len(os.Args) > 2 || isSystemSwitch {
		if strings.ToLower(os.Args[2]) == "all" || isSystemSwitch {
			fmt.Println("Switching system configuration")
			systemSwitchConfig(config)
		}
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

func main() {
	sessionDir := setup()
	var config Config
	readJson(&config, sessionDir)
	if len(os.Args) > 1 {
		switch os.Args[1] {
			case "read":
				read()
			case "help":
				help()
			case "switch":
				switchConfig(&config)
			case "desktop":
				switchDesktop(&config)

				// Only if sucsessfull we clear the theme (set it do DE default) and write the config.
				config.Theme = ""
				writeJson(config, sessionDir)
		}
	}
}
