package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"

	cache "github.com/patrickmn/go-cache"
)

/*
 * Routes
 */

// SMS API sends
func receivePlivoSMS(w http.ResponseWriter, r *http.Request) {

	// log.Println("receivePlivoSMS", r.Method)

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", 405)
		return
	}

	plivoHmac := r.Header.Get("X-Plivo-Signature")

	if plivoHmac == "" {
		log.Println("ERROR: No Plivo HMAC", r.RemoteAddr)
		http.Error(w, "Invalid Request", 400)
		return
	}

	// PostFormValue & FormValue both already call this if needed
	r.ParseForm()

	// Verify a valid message
	params := sortReqForm(r)
	url := absoluteRequestURL(r)
	hmac := ComputeHmac(url+params, configuration.PlivoAuthToken)

	log.Println("url+params", url+params)
	// log.Println("hmac", hmac, "X-Plivo-Signature", plivoHmac)

	if plivoHmac != hmac {
		log.Println("ERROR: invalid HMAC")
		http.Error(w, "Invalid SMS Request", 400)
		return
	}

	fromPhone := r.PostFormValue("From")

	// Get POST data
	sms := plivoSms{
		To:   r.PostFormValue("To"),
		From: r.PostFormValue("From"),
		// TotalRate string
		// Units string
		Text: strings.TrimSpace(r.PostFormValue("Text")),
		// TotalAmount string
		// type:        r.PostFormValue("Type"),
		MessageUUID: r.PostFormValue("MessageUUID"),
	}

	// Now that we saved it in the DB, hash the phone for the UI
	sms.From = hashPhone(sms.From)

	// Look for a hashtag
	hashtag, text := extractHashtag(sms.Text)

	if hashtag == "" {
		// Try to find it from previous txts
		hashtag = lookupHashtagFromPhone(fromPhone)

		if hashtag == "" {
			log.Println("ERROR: no hashtag")
			// return
		}
	}

	// Save for next request
	if hashtag != "" {
		phoneToHashtagMap.Set(fromPhone, hashtag, cache.DefaultExpiration)
	}

	// This might be a pre-message text just to register on this hashtag
	if ("#" + hashtag) == text {
		return
	}

	// Leaving the hashtag in the text helps the audience not forget it
	// sms.Text = text

	var jsonRes []byte
	// Is this a vote?
	vote := extractPollVote(text)
	if vote != "" {

		// Re-save now that the vote was extracted
		sms.Text = vote
		jsonRes, _ = json.Marshal(sms)

	} else {
		jsonRes, _ = json.Marshal(sms)
	}

	// Tell the client we're good
	w.Write([]byte("200"))

	if hashtag != "" {
		if vote != "" {
			sio.BroadcastTo(hashtag+"-guest", "vote", string(jsonRes))
			sio.BroadcastTo(hashtag, "vote", string(jsonRes))
		} else {
			sio.BroadcastTo(hashtag, "message", string(jsonRes))
		}
		return
	}

	// They did not send a hashtag ...and we can't ask for confirmation
	for i, k := range hashmap.Keys() {

		if i > 30 {
			break
		}

		// Don't send directly to guest view!
		if strings.Contains(k, "-guest") {
			continue
		}

		// Just tell everyone
		if vote != "" {
			sio.BroadcastTo(k+"-guest", "vote", string(jsonRes))
			sio.BroadcastTo(k, "vote", string(jsonRes))
		} else {
			sio.BroadcastTo(k, "message", string(jsonRes))
		}
	}
}

func serveHostView(w http.ResponseWriter, r *http.Request) {

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}

	// Remove the URL prefix
	// phone := r.URL.Path[4:]
	phone := randomPhoneForTexting()
	fmt.Println("phone", phone)

	if _, err := strconv.Atoi(phone); err != nil {
		http.Error(w, "Invalid Phone Number", 400)
	}

	homeTempl, err := template.ParseFiles("templates/host.html")

	if err != nil {
		log.Println(err)
		http.Error(w, "Server Error", 500)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	homeTempl.Execute(w, map[string]string{"phone": phone})
}

/*
 * Read-only page for whatever the presenter wants to display
 */
func serveGuestView(w http.ResponseWriter, r *http.Request) {

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}

	// Remove the URL prefix
	hashtag := r.URL.Path[7:]

	if onlyAlphanumeric(hashtag) == false {
		// log.Println("ERROR: invalid hashtag", hashtag)
		http.Error(w, "Invalid Presentation Hashtag", 400)
		return
	}

	phone := randomPhoneForTexting()

	homeTempl, err := template.ParseFiles("templates/guest.html")

	if err != nil {
		log.Println(err)
		http.Error(w, "Server Error", 500)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	homeTempl.Execute(w, map[string]string{"phone": phone, "hashtag": hashtag})
}
