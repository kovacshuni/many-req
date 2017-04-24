package main

import (
	"fmt"
	"net"
	"net/http"
	"sync/atomic"
	"time"
	"bytes"
	"os"
	"os/signal"
)

// App is the app
type App struct {
	okCount      *uint64
	notOkCount   *uint64
	errBuffer    *bytes.Buffer
	httpClient   *http.Client
	ticker       *time.Ticker
}

func main() {
	app := newApp()
	go app.report()
	app.requestManyConcurrently()
	defer func() {
		fmt.Print(app.errBuffer.String())
	}()
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	<- interrupt
}

func newApp() App {
	var okCount uint64
	var notOkCount uint64
	var errBuffer bytes.Buffer
	return App{
		okCount:      &okCount,
		notOkCount:   &notOkCount,
		errBuffer:    &errBuffer,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				DialContext: (&net.Dialer{
					Timeout:   30 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
				MaxIdleConnsPerHost: 100,
			},
		},
		ticker: time.NewTicker(time.Second),
	}
}

func (app App) requestManyConcurrently() {
	for i := 0; i < 8; i++ {
		go app.requestMany()
	}
}

func (app App) requestMany() {
	for {
		req, err := http.NewRequest("GET", "http://localhost:8080/", nil)
		if err != nil {
			app.errBuffer.WriteString(err.Error())
			app.errBuffer.WriteString("\n")
		}
		resp, err := app.httpClient.Do(req)
		if err != nil {
			app.errBuffer.WriteString(err.Error())
			app.errBuffer.WriteString("\n")
		} else {
			if resp.StatusCode != http.StatusOK {
				atomic.AddUint64(app.notOkCount, 1)
			} else {
				atomic.AddUint64(app.okCount, 1)
			}
		}
		//time.Sleep(1 * time.Millisecond)
	}
}

func (app App) report() {
	for range app.ticker.C {
		okCount := atomic.LoadUint64(app.okCount)
		atomic.StoreUint64(app.okCount, 0)
		notOkCount := atomic.LoadUint64(app.notOkCount)
		atomic.StoreUint64(app.notOkCount, 0)
		fmt.Printf("okCount=%d notOkCount=%d\n", okCount, notOkCount)
		fmt.Print(app.errBuffer.String())
		app.errBuffer.Reset()
	}
}
