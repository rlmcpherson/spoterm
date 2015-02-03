package spoterm

import (
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

	// termination time set after x seconds
	//
	//not ec2 instance, timeout request
	//{
	//func(w http.ResponseWriter, r *http.Request) {
	//},
	//time.Time{},
	//true,
	//fmt.Errorf("must be run on EC2 instance"),
	//},
}

func TestSpotermChan(t *testing.T) {

	tp, nt, pi := termPath, netTimeout, pollInterval
	defer func() { termPath, netTimeout, pollInterval = tp, nt, pi }()
	pollInterval = 400 * time.Millisecond
	netTimeout = 100 * time.Millisecond

	for _, tc := range spotermTests {
		// Set up http server
		server := httptest.NewServer(http.HandlerFunc(tc.handler))
		defer server.Close()
		termPath = server.URL

		c, err := SpotermChan()
		if err != nil &&
			err != tc.expErr &&
			!strings.Contains(err.Error(), tc.expErr.Error()) {
			t.Fatal(err)
		}
		tmr := time.NewTimer(1300 * time.Millisecond)
		select {
		case time, ok := <-c:
			t.Log("time received: ", time, " ", tc.termTime)
			if time != tc.termTime {
				t.Fatalf("expected: %s, got %s", tc.termTime, time)
			}
			if ok != tc.chOpen {
				t.Fatalf("expected channel open: %s, got %s", tc.termTime, time)
			}
		case <-tmr.C:
			tmr.Stop()
			t.Logf("time not received")
		}
		server.Close()
	}
}

func TestSpotermChanNotEC2(t *testing.T) {
	_, err := SpotermChan()
	if !(err != nil && strings.Contains(err.Error(), "must run on EC2 instance")) {
		t.Fatal(err)
	}

}