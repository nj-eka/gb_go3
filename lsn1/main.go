package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	errorsLimit  = 100000
	resultsLimit = 10000
)

var (
	url        string
	depthLimit int
	timeout    int
)

func init() {
	flag.StringVar(&url, "url", "", "url address")
	flag.IntVar(&depthLimit, "depth", 3, "max depth for run")
	flag.IntVar(&timeout, "timeout", 10, "timeout in seconds")
	flag.Parse()

	if url == "" {
		log.Print("no url set by flag")
		flag.PrintDefaults()
		os.Exit(1)
	}
}

func main() {
	started := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	go watchSignals(cancel)
	defer cancel()

	crawler := newCrawler(depthLimit)

	go watchDepth(ctx, crawler, 2)

	results := make(chan crawlResult)

	done := watchCrawler(ctx, results, errorsLimit, resultsLimit)

	crawler.run(ctx, url, results, 0)

	<-done

	log.Println(time.Since(started))
}

func watchSignals(cancel context.CancelFunc) {
	osSignalChan := make(chan os.Signal)

	signal.Notify(osSignalChan,
		syscall.SIGINT,
		syscall.SIGTERM)

	sig := <-osSignalChan
	log.Printf("got signal %q", sig.String())

	cancel()
}

func watchCrawler(ctx context.Context, results <-chan crawlResult, maxErrors, maxResults int) chan struct{} {
	readersDone := make(chan struct{})

	go func() {
		defer close(readersDone)
		for {
			select {
			case <-ctx.Done():
				return

			case result := <-results:
				if result.err != nil {
					maxErrors--
					if maxErrors <= 0 {
						log.Println("max errors exceeded")
						return
					}
					continue
				}

				log.Printf("crawling result: %v", result.msg)
				maxResults--
				if maxResults <= 0 {
					log.Println("got max results")
					return
				}
			}
		}
	}()

	return readersDone
}

func watchDepth(ctx context.Context, c *crawler, n int) {
	ctxSig, stop := signal.NotifyContext(context.Background(), syscall.SIGUSR1)
	defer stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ctxSig.Done():
			log.Printf("got signal %q", syscall.SIGUSR1)
			c.addDepth(n)
		}
	}
}
