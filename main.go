package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"
	"net/http"
	"crypto/tls"
	"strings"

	"pgwebhook/config"

	"github.com/lib/pq"
)

/*
	The listener consists of two cases: a notification is sent, or time has elapsed
	this will ensure that the connection is being kept alive. If desired "Pl(pinging)" and "load" can be uncommented and returned in the second case for debugging purposes
*/
func listen(l *pq.Listener) (load string) {
	for {
		select {
		// listener comes with a channel
		case n := <-l.Notify:
			fmt.Println("Received data from channel [", n.Channel, "] :")
			load := n.Extra
			return load

		case <-time.After(90*time.Second):
			// fmt.Println("Got no notifications, pinging!")
			// run concurrently
			go func() {
				l.Ping()
			}()
			// load := "pong!"
			return ""
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

func main() {

	// make config struct accessible
	errr := config.Read_config()
	ce(errr)

	// connection string is created based off of the config.json, the password should normally not be stored in plaintext, but the security implementation is out of scope of this work
	connection := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable", config.Pguser, config.Pgpass, config.Pgdb)
	db, err := sql.Open("postgres", connection)
	ce(err)

	// set up trigger function
	_, err = db.Exec(create_function)
	ce(err)
	// set up trigger
	_, err = db.Exec(create_trigger)
	ce(err)
	

	// callback fx for fatal error
	problem := func(ev pq.ListenerEventType, err error) {
		ce(err)
	}

	event_listener := pq.NewListener(connection, 1*time.Second, 60*time.Second, problem)
	err = event_listener.Listen("logins_logger")
	ce(err)
	fmt.Sprintf("Listening on: %s",config.Webhook_url)
	
	for {
		conts := listen(event_listener)
		
		// let's not spam the webhook endpoint and only post when there's actual content
		if len(conts) > 0 {
			webhook(conts, config.Webhook_url)
		}
	}

}

// error wrapper function
func ce(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

/*
	We'll create the two sql scripts in here for now and inject them directly.
	In the future this will be generalised so it's more scalable.
	We need a function that will send a json payload to a channel via a notifier,
	and a trigger that gets triggered upon insertion/update/deletion of a specific table -
	in our case, keycloak's event_entity
*/

var create_trigger string = `-- clean previous triggers
DROP TRIGGER IF EXISTS send_notif ON public.event_entity;
CREATE TRIGGER send_notif AFTER
INSERT
OR
DELETE
OR
UPDATE
ON
public.event_entity FOR EACH ROW EXECUTE FUNCTION notify_trigger();
`

/* 
	attention must be paid to the types of the table, we are sending the row as json, _but_ if one of the columns contains json whilst not being a json type, it must be converted:
	this script is aimed at keycloak, and keycloak loves storing json as varchar 2550
	thus before sending the payload, the contents must be cast as jsonb, otherwise quotes will appear escaped
	optionally the payload can be retrieved in go and converted there, in this case we went with the former option

	also sql is just goddamn weird, note that we're creating the payload _every time_ in the cases because this is the only way to prevent sql from going tits up, *i don't know why*
	*/

var create_function string = `-- clean previous functions
CREATE OR REPLACE FUNCTION public.notify_trigger()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
DECLARE
  rec RECORD;
  dat RECORD;
  payload TEXT;
BEGIN

  -- Set record row depending on operation
  CASE TG_OP
  WHEN 'UPDATE' THEN
     rec := NEW;
     dat := OLD;
  -- Build the payload
  payload := json_build_object(
	  'timestamp',CURRENT_TIMESTAMP,
	  'operation',LOWER(TG_OP),
	  'schema',TG_TABLE_SCHEMA,
	  'table',TG_TABLE_NAME,
	  'record',to_jsonb(rec) || jsonb_build_object('details_json', rec.details_json::jsonb),
	  'old',to_jsonb(dat)
  );
  WHEN 'INSERT' THEN
     rec := NEW;
  -- Build the payload
  payload := json_build_object(
	  'timestamp',CURRENT_TIMESTAMP,
	  'operation',LOWER(TG_OP),
	  'schema',TG_TABLE_SCHEMA,
	  'table',TG_TABLE_NAME,
	  'record',to_jsonb(rec) || jsonb_build_object('details_json', rec.details_json::jsonb)
  );
  WHEN 'DELETE' THEN
     rec := OLD;
  -- Build the payload
  payload := json_build_object(
	  'timestamp',CURRENT_TIMESTAMP,
	  'operation',LOWER(TG_OP),
	  'schema',TG_TABLE_SCHEMA,
	  'table',TG_TABLE_NAME,
	  'record',to_jsonb(rec) || jsonb_build_object('details_json', rec.details_json::jsonb)
  );
  ELSE
     RAISE EXCEPTION 'Unknown TG_OP: "%". Should not occur!', TG_OP;
  END CASE;


  -- Notify the channel
  PERFORM pg_notify('logins_logger',payload);

  RETURN rec;
END;
$function$
;
`
