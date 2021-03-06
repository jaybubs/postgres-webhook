package config

// simply loading a config.json located in the root that should contain connection info

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	. "pgwebhook/utilities"
)

var (
	Webhook_url string
	Pguser      string
	Pgpass      string
	Pgdb        string
	config		*config_struct
)

type config_struct struct {
	Webhook_url string	`json:"webhook_url"`
	Pguser      string	`json:"pguser"`
	Pgpass      string	`json:"pgpass"`
	Pgdb        string	`json:"pgdb"`
}

func Read_config() error {
	fmt.Println("Reading config file...")
	file, err := ioutil.ReadFile("./config.json")
	CE(err)
	
	err = json.Unmarshal(file, &config)
	CE(err)

	Webhook_url = config.Webhook_url
	Pguser = config.Pguser
	Pgpass = config.Pgpass
	Pgdb = config.Pgdb

	return nil
}
