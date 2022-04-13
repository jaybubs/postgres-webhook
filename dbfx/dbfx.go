package dbfx

// functions for db connection and execution

import (
	"fmt"
	"time"

	. "pgwebhook/utilities"

	"github.com/lib/pq"
)

func Create_Listener(connection string, tvars *Template_vars) (*pq.Listener, error) {
	prob := func(ev pq.ListenerEventType, err error) {
		CE(err)
	}
	listener := pq.NewListener(connection, 1*time.Second, 60*time.Second, prob)
	err := listener.Listen(tvars.Pgchannel)
	CE(err)
	fmt.Println("listener created")
	return listener, nil
}

/*
	The listener consists of two cases: a notification is sent, or time has elapsed
	this will ensure that the connection is being kept alive. If desired "Pl(pinging)" and "load" can be uncommented and returned in the second case for debugging purposes
*/
func Listen(l *pq.Listener,ch chan<- string) {
	for {
		select {
		// listener comes with a channel
		case n := <-l.Notify:
			fmt.Println("Received data from channel [", n.Channel, "] :")
			load := n.Extra
			ch <-load

		case <-time.After(90*time.Second):
			// run concurrently
			go func() {
				l.Ping()
			}()
			ch <-""
		}
	}
}
