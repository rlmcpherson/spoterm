package spoterm

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"
)

//TODO:
// remove logging
// write example test

const (
	timeFormat = "2006-01-02T15:04:05Z" // from AWS docs, see url below
)

var termPath string = "http://169.254.169.254/latest/meta-data/spot/termination-time"
var pollInterval time.Duration = 5 * time.Second
var netTimeout time.Duration = 2 * time.Second

// SpotermChan returns a channel of time.Time or an error. When the spot termination-time is set in instance metadata, the time is sent on the channel. If an unrecoverable error occurs after initialization, the channel is closed.  The termination-time is polled every 5 seconds, giving a minimum of 115 seconds from notification to termination.
// See http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/spot-interruptions.html
func SpotermChan() (chan time.Time, error) {
	tc := make(chan time.Time, 1)
	ticker := time.NewTicker(pollInterval)

	// poll for errors once before returning
	if _, err := pollInstanceMetadata(); err != nil {
		close(tc)
		return nil, err
	}
	go func() {
		for _ = range ticker.C {
			ts, err := pollInstanceMetadata()
			if err != nil || !ts.IsZero() {
				ticker.Stop()
				tc <- ts
				close(tc)
				break
			}

		}
	}()

	return tc, nil
}

//
func pollInstanceMetadata() (t time.Time, err error) {
	client := http.Client{Timeout: netTimeout}
	resp, err := client.Get(termPath)
	if err != nil {
		if strings.Contains(err.Error(), "request canceled") {
			return t, fmt.Errorf("must run on EC2 instance: %v", err)
		}
		return
	}
	defer resp.Body.Close()
	// 404, no termination-time set yet
	if resp.StatusCode == 404 {
		return
	}
	if resp.StatusCode != 200 {
		return t, fmt.Errorf("reponse error %d", resp.StatusCode)
	}
	ts, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	log.Printf("body: %s", ts)
	if t, err = time.Parse(timeFormat, string(ts)); err != nil {
		log.Println(err)
	}
	return t, nil
}