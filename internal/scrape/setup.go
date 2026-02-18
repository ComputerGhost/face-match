package scrape

import (
	"fmt"
	"time"

	"github.com/gocolly/colly"
)

func SetupBackoff(c *colly.Collector, delay time.Duration) {
	c.OnError(func(r *colly.Response, e error) {
		if e.Error() != "Too Many Requests" {
			return
		}
		fmt.Println("Too many requests. Backing off...")
		time.Sleep(delay)
		r.Request.Retry()
	})
}

func SetupDelay(c *colly.Collector, delay time.Duration) {
	_ = c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Delay:       delay,
		RandomDelay: 500 * time.Millisecond,
	})
}

func SetupErrorLogging(c *colly.Collector) {
	c.OnError(func(r *colly.Response, err error) {
		fmt.Println("Error visiting", r.Request.URL, err)
	})
}

func SetupRequestLogging(c *colly.Collector) {
	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting", r.URL)
	})
}
