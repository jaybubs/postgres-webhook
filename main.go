package main

import (
	"database/sql"
	"fmt"
	"time"

	"pgwebhook/config"
	"pgwebhook/dbfx"
	"pgwebhook/webhook"
	. "pgwebhook/utilities"
)

func main() {

	// make config struct accessible
	errr := config.Read_config()
	CE(errr)

	// connection string is created based off of the config.json, the password should normally not be stored in plaintext, but the security implementation is out of scope of this work
	connection := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable", config.Pguser, config.Pgpass, config.Pgdb)
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

	err = dbfx.Parse_and_exec("./create_function.sql.tmpl", &login_event, db)
	CE(err)
	err = dbfx.Parse_and_exec("./create_trigger.sql.tmpl", &login_event, db)
	CE(err)
	err = dbfx.Parse_and_exec("./create_function.sql.tmpl", &admin_event, db)
	CE(err)
	err = dbfx.Parse_and_exec("./create_trigger.sql.tmpl", &admin_event, db)
	CE(err)

	event_listener, err := dbfx.Create_Listener(connection, &login_event)
	CE(err)
	admin_event_listener, err := dbfx.Create_Listener(connection, &admin_event)
	CE(err)

	event_channel := make(chan string)
	admin_channel := make(chan string)

	//create event and admin event listeners asynchronously
	go dbfx.Listen(event_listener, event_channel)
	go webhook.Push(event_channel, config.Webhook_url)

	go dbfx.Listen(admin_event_listener, admin_channel)
	go webhook.Push(admin_channel, config.Webhook_url)

	// for loop to keep the program running
	for {
		time.Sleep(time.Second * 10)
	}


}

