// vim:ts=4:sw=4
// Library which implements the RZLBus standard.
// See http://raumzeitlabor.de/wiki/Hausbus2
// © 2012 Michael Stapelberg (see also: LICENSE)
package rzlbus

import (
	"http"
	"json"
	"log"
	"fmt"
	"io"
	"strings"
	"regexp"
	"encoding/base64"
	"bytes"
	"flag"
	"io/ioutil"
)

var (
	http_listen = flag.String("rzlbus_listen", "*:10443",
    	"Which host:port the (SSL) webserver should listen on.")
	ssl_key = flag.String("rzlbus_ssl_key", "server.key",
    	"Path to the SSL private key (PEM).")
	ssl_cert = flag.String("rzlbus_ssl_cert", "server.crt",
    	"Path to the SSL certificate (PEM, also containing CA).")
)

type StateModifiedCallback func(key string, oldValue interface{}, newValue interface{})

type stateEntry struct {
	Value interface{}

	// Whether anyone can overwrite this externally (POST /_/state).
	Writable bool

	ModifiedCallback StateModifiedCallback
}

var state map[string] stateEntry

// Hopefully something like this shows up in the standard http package soon :).
var basic_auth_rgx = regexp.MustCompile(`^Basic ([a-zA-Z0-9\+/=]+)`)

func GetBasicAuth(r *http.Request) (user, passwd string) { 
	auth := r.Header.Get("Authorization") 
	if auth == "" { 
		return 
	} 
	m := basic_auth_rgx.FindStringSubmatch(auth) 
	if len(m) != 2 { 
		return 
	} 
	buf, err := base64.StdEncoding.DecodeString(m[1]) 
	if err != nil { 
		return 
	} 
	up := bytes.SplitN(buf, []byte{':'}, 2) 
	if len(up) != 2 { 
		return 
	} 
	return string(up[0]), string(up[1]) 
} 

// Handler for requests to /_/state/(.*)
// Dumps the whole state by default (GET /_/state)
// Filters, then dumps the whole state (GET /_/state/something)
// Updates state externally (POST /_/state/)
func handle_state(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		// Extract the filter from the path, if any.
		filter := r.URL.Path[len("/_/state/"):]

		// Transform 'state' into a simple map, filter if necessary.
		filteredState := make(map[string] interface{}, 50)
		for key, entry := range state {
			if filter == "" || strings.HasPrefix(key, filter) {
				filteredState[key] = entry.Value
			}
		}

		// Send the map as nicely indented JSON.
		bytes, err := json.MarshalIndent(filteredState, "", " ")
		if err != nil {
			log.Fatal("JSON encoding for /_/state/:", err)
		}
		w.Write(bytes)
	} else if r.Method == "POST" {
		body, _ := ioutil.ReadAll(r.Body)
		r.Body.Close()
		var postedState map[string]interface{}
		if err := json.Unmarshal(body, &postedState); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			io.WriteString(w, "Could not decode JSON into a single map: ")
			io.WriteString(w, err.String())
			return
		}

		// First verify that all posted keys are actually writable, so that we
		// don’t end up in the situation where we applied half of the input and
		// reject the other half.
		for postedKey, _ := range(postedState) {
			entry, ok := state[postedKey]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				fmt.Fprintf(w, `The key "%s" was not found.`, postedKey)
				return
			}
			if !entry.Writable {
				w.WriteHeader(http.StatusForbidden)
				fmt.Fprintf(w, `The key "%s" is not writable.`, postedKey)
				return
			}
		}

		// By now, we know that all keys are present and can be written to.
		for postedKey, postedValue := range(postedState) {
			entry := state[postedKey]
			oldValue := entry.Value
			entry.Value = postedValue
			state[postedKey] = entry

			entry.ModifiedCallback(postedKey, oldValue, postedValue)
		}

		io.WriteString(w, "State modified successfully.")
	} else {
		w.WriteHeader(http.StatusNotImplemented)
		io.WriteString(w, "HTTP Method not implemented, use GET or POST")
	}
}

// Handler for /_/reboot (POST, needs authentication)
func handle_reboot(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusNotImplemented)
		io.WriteString(w, "HTTP Method not implemented, use POST")
		return
	}
	user, passwd := GetBasicAuth(r)
	// TODO: make this configurable
	if user != "foo" || passwd != "bar" {
		w.WriteHeader(http.StatusForbidden)
		io.WriteString(w, "Specify user and password")
		return
	}

	fmt.Fprint(w, "I would totaly reboot now.")
	// TODO: actually reboot, if a flag is set
}

// Initializes the data structures, sets up the HTTP server.
// If you need any custom HTTP handlers, you need to register them by calling
// http.HandleFunc() before calling rzlbus.Init().
func Init() {
	// Create the state map with a capacity hint of 50.
	// 50 is a lot more than we probably need to store.
	state = make(map[string] stateEntry, 50)

	// Register the common RZLBus handlers.
	http.HandleFunc("/_/state/", handle_state)
	http.HandleFunc("/_/reboot", handle_reboot)

	// Handle HTTP requests in a different Goroutine.
	go func() {
		err := http.ListenAndServeTLS(*http_listen, *ssl_cert, *ssl_key, nil)
		if err != nil {
			log.Fatal("ListenAndServeTLS: ", err.String())
		}
	}()
}

// Adds an entry to the exported state (read-only for HTTP clients).
// If you want to export state which can be written to, use SetWritableState.
func SetState(key string, value interface{}) {
	var entry stateEntry
	entry.Value = value
	entry.Writable = false
	state[key] = entry
}

// Adds an entry to the exported state which can be modified externally.
// The specified callback will be invoked whenever the state was modified.
func SetWritableState(key string, value interface{}, callback StateModifiedCallback) {
	var entry stateEntry
	entry.Value = value
	entry.Writable = true
	entry.ModifiedCallback = callback
	state[key] = entry
}
