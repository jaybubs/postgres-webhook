package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"
	"net/http"
	"crypto/tls"
	"strings"
	"text/template"
	"bytes"

	"pgwebhook/config"

	"github.com/lib/pq"
)

/*
	The listener consists of two cases: a notification is sent, or time has elapsed
	this will ensure that the connection is being kept alive. If desired "Pl(pinging)" and "load" can be uncommented and returned in the second case for debugging purposes
*/
func listen(l *pq.Listener,ch chan<- string) {
	for {
		select {
		// listener comes with a channel
		case n := <-l.Notify:
			fmt.Println("Received data from channel [", n.Channel, "] :")
			load := n.Extra
			ch <-load

		case <-time.After(90*time.Second):
			// fmt.Println("Got no notifications, pinging!")
			// run concurrently
			go func() {
				l.Ping()
			}()
			// load := "pong!"
			ch <-""
		}
	}
}

/*
	Here we push the channel contents to the webhook everytime new content is received, set up as to run forever in a similar manner to the listener, but independent of it (rather than embedding the webhook directly inside the listener)
*/
func push(ch <-chan string) {
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

// in our case a webhook is nothing but a simple post with json
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

type Template_vars struct {
	Pgchannel string
	Table_name string
	Json_column string
}

func main() {

	// make config struct accessible
	errr := config.Read_config()
	ce(errr)

	// connection string is created based off of the config.json, the password should normally not be stored in plaintext, but the security implementation is out of scope of this work
	connection := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable", config.Pguser, config.Pgpass, config.Pgdb)
	db, err := sql.Open("postgres", connection)
	ce(err)

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

	// set up function
	tmpl, err := template.ParseFiles("./create_function.sql.tmpl")
	// have to send string to bytes buffer because of io.reader method, only need to declare once
	var buf bytes.Buffer
	tmpl.Execute(&buf, login_event)
	// and back to string for db.exec purpose
	create_function := buf.String()
	_, err = db.Exec(create_function)
	ce(err)

	// set up trigger
	tmpl, err = template.ParseFiles("./create_trigger.sql.tmpl")
	tmpl.Execute(&buf, login_event)
	create_trigger := buf.String()
	_, err = db.Exec(create_trigger)
	ce(err)

	// set up function
	tmpl, err = template.ParseFiles("./create_function.sql.tmpl")
	// have to send string to bytes buffer because of io.reader method
	tmpl.Execute(&buf, admin_event)
	// and back to string for db.exec purpose
	create_function = buf.String()
	_, err = db.Exec(create_function)
	ce(err)

	// set up trigger
	tmpl, err = template.ParseFiles("./create_trigger.sql.tmpl")
	tmpl.Execute(&buf, admin_event)
	create_trigger = buf.String()
	_, err = db.Exec(create_trigger)
	ce(err)

	// callback fx for fatal error
	problem := func(ev pq.ListenerEventType, err error) {
		ce(err)
	}

	event_listener := pq.NewListener(connection, 1*time.Second, 60*time.Second, problem)
	err = event_listener.Listen(login_event.Pgchannel)
	admin_event_listener := pq.NewListener(connection, 1*time.Second, 60*time.Second, problem)
	err = admin_event_listener.Listen(admin_event.Pgchannel)
	ce(err)
	fmt.Sprintf("Listening on: %s",config.Webhook_url)
	
	event_channel := make(chan string)
	admin_channel := make(chan string)

	//create event and admin event listeners asynchronously
	go listen(event_listener, event_channel)
	go push(event_channel)

	go listen(admin_event_listener, admin_channel)
	go push(admin_channel)

	// for loop to keep the program running
	for {
		time.Sleep(time.Second * 10)
	}


}

// error wrapper function
func ce(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

