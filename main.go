package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

var pageAccessToken string
var verifyToken string

func handleMessage(sender string, message *Message) {
	answer := SendMessage{}
	answer.Recipient.ID = sender
	answer.Message.Text = fmt.Sprintf("Hello! You said %q to me.", message.Text)
	fmt.Printf("Sending: %#v", answer)
	resp, err := callSendAPI(answer)
	if err != nil {
		log.Println(err)
	}

	log.Println("Response:", resp)
}

func handlePostbackMessage(sender string, message *Postback) {}

type SendMessage struct {
	Recipient struct {
		ID string `json:"id"`
	} `json:"recipient"`
	Message struct {
		Text string `json:"text"`
	} `json:"message"`
}

type Response struct {
	RecipientID string `json:"recipient_id"`
	MessageID   string `json:"message_id"`
}

func callSendAPI(message SendMessage) (*Response, error) {
	url := "https://graph.facebook.com/v2.6/me/messages?access_token=" + pageAccessToken

	b := new(bytes.Buffer)
	if err := json.NewEncoder(b).Encode(&message); err != nil {
		return nil, fmt.Errorf("encoding error: %v ", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, b)
	if err != nil {
		return nil, fmt.Errorf("request error: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("client error: %v", err)
	}

	resp := Response{}
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("decoding error: %v", err)
	}

	return &resp, nil
}

type Postback struct {
	Title    string `json:"title"`
	Payload  string `json:"payload"`
	Referral struct {
		Ref    string `json:"ref"`
		Source string `json:"source"`
		Type   string `json:"type"`
	} `json:"referral"`
}

type Message struct {
	MID        string `json:"mid"`
	Seq        int    `json:"seq"`
	Text       string `json:"text"`
	QuickReply struct {
		Payload string `json:"payload"`
	} `json:"quickreply"`
	Attachments []struct {
		Type    string `json:"type"`
		Payload struct {
			URL string `json:"payload"`
		} `json:"payload"`
	} `json:"attachments,omitempty"`
}

type Messaging struct {
	Sender struct {
		ID string `json:"id"`
	} `json:"sender"`
	Recipient struct {
		ID string `json:"id"`
	} `json:"recipient`
	Message  *Message  `json:"message,omitempty"`
	Postback *Postback `json:"postback,omitempty"`
}

type Callback struct {
	Object string `json:"object"`
	Entry  []struct {
		ID        string      `json:"string"`
		Time      int64       `json:"time"`
		Messaging []Messaging `json:"messaging"`
	} `json:"entry"`
}

func webhookHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		mode := r.URL.Query().Get("hub.mode")
		token := r.URL.Query().Get("hub.verify_token")
		challenge := r.URL.Query().Get("hub.challenge")

		if mode != "subscribe" || token != verifyToken {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}

		log.Print("Verification OK")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, challenge)
		return

	case http.MethodPost:
		cb := Callback{}
		err := json.NewDecoder(r.Body).Decode(&cb)
		if err != nil {
			log.Println(err)
		}

		if cb.Object != "page" {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		for _, entry := range cb.Entry {
			fmt.Println(len(entry.Messaging))
			for _, m := range entry.Messaging {
				if m.Message != nil {
					handleMessage(m.Sender.ID, m.Message)
				}

				if m.Postback != nil {
					handlePostbackMessage(m.Sender.ID, m.Postback)
				}
			}
		}

	default:
		http.Error(w, "No way", http.StatusForbidden)
	}
}

func main() {
	pageAccessToken = os.Getenv("PAGE_ACCESS_TOKEN")
	if pageAccessToken == "" {
		fmt.Println("PAGE_ACCESS_KEY not set")
		os.Exit(1)
	}

	verifyToken = os.Getenv("VERIFY_TOKEN")
	if verifyToken == "" {
		fmt.Println("VERIFY_TOKEN not set")
		os.Exit(1)
	}

	http.HandleFunc("/webhook", webhookHandler)
	log.Fatal(http.ListenAndServe(":8081", nil))
}
