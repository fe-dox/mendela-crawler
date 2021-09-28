package app

import (
	"github.com/gin-gonic/gin"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
)

type App struct {
	log log.Logger
}

type CrawlOptions struct {
	Url   string `json:"url"`
	Depth uint   `json:"depth"`
}

type CrawlResult struct {
	SiteName string   `json:"siteName"`
	Emails   []string `json:"emails"`
}

func (a *App) Start() {
	router := gin.Default()
	router.OPTIONS("/crawl", func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")

		c.AbortWithStatus(204)
	})
	router.POST("/crawl", func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		var options CrawlOptions
		err := c.ShouldBindJSON(&options)
		if err != nil {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		resultsChannel := make(chan CrawlResult)
		wg := &sync.WaitGroup{}
		wg.Add(1)
		go func() {
			wg.Wait()
			close(resultsChannel)
		}()
		go Crawl(options, resultsChannel, wg)
		var results []CrawlResult
		for result := range resultsChannel {
			results = append(results, result)
		}

		c.JSON(200, map[string]interface{}{"data": results})
	})
	err := router.Run(":3000")
	if err != nil {
		log.Fatal(err)
	}
}

func Crawl(options CrawlOptions, c chan<- CrawlResult, wg *sync.WaitGroup) {
	mailRegexp := regexp.MustCompile(`(?m)[a-zA-Z0-9~_\-+]+@([a-zA-Z\-]+\.?)+`)
	res, err := http.Get(options.Url)
	if err != nil {
		wg.Done()
		return
	}

	if res.StatusCode != http.StatusOK {
		wg.Done()
		return
	}
	byteBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		wg.Done()
		return
	}
	body := string(byteBody)
	result := CrawlResult{SiteName: options.Url}
	result.Emails = mailRegexp.FindAllString(body, -1)
	c <- result
	if options.Depth > 0 {
		urlRegexp := regexp.MustCompile(`(?m)href="https?:\/\/(www\.)?[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b([-a-zA-Z0-9()@:%_\+.~#?&//=]*)"`)
		result := urlRegexp.FindAllString(body, -1)

		i := 0
		for _, r := range result {
			if !strings.HasSuffix(r, ".pdf\"") && !strings.HasSuffix(r, ".png\"") && !strings.HasSuffix(r, ".css\"") && !strings.HasSuffix(r, ".jpg\"") && !strings.HasSuffix(r, ".ico\"") {
				result[i] = r
				i++
			}
		}
		result = result[:i]
		if len(result) != 0 {
			end := 9
			if len(result) < 10 {
				end = len(result) - 1
			}
			for _, url := range result[:end] {
				url = url[6 : len(url)-1]
				wg.Add(1)
				go Crawl(CrawlOptions{Url: url, Depth: options.Depth - 1}, c, wg)
			}
		}
	}
	wg.Done()

}
