package config

import (
	"bytes"
	"errors"
	"io/ioutil"
	"os"
	"os/user"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/Songmu/prompter"
	"github.com/zaru/jobcan/types"
)

// Config is command parameters
type Config struct {
	Credential CredentialConfig
}

// CredentialConfig is jobcan credential
type CredentialConfig struct {
	ClientID    string
	LoginID     string
	Password    string
	AccountType string
}

func configPath() string {
	// only OSX
	usr, _ := user.Current()
	return strings.Replace("~/.jobcan", "~", usr.HomeDir, 1)
}

func Init() {
	ac := []string{types.Admin, types.Staff}

	var config Config
	var credentialConfig CredentialConfig
	credentialConfig.AccountType = prompter.Choose("Choose your account type", ac, types.Staff)
	credentialConfig.ClientID = prompter.Prompt("Enter your client ID", "")
	credentialConfig.LoginID = prompter.Prompt("Enter your login ID", "")
	credentialConfig.Password = prompter.Prompt("Enter your password", "")
	config.Credential = credentialConfig

	var buffer bytes.Buffer
	encoder := toml.NewEncoder(&buffer)
	_ = encoder.Encode(config)

	ioutil.WriteFile(configPath(), []byte(buffer.String()), os.ModePerm)
}

func Read() (Config, error) {
	var config Config
	_, err := toml.DecodeFile(configPath(), &config)
	if err != nil {
		return config, errors.New("Config file is broken ;; please try `jobcan init`.")
	}
	return config, nil
}
