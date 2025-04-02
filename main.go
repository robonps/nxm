package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"reflect"
)

type Config struct {
	Hostname string `json:"hostname"`
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

func switchConfig(config *Config) {

	fmt.Println("Setting Env Vars...")
	setEnvVars(*config)
	fmt.Println("Done!")

	fmt.Println("Switching to new home-manager config")
	cmd := exec.Command("home-manager", "switch", "--flake", config.FlakeDir + "#robert", "--impure")

	out, err := cmd.StdoutPipe()
	cmd.Stderr = cmd.Stdout
	cmd.Start()
	if err != nil {
		// if there was any error, print it here
		log.Fatal(err)
	}
	outScanner := bufio.NewScanner(out)
	for outScanner.Scan() {
		m := outScanner.Text()
		fmt.Println(m)
	}
}

func switchDesktop(config *Config) {
	if len(os.Args) < 3 {
		log.Fatal("Please specify a desktop environment to switch to.")
	}
	config.DesktopEnvironment = os.Args[2]
	switchConfig(config)
	
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
