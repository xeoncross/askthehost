package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Pallinder/go-randomdata"
	"github.com/googollee/go-socket.io"
	"github.com/patrickmn/go-cache"
	"github.com/streamrail/concurrent-map"
)

// Configuration is read from config.json file
type Configuration struct {
	PlivoAuthToken  string
	PlivoAuthID     string
	PhoneNumbers    []string
	RandomHashValue string
	URL             string
	Port            string
}

// MessageType is any type of message sent by the presenter
type MessageType struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// Default params, multi-part texts and photo texts might have other fields...
type plivoSms struct {
	To   string `json:"to"`
	From string `json:"from"`
	// TotalRate string
	// Units string
	Text string `json:"message"`
	// TotalAmount string
	// type        string
	MessageUUID string `json:"id"`
	SentOn      time.Time
}

// Global config object
var configuration Configuration

// Keeping a dictionary of all registered hashtags->socket.Id() and vis-versa
var hashmap cmap.ConcurrentMap

// Global mapping of phone numbers to hashtags
var phoneToHashtagMap *cache.Cache

// Global socket server instance
var sio *socketio.Server

func main() {

	// Sets the number of maxium goroutines to the 2*numberCPU + 1
	// runtime.GOMAXPROCS()

	var err error

	file, err := os.Open("config.json")
	if err != nil {
		log.Fatal(err)
	}

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&configuration)
	if err != nil {
		log.Fatal("Error loading config file:", err)
	}

	// Configuring socket.io Server
	sio, err = socketio.NewServer(nil)
	if err != nil {
		log.Fatal(err)
	}

	hashmap = cmap.New()

	// Create a cache with a default expiration time of 5 minutes, and which
	// purges expired items every 30 seconds
	// This is needed because Plivo doesn't give information about concatenated
	// messages so we have to guess
	// messageCache := cache.New(5*time.Minute, 30*time.Second)

	// People forget to include the hashtag after the first message
	// This will help us use hashtags from previous txts
	phoneToHashtagMap = cache.New(12*time.Hour, 5*time.Minute)

	/*
	 * There are two types of clients: presenter and viewer
	 * [presenter] needs to join a room of his socket.io so we can
	 * alert him when new texts come in.
	 * [viewer] needs to join a hashtag room so presenter can send
	 * him messages to display.
	 */
	sio.On("connection", func(so socketio.Socket) {

		// So we can emit to the client like javascript version
		so.Join(so.Id())

		so.On("disconnection", func() {
			clearSocketHashtag(so.Id())
		})

		// Used by viewer to join a room matching the URL hashtag
		so.On("join room", func(room string) {
			if onlyAlphanumeric(room) == false {
				return
			}

			so.Join(room + "-guest")

			// Tell the presenter a guest joined
			so.BroadcastTo(room, "guest joined")
		})

		// Each presenter can only register a single hashtag (needed before anything else)
		so.On("register hashtag", func(hashtag string) string {

			// We might want to support unicode later on
			// Then again, we only have US carriers and the URL must be ASCII
			if onlyAlphanumeric(hashtag) == false {
				log.Println("ERROR: onlyAlphanumeric", hashtag)
				return "0"
			}

			if tmp, ok := hashmap.Get(hashtag); ok {
				id := tmp.(string)

				if id == so.Id() {
					return "1"
				}

				// log.Println("already taken by", id)
				return "0"
			}

			// This socket may or may not have already registered a hashtag
			clearSocketHashtag(so.Id())

			hashmap.Set(hashtag, so.Id())
			hashmap.Set(so.Id(), hashtag) // so we can lookup hashtag on disconnect

			// log.Println("registered", hashtag, so.Id())
			// log.Println(hashmap.Items())

			// Only registered people can pick a room to join
			so.Join(hashtag)

			// Tell any guests a presenter joined
			so.BroadcastTo(hashtag+"-guest", "presenter joined")

			return "1"
		})

		// Presenter publishing a message to viewers
		so.On("publish", func(data string) {

			log.Println("publish", data)

			var msg MessageType
			err = json.Unmarshal([]byte(data), &msg)
			if err != nil {
				log.Println(err, data)
			}

			if tmp, ok := hashmap.Get(so.Id()); ok {
				hashtag := tmp.(string)
				// log.Println("broadcastTo", hashtag+"-guest")
				// so.BroadcastTo(hashtag+"-guest", "message", data)
				so.BroadcastTo(hashtag+"-guest", msg.Type, msg.Data)
				return
			}

			// No registered hashmap yet? Weird...
			log.Println("ERROR: no hashtag yet")
		})

		// Presenter publishing poll to viewers
		// so.On("poll", func(data string) {
		//
		// 	log.Println("poll", data)
		//
		// 	// No registered hashmap yet? Weird...
		// 	if tmp, ok := hashmap.Get(so.Id()); ok {
		// 		hashtag := tmp.(string)
		// 		log.Println("broadcastTo", hashtag+"-guest")
		// 		so.BroadcastTo(hashtag+"-guest", "poll", data)
		// 		return
		// 	}
		//
		// })

	})
	sio.On("error", func(so socketio.Socket, err error) {
		log.Println("ERROR:", err)
	})

	// Send a random sample message every X seconds
	sendDebugMessages(sio)

	// Send a random vote every X seconds
	sendDebugVotes(sio)

	// Sets up the handlers and listen on port 8080
	http.Handle("/socket.io/", sio)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))

	http.HandleFunc("/about/", serveGuestView)

	http.HandleFunc("/start/", serveHostView)

	http.HandleFunc("/receivePlivoSMS/", receivePlivoSMS)

	http.Handle("/", http.FileServer(http.Dir("./templates/")))

	log.Println("listening on", configuration.Port)
	err = http.ListenAndServe(configuration.Port, nil)
	log.Println(err)
}

/*
 * Helper Functions
 */

// On first (correct) text, store the hashtag-to-phone relation
func lookupHashtagFromPhone(phone string) (hashtag string) {
	if x, found := phoneToHashtagMap.Get(phone); found {
		hashtag = x.(string)
	}
	return
}

func checkAuth(w http.ResponseWriter, r *http.Request, user string, pass string) bool {
	s := strings.SplitN(r.Header.Get("Authorization"), " ", 2)
	if len(s) != 2 {
		return false
	}

	b, err := base64.StdEncoding.DecodeString(s[1])
	if err != nil {
		return false
	}

	pair := strings.SplitN(string(b), ":", 2)
	if len(pair) != 2 {
		return false
	}

	return pair[0] == user && pair[1] == pass
}

/*
 * Pretend to send SMS messages
 */
func sendDebugMessages(sio *socketio.Server) {

	// Temp Debug Code
	ticker := time.NewTicker(5 * time.Second)
	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:

				randID := strings.Replace(strconv.FormatFloat(rand.Float64(), 'f', 15, 64), "0.", "", -1)

				randomMessage := "Is it true that " +
					randomdata.FullName(randomdata.RandomGender) +
					" was born on " +
					" " + randomdata.Month() + "/" + fmt.Sprintf("%d", time.Now().Year()-time.Now().Hour()) +
					" at " + randomdata.Street() + "?"

				// Send a sample message every whatever time
				res := map[string]interface{}{
					"from":    hashPhone("800-222-2222"),
					"id":      randID,
					"message": randomMessage,
					// "dateTime": time.Now().UTC().Format(time.RFC3339),
					"type": "message",
				}
				jsonRes, _ := json.Marshal(res)
				sio.BroadcastTo("test", "message", string(jsonRes))
				sio.BroadcastTo(fmt.Sprintf("test%d", time.Now().Year()), "message", string(jsonRes))
				// sio.BroadcastTo("hashtagoneforgood", "message", string(jsonRes))
				// log.Println("message")

			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()
}

/*
 * Pretend to send SMS messages
 */
func sendDebugVotes(sio *socketio.Server) {

	// Temp Debug Code
	ticker := time.NewTicker(1 * time.Second)
	quit := make(chan struct{})
	go func() {

		voteOptions := []string{"a", "b", "c"}

		for {
			select {
			case <-ticker.C:

				randID := strings.Replace(strconv.FormatFloat(rand.Float64(), 'f', 15, 64), "0.", "", -1)

				vote := voteOptions[rand.Intn(len(voteOptions))]

				// Send a sample message every whatever time
				res := map[string]interface{}{
					"from":    hashPhone("800-222-2222"),
					"id":      randID,
					"message": vote,
					"type":    "message",
				}
				jsonRes, _ := json.Marshal(res)

				sio.BroadcastTo("test-guest", "vote", string(jsonRes))
				sio.BroadcastTo("test", "vote", string(jsonRes))

				y := fmt.Sprintf("%d", time.Now().Year())
				sio.BroadcastTo("test"+y+"-guest", "vote", string(jsonRes))
				sio.BroadcastTo("test"+y, "vote", string(jsonRes))

			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()
}
