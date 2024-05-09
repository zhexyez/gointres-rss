package main

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

const (
	EXISTS    = "exists"
	ONDELETE  = "ondelete"
	PERMANENT = "permanent"
)

type LinkStruct struct {
	Index   int               `json:"-"`
	Links   []string          `json:"Links"`
	Names   []string          `json:"Names"`
	Objects []*DelveXML       `json:"Objects"`
	Mapping map[int]*DelveXML `json:"-"`
}

type Enclosure struct {
	URL    string `xml:"url,attr"`
	Length string `xml:"length,attr"`
	Type   string `xml:"type,attr"`
}

type Item struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
	PubDateF    time.Time
	Enclosure   Enclosure `xml:"enclosure"`
	Guid        string    `xml:"guid"`
}

type DelveXML struct {
	CustomName   string
	NewInSection bool
	ChannelName  string `xml:"channel>title"`
	Items        []Item `xml:"channel>item"`
}

func (l *LinkStruct) Push(name string, url string) error {
	if len(url) == 0 {
		return errors.New("url cannot be empty")
	}
	if len(l.Links) != 0 {
		for _, link := range l.Links {
			if link == url {
				return errors.New("url already exist")
			}
		}
		l.Links = append(l.Links, url)
		l.Names = append(l.Names, name)
		l.Index += 1
		return nil
	}
	l.Links = append(l.Links, url)
	l.Names = append(l.Names, name)
	l.Index += 1
	return nil
}

func main() {

	guidfilename := "guids"
	datafilename := "data"
	jsonformat := ".json"

	timeFormat := time.Now().Format("2006-01-02_15-04-05")
	datafilename += "_" + timeFormat

	guidlist, err := os.Open(guidfilename + jsonformat)
	if err != nil {
		guidlist, err = os.Create(guidfilename + jsonformat)
		if err != nil {
			log.Fatalln("unable to open and create a json guid list file: ", err)
		}
	}
	defer guidlist.Close()

	glBytes, err := io.ReadAll(guidlist)
	if err != nil {
		log.Fatalln(err)
	}

	if len(glBytes) == 0 {
		braces := []byte{'{', '}'}
		glBytes = append(glBytes, braces...)
	}

	var jsonReady map[string]string

	err = json.Unmarshal(glBytes, &jsonReady)
	if err != nil {
		log.Fatalln(err)
	}

	newLinks := LinkStruct{
		Index:   0,
		Links:   []string{},
		Names:   []string{},
		Objects: []*DelveXML{},
		Mapping: map[int]*DelveXML{},
	}

	_ = newLinks.Push("Megaphone.fm/New Heights with Jason and Travis Kelce", "https://feeds.megaphone.fm/newheights")
	_ = newLinks.Push("NBCnews.com/Murder in Apartment 12", "https://podcastfeeds.nbcnews.com/RPWEjhKq")
	_ = newLinks.Push("Art19.com/Exposed: Cover-Up at Columbia University", "https://rss.art19.com/-exposed-")
	_ = newLinks.Push("SimpleCast.com/The Daily", "https://feeds.simplecast.com/54nAGcIl")
	_ = newLinks.Push("CNBC.com/US Top News and Analysis", "https://search.cnbc.com/rs/search/combinedcms/view.xml?partnerId=wrss01&id=100003114")
	_ = newLinks.Push("CNBC.com/International: Top News And Analysis", "https://search.cnbc.com/rs/search/combinedcms/view.xml?partnerId=wrss01&id=100727362")
	_ = newLinks.Push("CNBC.com/Economy", "https://search.cnbc.com/rs/search/combinedcms/view.xml?partnerId=wrss01&id=20910258")
	_ = newLinks.Push("CNBC.com/Finance", "https://search.cnbc.com/rs/search/combinedcms/view.xml?partnerId=wrss01&id=10000664")
	_ = newLinks.Push("CNBC.com/Energy", "https://search.cnbc.com/rs/search/combinedcms/view.xml?partnerId=wrss01&id=19836768")
	_ = newLinks.Push("CNBC.com/Investing", "https://search.cnbc.com/rs/search/combinedcms/view.xml?partnerId=wrss01&id=15839069")
	_ = newLinks.Push("CNBC.com/Top Videos", "https://search.cnbc.com/rs/search/combinedcms/view.xml?partnerId=wrss01&id=15839263")
	_ = newLinks.Push("CNBC.com/Futures Now", "https://search.cnbc.com/rs/search/combinedcms/view.xml?partnerId=wrss01&id=48227449")
	_ = newLinks.Push("TechCrunch.com", "https://techcrunch.com/feed/")
	_ = newLinks.Push("Wired.com/Business", "https://www.wired.com/feed/category/business/latest/rss")
	_ = newLinks.Push("Wired.com/Gear", "https://www.wired.com/feed/category/gear/latest/rss")
	_ = newLinks.Push("Wired.com/AI", "https://www.wired.com/feed/tag/ai/latest/rss")
	_ = newLinks.Push("Wired.com/Culture", "https://www.wired.com/feed/category/culture/latest/rss")
	_ = newLinks.Push("Wired.com/Security", "https://www.wired.com/feed/category/security/latest/rss")
	_ = newLinks.Push("Wired.com/Backchannel", "https://www.wired.com/feed/category/backchannel/latest/rss")
	_ = newLinks.Push("Billboard.com/Billboard", "https://www.billboard.com/feed")

	var wg sync.WaitGroup

	// In the created goroutines usage of sync.Mutex is not completely clear.
	// On the one hand, it will cause no data racing, because the order is not
	// need to be preserved.
	// On the other, who knows. I will test it later on.
	//
	//var m  sync.Mutex

	fmt.Println("Links count is:",len(newLinks.Links))
	for i := range len(newLinks.Links) {
		wg.Add(1)
		go func(i int, wg *sync.WaitGroup) {

			fmt.Println("getting feed from ", newLinks.Links[i])

			resp, err := http.Get(newLinks.Links[i])
			if err != nil {
				log.Fatalln(err)
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Fatalln(err)
			}

			var parsed DelveXML
			err = xml.Unmarshal(body, &parsed)
			if err != nil {
				log.Fatalln(err)
			}

			parsed.NewInSection = false

			tmpSlice := []Item{}

			// This is an ugly mess and I don't like it.
			// I need to find a way to check either in parallel,
			// or to rewrite it completely.
			if len(jsonReady) > 0 {
				for _, e := range parsed.Items {
					if _, exist := jsonReady[e.Guid]; exist {
						continue
					} else {
						jsonReady[e.Guid] = EXISTS
						tmpSlice = append(tmpSlice, e)
						parsed.NewInSection = true
					}
				}
			} else {
				for _, e := range parsed.Items {
					jsonReady[e.Guid] = EXISTS
					tmpSlice = append(tmpSlice, e)
					parsed.NewInSection = true
				}
			}

			// Personally, I think I need to rewrite it using
			// pointers. Copying elements require memory and
			// additional CPU cycles.
			if len(tmpSlice) != len(parsed.Items) {
				parsed.Items = tmpSlice
			}

			// We will proceed only if something new was parsed
			if len(parsed.Items) > 0 {
				//m.Lock()
				newLinks.Objects = append(newLinks.Objects, &parsed)
				newLinks.Mapping[i] = &parsed
				//m.Unlock()
			}

			wg.Done()
		}(i, &wg)
	}

	wg.Wait()

	// We return if nothing new is found
	if len(newLinks.Mapping) == 0 {
		fmt.Println("No new feeds")
		return
	}

	jsonToWrite, err := json.Marshal(jsonReady)
	if err != nil {
		log.Fatalln(err)
	}

	err = os.WriteFile(guidfilename+jsonformat, jsonToWrite, 0644)
	if err != nil {
		log.Fatalln(err)
	}

	jsonData, err := json.Marshal(newLinks)
	if err != nil {
		log.Fatalln("Error marshalling to JSON:", err)
	}

	err = os.WriteFile(datafilename+jsonformat, jsonData, 0644)
	if err != nil {
		log.Fatalln(err)
	}

	// This is mess. Need better way to display.
	for link, e := range newLinks.Mapping {
		if e.NewInSection {
			fmt.Print("\n=== SHOWING NEW FEED FROM ", newLinks.Names[link], " ===\n\n")
			fmt.Print("Channel title ==> ", newLinks.Mapping[link].ChannelName, "\n\n")
			for _, i := range newLinks.Mapping[link].Items {
				fmt.Println("<== == == == == == == == == == == == == == == == == == == == == ==>")
				fmt.Println("Guid             ->", i.Guid)
				fmt.Println("Title            ->", i.Title)
				layout := "Mon, 02 Jan 2006 15:04:05 -0700"
				t, err := time.Parse(layout, i.PubDate)
				if err != nil {
					//fmt.Println("!!e!! => Error parsing date:", err)
					fmt.Println("Publication date ->", i.PubDate)
				} else {
					i.PubDateF = t
					fmt.Println("Publication date ->", i.PubDateF)
				}
				fmt.Println("Description      ->", i.Description)
				fmt.Println("Full link        ->", i.Link)
				fmt.Println("Enclosure type   ->", i.Enclosure.Type)
				fmt.Println("Enclosure length ->", i.Enclosure.Length)
				fmt.Println("Enclosure URL    ->", i.Enclosure.URL)
				fmt.Println("<== == == == == == == == == == == == == == == == == == == == == ==>")
			}
		} else {
			fmt.Println("\nNothing new from ", newLinks.Names[link])
			fmt.Print("Channel title ==> ", newLinks.Mapping[link].ChannelName, "\n")
		}
	}
	
	fmt.Println("everything is OK. check for newly created file named", datafilename+jsonformat)
}
