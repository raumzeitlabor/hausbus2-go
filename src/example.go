// vim:ts=4:sw=4
// Example for the rzlbus Go library. Roughly resembles the Pinpad controller.
// © 2012 Michael Stapelberg (see also: LICENSE)

package main

import (
	"fmt"
	"http"
	"rzlbus"
	"time"
	"io"
	"flag"
)

func requireAuth(w http.ResponseWriter, r *http.Request) bool {
	user, passwd := rzlbus.GetBasicAuth(r)
	if user != "foo" || passwd != "bar" {
		w.WriteHeader(http.StatusForbidden)
		io.WriteString(w, "Specify user and password")
		return false
	}

	return true
}

func handleLock(w http.ResponseWriter, r *http.Request) {
	if !requireAuth(w, r) {
		return
	}

	fmt.Fprint(w, "I would lock the door now.");
}

func handleUnlock(w http.ResponseWriter, r *http.Request) {
	if !requireAuth(w, r) {
		return
	}

	fmt.Fprint(w, "I would unlock the door now.");
}

func main() {
	flag.Parse()

	fmt.Println("ohai")

	http.HandleFunc("/lock_door", handleLock)
	http.HandleFunc("/unlock_door", handleUnlock)
	rzlbus.Init()
	rzlbus.SetState("pinpad.door", "locked")
	rzlbus.SetWritableState("pinpad.msg", "",
		func(key string, oldValue interface{}, newValue interface{}) {
			fmt.Println("I should change the LCD message to:")
			switch vv := newValue.(type) {
			case string:
				fmt.Println(vv)
			default:
				fmt.Println("ERROR: The message is not of type string.")
			}
		})

	for {
		rzlbus.SetState("pinpad.door", "locked")
		time.Sleep(1 * 1000 * 1000)
		rzlbus.SetState("pinpad.door", "open")
		time.Sleep(1 * 1000 * 1000)
	}
}
