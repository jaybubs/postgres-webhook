package main

import (
	"database/sql"
	"fmt"
	"time"
	"net/http"
	"crypto/tls"
	"strings"
	"text/template"
	"bytes"

	"pgwebhook/config"
	"pgwebhook/dbfx"
	. "pgwebhook/utilities"
)



/*
	Here we push the channel contents to the webhook everytime new content is received, set up as to run forever in a similar manner to the listener, but independent of it (rather than embedding the webhook directly inside the listener)
*/
func Push(ch <-chan string) {
	for {
		// let's not spam the webhook endpoint and only post when there's actual content
		select {
		case payload := <-ch:
			if len(payload) > 0 {
				webhook(payload, config.Webhook_url)
			}
		}
	}
}

// in our case a webhook is nothing but a simple post request with json
func webhook(load string, webhook_url string) {
	payload := strings.NewReader(load)
	fmt.Println(payload)
	
	// guess what, self signed and corporation certs are a pita, so we skip the check, _be careful you trust the dest_ (in our case it's all in local cluster, we good)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	client.Post(webhook_url, "application/json", payload)
	
}


func Parse_and_exec(filename string, tvars *Template_vars, db *sql.DB) error {

	tmpl, err := template.ParseFiles(filename)
	CE(err)
	// save to bytes
	var buffalo bytes.Buffer
	tmpl.Execute(&buffalo, tvars)
	// and back to string for db.exec
	_, err = db.Exec(buffalo.String())
	CE(err)

	return nil

}

func main() {

	// make config struct accessible
	errr := config.Read_config()
	CE(errr)

	// connection string is created based off of the config.json, the password should normally not be stored in plaintext, but the security implementation is out of scope of this work
	connection := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable", config.Pguser, config.Pgpass, config.Pgdb)
	// db is a pointer already!
	db, err := sql.Open("postgres", connection)
	CE(err)

	// one struct for keycloak login events and one for admin events

	login_event := Template_vars{
		Pgchannel: "logins_logger",
		Table_name: "event_entity",
		Json_column: "details_json",
	}
	admin_event := Template_vars{
		Pgchannel: "admin_logger",
		Table_name: "admin_event_entity",
		Json_column: "representation",
	}

	err = Parse_and_exec("./create_function.sql.tmpl", &login_event, db)
	CE(err)
	err = Parse_and_exec("./create_trigger.sql.tmpl", &login_event, db)
	CE(err)
	err = Parse_and_exec("./create_function.sql.tmpl", &admin_event, db)
	CE(err)
	err = Parse_and_exec("./create_trigger.sql.tmpl", &admin_event, db)
	CE(err)

	event_listener, err := dbfx.Create_Listener(connection, &login_event)
	CE(err)
	admin_event_listener, err := dbfx.Create_Listener(connection, &admin_event)
	CE(err)

	event_channel := make(chan string)
	admin_channel := make(chan string)

	//create event and admin event listeners asynchronously
	go dbfx.Listen(event_listener, event_channel)
	go Push(event_channel)

	go dbfx.Listen(admin_event_listener, admin_channel)
	go Push(admin_channel)

	// for loop to keep the program running
	for {
		time.Sleep(time.Second * 10)
	}


}

