package spoterm

import (
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

var spotermTests = []struct {
	handler  func(w http.ResponseWriter, r *http.Request)
	termTime time.Time
	chOpen   bool
	expErr   error
}{
	{ // termination time not set, 404
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		},
		time.Time{},
		true,
		nil,
	},

	// termination time not set, non-time data present
	{
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("foo"))
		},
		time.Time{},
		true,
		nil,
	},

	// termination time set
	{
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("2015-02-04T21:22:49Z"))
		},

		time.Unix(0, 1423084969000000000).UTC(),
		true,
		nil,
	},
}

func TestSpotermNofify(t *testing.T) {

	tp, nt, pi := termPath, httpTimeout, pollInterval
	defer func() { termPath, httpTimeout, pollInterval = tp, nt, pi }()
	pollInterval = 400 * time.Millisecond
	httpTimeout = 100 * time.Millisecond

	for _, tc := range spotermTests {
		// Set up http server
		server := httptest.NewServer(http.HandlerFunc(tc.handler))
		defer server.Close()
		termPath = server.URL

		c, err := SpotermNotify()
		if err != nil {
			if strings.Contains(err.Error(), tc.expErr.Error()) {
				return // error expected
			}
			t.Fatal(err)
		}
		if tc.expErr != nil {
			t.Fatalf("expected error: %v", tc.expErr)
		}
		tmr := time.NewTimer(900 * time.Millisecond)
		select {
		case time, ok := <-c:
			t.Log("time received: ", time, " ", tc.termTime)
			if time != tc.termTime {
				t.Fatalf("expected: %v, got %v", tc.termTime, time)
			}
			if ok != tc.chOpen {
				t.Fatalf("expected channel open: %v, got %v", tc.termTime, time)
			}
		case <-tmr.C:
			tmr.Stop()
			t.Logf("timeout")
			if !tc.chOpen {
				t.Fatalf("expected channel closed")
			}
		}
		server.Close()
	}
}

func TestSpotermNotifyNotEC2(t *testing.T) {
	httpTimeout = 200 * time.Millisecond
	_, err := SpotermNotify()
	if !(err != nil && strings.Contains(err.Error(), "must run on EC2 instance")) {
		t.Fatal(err)
	}
	t.Log(err)

}

func ExampleSpotermNotify() {
	ch, err := SpotermNotify()
	if err != nil {
		// handle error
		// an error will occur if run on a non-ec2 instance
		log.Fatal(err)
	}
	go func() {
		if t, ok := <-ch; ok {
			// received termination-time
			// run cleanup actions here
			log.Printf("the instance will be terminated at %v", t)

		} else {
			log.Printf("SpotermNotify channel closed due to error")
		}
	}()
}
