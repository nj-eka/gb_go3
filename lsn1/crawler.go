package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"
)

type crawlResult struct {
	err error
	msg string
}

type crawler struct {
	sync.Mutex
	visited       map[string]string
	maxDepth      int
	maxDepthMutex sync.RWMutex
}

func (c *crawler) validDepth(depth int) bool {
	c.maxDepthMutex.RLock()
	defer c.maxDepthMutex.RUnlock()
	return depth <= c.maxDepth
}

func (c *crawler) addDepth(step int) {
	c.maxDepthMutex.Lock()
	defer c.maxDepthMutex.Unlock()
	c.maxDepth += step
}

func newCrawler(maxDepth int) *crawler {
	return &crawler{
		visited:  make(map[string]string),
		maxDepth: maxDepth,
	}
}

func (c *crawler) run(ctx context.Context, url string, results chan<- crawlResult, depth int) {
	time.Sleep(2 * time.Second)

	select {
	case <-ctx.Done():
		return
	default:
		if !c.validDepth(depth) {
			return
		}

		page, err := parse(url)
		if err != nil {
			results <- crawlResult{
				err: errors.Wrapf(err, "parse page %s", url),
			}
			return
		}

		title := pageTitle(page)
		links := pageLinks(nil, page)

		c.Lock()
		c.visited[url] = title
		c.Unlock()

		results <- crawlResult{
			err: nil,
			msg: fmt.Sprintf("%s -> %s\n", url, title),
		}

		for link := range links {
			if c.checkVisited(link) {
				continue
			}
			go c.run(ctx, link, results, depth+1)
		}
	}
}

func (c *crawler) checkVisited(url string) bool {
	c.Lock()
	defer c.Unlock()

	_, ok := c.visited[url]
	return ok
}
