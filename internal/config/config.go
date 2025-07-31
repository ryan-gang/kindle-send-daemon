package config

import (
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path"
	"strconv"
	"strings"

	"github.com/ryan-gang/kindle-send-daemon/internal/util"
)

type config struct {
	Sender    string `json:"sender"`
	Receiver  string `json:"receiver"`
	StorePath string `json:"storepath"`
	Password  string `json:"password"`
	Server    string `json:"server"`
	Port      int    `json:"port"`

	BookmarkPath  string `json:"bookmark_path"`
	CheckInterval int    `json:"check_interval_minutes"`
	DaemonEnabled bool   `json:"daemon_enabled"`
	LogPath       string `json:"log_path"`
	PidFile       string `json:"pid_file"`
}

const DefaultTimeout = 120
const XdgConfigHome = "XDG_CONFIG_HOME"
const ConfigFolderName = "kindle-send"

var instance *config

func isGmail(mail string) bool {
	return strings.HasSuffix(strings.ToLower(mail), "@gmail.com")
}

func DefaultConfigPath() (string, error) {
	user, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("couldn't get current user: %w", err)
	}
	xdgConfigHome := os.Getenv(XdgConfigHome)
	var configFolder string
	if len(xdgConfigHome) == 0 {
		configFolder = path.Join(user.HomeDir, ".config")
		configFolder = path.Join(configFolder, ConfigFolderName)
	} else {
		configFolder = path.Join(xdgConfigHome, ConfigFolderName)
	}
	if err := os.MkdirAll(configFolder, os.ModePerm); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}

	return path.Join(configFolder, "KindleConfig.json"), nil
}

func SetDaemonDefaults(c *config) error {
	if c.LogPath == "" {
		configPath, err := DefaultConfigPath()
		if err != nil {
			return err
		}
		configDir := path.Dir(configPath)
		c.LogPath = path.Join(configDir, "kindle-send.log")
	}

	if c.PidFile == "" {
		configPath, err := DefaultConfigPath()
		if err != nil {
			return err
		}
		configDir := path.Dir(configPath)
		c.PidFile = path.Join(configDir, "kindle-send.pid")
	}

	return nil
}

func exists(filename string) bool {
	if _, err := os.Stat(filename); err != nil {
		util.Red.Println(err)
		return false
	}
	return true
}

func NewConfig() *config {
	config := config{}
	config.Server = "smtp.gmail.com"
	config.Port = 465

	config.CheckInterval = 15
	config.DaemonEnabled = false
	config.BookmarkPath = ""
	config.LogPath = ""
	config.PidFile = ""
	return &config
}

func CreateConfig() (*config, error) {
	util.CyanBold.Println("CONFIGURE KINDLE-SEND")

	configuration := NewConfig()
	util.Cyan.Printf("Email of your device and press enter (eg. ryan@kindle.com) : ")
	configuration.Receiver = util.ScanlineTrim()
	util.Cyan.Printf("Email that'll be used to send documents to device (eg. yourname@gmail.com) : ")
	configuration.Sender = util.ScanlineTrim()

	if !isGmail(configuration.Sender) {
		util.Cyan.Println("Sender email is different then Gmail, " +
			"can you help with SMTP server address and SMTP port for your email provider\n" +
			"Just search SMTP settings for <your email domain>.com on internet")

		util.Cyan.Printf("Enter SMTP Server Address (eg. smtp.gmail.com) : ")
		configuration.Server = util.ScanlineTrim()
		for {
			util.Cyan.Printf("Enter SMTP port (usually 587 or 465) : ")
			portStr := util.ScanlineTrim()
			portInt, err := strconv.Atoi(portStr)
			if err != nil {
				util.Red.Println("Entered port number is either invalid or not an integer, please try again")
				continue
			}
			configuration.Port = portInt
			break
		}
	}

	util.Cyan.Printf("Enter password for Sender %s (password remains encrypted in your machine) : ", configuration.Sender)
	configuration.Password = util.ScanlineTrim()

	util.Cyan.Printf("File path to store all the documents on your computer (empty is ok) :")
	configuration.StorePath = util.ScanlineTrim()

	util.CyanBold.Println("\nDAEMON CONFIGURATION")
	util.Cyan.Printf("Path to bookmark file/folder to monitor (empty to disable daemon) :")
	configuration.BookmarkPath = util.ScanlineTrim()
	if configuration.BookmarkPath != "" {
		configuration.DaemonEnabled = true
		util.Cyan.Printf("Check interval in minutes (default 15) :")
		intervalStr := util.ScanlineTrim()
		if intervalStr != "" {
			if interval, err := strconv.Atoi(intervalStr); err == nil && interval > 0 {
				configuration.CheckInterval = interval
			}
		}
	}

	encryptedPass, err := Encrypt(configuration.Sender, configuration.Password)
	if err != nil {
		return nil, fmt.Errorf("error encrypting password: %w", err)
	}
	configuration.Password = encryptedPass

	return configuration, nil
}

func handleCreation(filename string) error {
	util.Red.Println("Configuration file doesn't exist\n Answer next few questions to create config file")
	configuration, err := CreateConfig()
	if err != nil {
		return fmt.Errorf("failed to create configuration: %w", err)
	}
	err = Save(*configuration, filename)
	if err != nil {
		util.Red.Println("Error while writing config to ", filename, err)
		return err
	}
	util.Green.Printf("Config created successfully and stored at %s, you can directly edit it later on \n", filename)
	return nil
}

func LoadProvider(filename string) (ConfigProvider, error) {
	cfg, err := Load(filename)
	if err != nil {
		return nil, err
	}
	return NewConfigProvider(&cfg), nil
}

func Load(filename string) (config, error) {
	if !exists(filename) {
		err := handleCreation(filename)
		if err != nil {
			return config{}, err
		}
	}
	data, err := os.ReadFile(filename)
	if err != nil {
		util.Red.Println("Error reading config ", err)
		return config{}, err
	}
	var c config
	err = json.Unmarshal(data, &c)
	if err != nil {
		util.Red.Println("Error converting config to json ", err)
		return config{}, err
	}
	decryptedPass, err := Decrypt(c.Sender, c.Password)
	if err != nil {
		return config{}, fmt.Errorf("error decrypting password: %w", err)
	}
	c.Password = decryptedPass

	if err := SetDaemonDefaults(&c); err != nil {
		util.Red.Println("Error setting daemon defaults: ", err)
		return config{}, err
	}

	InitializeConfig(&c)
	return c, nil
}

func Save(c config, filename string) error {
	data, err := json.MarshalIndent(c, "", "	")
	if err != nil {
		util.Red.Println("Error parsing configuration for writing")
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

func InitializeConfig(c *config) {
	if instance == nil {
		instance = c
		util.Green.Println("Loaded configuration")
	}
}

func GetInstance() *config {
	return instance
}
