package main

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"math/rand"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

func absoluteRequestURL(r *http.Request) string {

	if r.URL.IsAbs() {
		return r.URL.String()
		// url = r.URL.Scheme + "://" + r.URL.Host
	}

	return configuration.URL + r.URL.Path

}

func sortReqForm(req *http.Request) (params string) {

	req.ParseForm()

	var keys []string
	for k := range req.Form {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		// fmt.Println("Key:", k, "Value:", req.Form[k])
		params += k + req.Form[k][0]
	}

	return
}

func randomPhoneForTexting() string {
	if len(configuration.PhoneNumbers) > 1 {
		return string(configuration.PhoneNumbers[rand.Intn(len(configuration.PhoneNumbers))])
	}

	return string(configuration.PhoneNumbers[0])
}

// func currentURL(r *http.Request) string {
// 	u := r.URL
//
// 	// The scheme is http because that's the only protocol your server handles.
// 	u.Scheme = "http"
//
// 	// If client specified a host header, then use it for the full URL.
// 	u.Host = r.Host
//
// 	// Otherwise, use your server's host name.
// 	if u.Host == "" {
// 		u.Host = "your-host-name.com"
// 	}
//
// }

// ComputeHmac to Verify Plivo actually sent this message
func ComputeHmac(message string, secret string) string {
	key := []byte(secret)
	h := hmac.New(sha1.New, key)
	h.Write([]byte(message))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// var nexmocidr = []string{
// 	"192.168.1.0/24",
// 	"192.168.2.0/24",
// }
//
// func validNexmoIp(ip string) (found bool) {
// 	myaddr := net.ParseIP(ip)
// 	for _, cidr := range nexmocidr {
// 		_, cidrnet, err := net.ParseCIDR(cidr)
//
// 		if err != nil {
// 			panic(err) // assuming I did it right above
// 		}
//
// 		if cidrnet.Contains(myaddr) {
// 			found = true
// 			// fmt.Println(cidr, "contains", ip)
// 			break
// 		} else {
// 			// fmt.Println(cidr, "does not contain", ip)
// 		}
//
// 	}
//
// 	return
// }

// https://play.golang.org/p/BUD-aTSO-c
// func disguisePhone(phone string) string {
// 	area_code := phone[1:4]
//
// 	h := sha1.New()
// 	h.Write([]byte(phone))
//   sha1_hash := hex.EncodeToString(h.Sum(nil))
//
// 	return area_code + "-" + sha1_hash[:3] + "-" + sha1_hash[3:7]
// }

// Hash the phone number
func hashPhone(phone string) string {
	h := sha1.New()
	h.Write([]byte(phone + configuration.RandomHashValue))
	return hex.EncodeToString(h.Sum(nil))
}

/*
 * Used for hashtags
 * https://play.golang.org/p/joKU6yIneg
 */
func onlyAlphanumeric(s string) bool {
	r, _ := regexp.Compile("^[a-z0-9]{2,}$")
	return r.MatchString(s)
}

/*
 * Cleanup when registering a new hashtag, or closing socket
 */
func clearSocketHashtag(socketID string) {
	if tmp, ok := hashmap.Get(socketID); ok {
		currentHashtag := tmp.(string)
		// log.Println("removing hold on", currentHashtag, "by", socketID)
		hashmap.Remove(currentHashtag)
		hashmap.Remove(socketID)
	}
}

// Probably be at the end or begining of string
func extractHashtag(s string) (string, string) {
	r, _ := regexp.Compile("#\\w+")
	hashtag := r.FindString(s)
	if hashtag != "" {
		hashtag = hashtag[1:]
	}
	s = r.ReplaceAllString(s, "")

	// final cleanup needed because there might be space around #hashtag
	s = stringWhitespaceMinifier(s)

	return strings.ToLower(hashtag), strings.TrimSpace(s)
}

// https://play.golang.org/p/7XgjWNqXr-
func extractPollVote(s string) (vote string) {
	r, _ := regexp.Compile("^\\W{0,5}([a-dA-D])\\W{0,5}$")
	match := r.FindStringSubmatch(s)

	if len(match) == 2 {
		vote = strings.ToLower(match[1])
	}
	return
}

// https://play.golang.org/p/XgrEodVwB6
func stringWhitespaceMinifier(in string) (out string) {
	white := false
	for _, c := range in {
		if unicode.IsSpace(c) {
			if !white {
				out = out + " "
			}
			white = true
		} else {
			out = out + string(c)
			white = false
		}
	}
	return
}

/*
 * Only dealing with simple key->value pairs for routing data
 */
// func decodeJSONMessage(message string) (data map[string]string, err error) {
// 	err = json.Unmarshal([]byte(message), &data)
// 	return
// }
