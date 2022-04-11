package config

import (
	"encoding/json"
	"fmt"
	"log"
	"io/ioutil"
)

var (
	Webhook_url string
	Pguser      string
	Pgpass      string
	Pgdb        string
	config		*config_struct
)

type config_struct struct {
	Webhook_url string	`json: "webhook_url"`
	Pguser      string	`json: "pguser"`
	Pgpass      string	`json: "pgpass"`
	Pgdb        string	`json: "pgdb"`
}

func Read_config() error {
	fmt.Println("Reading config file...")
	file, err := ioutil.ReadFile("./config.json")
	ce(err)
	
	err = json.Unmarshal(file, &config)
	ce(err)

	// public variables passed on
	Webhook_url = config.Webhook_url
	Pguser = config.Pguser
	Pgpass = config.Pgpass
	Pgdb = config.Pgdb

	return nil
}

// error wrapper function
func ce(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
