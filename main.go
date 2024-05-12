package main

// Global TODO: change saved PubDateF to be Unix.Milli
// Global TODO: sanitize HTML from description (?)
// 					OK: make errors globally avialable
// Global TODO: save entries to local SQLite database (?)

import (
	"bufio"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"sync"
	"time"
)

const (
	EXISTS    = "exists"
	//ONDELETE  = "ondelete"
	//PERMANENT = "permanent"
)

const (
	ERR_UnreachableVendorIndex = iota
	ERR_UnreachableItemIndex
	ERR_EmptyGlobalStruct
	ERR_EmptyVendorStruct
	ERR_VendorNil
	ERR_EmptyURL
)

var Errors []error = []error{
	errors.New("unreachable vendor index"),
	errors.New("unreachable item index"),
	errors.New("empty global struct"),
	errors.New("empty vendor struct"),
	errors.New("vendor cannot be nil"),
	errors.New("URL cannot be empty"),
}

type LinkStruct struct {
	Index   int               `json:"-"`
	Links   []string          `json:"Links"`
	Names   []string          `json:"Names"`
	Objects []*Vendor       `json:"Objects"`
	Mapping map[int]*Vendor `json:"-"`
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

type Vendor struct {
	UpdatedAt		 int64
	CustomName   string
	NewInSection bool
	ChannelName  string `xml:"channel>title"`
	ChannelLang  string `xml:"channel>language"`
	Items        []Item `xml:"channel>item"`
}

func (l *LinkStruct) Push(name string, url string) error {
	if len(url) == 0 {
		return Errors[ERR_EmptyURL]
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

func UnixMilliToTime (unixmilli string, base int, bitSize int) (time.Time){
	i, err := strconv.ParseInt(unixmilli, base, bitSize)
	if err != nil {
		log.Fatalln("could not parse time: ", err)
	}
	return time.UnixMilli(i)
}

func scrclear () {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", "cls")
	} else {
		cmd = exec.Command("clear")
	}
	cmd.Stdout = os.Stdout
	cmd.Run()
}

func (l *LinkStruct) global_empty() bool {
	return len(l.Mapping) == 0
}

func (l *LinkStruct) vendor_empty(v *Vendor) bool {
	return len(v.Items) == 0
}

func (l *LinkStruct) print_new() {
	if l.global_empty() {
		fmt.Println(Errors[ERR_EmptyGlobalStruct])
		return
	}
	fmt.Println("There are channels that got updated:")
	for _, e := range l.Mapping {
		if e.NewInSection {
			fmt.Print("\nCustom name: ", e.CustomName, ", title: ", e.ChannelName,", ")
			fmt.Println(
				"updated at:",
				UnixMilliToTime(fmt.Sprint(e.UpdatedAt), 10, 64),
			)
		}
	}
}

func (l *LinkStruct) print_all_in_vendor(vendor string) {
	selected, err := l.GetVendorByName(vendor)
	if err == Errors[ERR_UnreachableVendorIndex] {
		fmt.Println("The vendor with name", vendor, "does not exist")
		return
	} else if err == Errors[ERR_EmptyVendorStruct] {
		fmt.Println("The vendor with name", vendor, "got no updates")
		return
	} else if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Print("Showing all new posts from ", selected.CustomName,"\n")
	for index, newPost := range selected.Items {
		fmt.Print("\n")
		fmt.Println("Post number", index+1)
		fmt.Println("Post title       ->", newPost.Title)
		fmt.Println("Post link        ->", newPost.Link)
		fmt.Println("Post description ->", newPost.Description)
		layout := "Mon, 02 Jan 2006 15:04:05 -0700 GMT"
		t, err := time.Parse(layout, newPost.PubDate)
		if err != nil {
			//fmt.Println("!!e!! => Error parsing date:", err)
			fmt.Println("Publication date ->", newPost.PubDate)
		} else {
			newPost.PubDateF = t
			fmt.Println("Publication date ->", newPost.PubDateF)
		}
		if newPost.Enclosure.URL != "" {
			fmt.Println("Enclosure type   ->", newPost.Enclosure.Type)
			fmt.Println("Enclosure length ->", newPost.Enclosure.Length)
			fmt.Println("Enclosure URL    ->", newPost.Enclosure.URL)
		}
	}
}

func (l *LinkStruct) PrintDialogue() {
	for {
		fmt.Print("\nEnter the custom name to view posts / l to list updated vendors / c to clear screen / q to quit: ")
		scanstdin := bufio.NewScanner(os.Stdin)
		if scanstdin.Scan() {
			input := scanstdin.Text()
			if input == "q" {
				return
			}
			if input == "l" {
				l.print_new()
				continue
			}
			if input == "c" {
				scrclear()
				continue
			}
			l.print_all_in_vendor(input)
		}
	}
}

func (l *LinkStruct) GetAllNew() (out []*Vendor, err error) {
	if l.global_empty() {
		return nil, Errors[ERR_EmptyGlobalStruct]
	}
	for _, e := range l.Mapping {
		if e.NewInSection {
			out = append(out, e)
		}
	}
	return out, nil
}

func (l *LinkStruct) GetVendorByName(vendor string) (out *Vendor, err error) {
	if l.global_empty() {
		return nil, Errors[ERR_EmptyGlobalStruct]
	}
	for _, selected := range l.Mapping {
		if vendor == selected.CustomName || vendor == selected.ChannelName {
			if l.vendor_empty(selected) {
				return nil, Errors[ERR_EmptyVendorStruct]
			}
			return selected, nil
		}
	}
	return nil, Errors[ERR_UnreachableVendorIndex]
}

func (l *LinkStruct) GetVendorByIndex(vendor_index int) (out *Vendor, err error) {
	if l.global_empty() {
		return nil, Errors[ERR_EmptyGlobalStruct]
	}
	if vendor_index <= 0 {
		return nil, Errors[ERR_UnreachableVendorIndex]
	}
	found, exist := l.Mapping[vendor_index]
	if exist {
		return found, nil
	}
	return nil, Errors[ERR_UnreachableVendorIndex]
}

func (l *LinkStruct) GetNewSelectedItem(vendor *Vendor, item_index int) (out *Item, err error) {
	if vendor == nil {
		return nil, Errors[ERR_VendorNil]
	}
	if item_index > len(vendor.Items) || item_index < 0 {
		return nil, Errors[ERR_UnreachableItemIndex]
	}
	return &vendor.Items[item_index], nil
}

func main() {

	guidfilename := "guids"
	datafilename := "data"
	jsonformat := ".json"

	timeFormat := time.Now().UnixMicro()
	datafilename += "_" + fmt.Sprint(timeFormat)

	guidlist, err := os.Open(guidfilename + jsonformat)
	if err != nil {
		guidlist, err = os.Create(guidfilename + jsonformat)
		if err != nil {
			log.Fatalln("unable to open and create a json guid list file: ", err)
		}
	}

	glBytes, err := io.ReadAll(guidlist)
	if err != nil {
		log.Fatalln(err)
	}

	guidlist.Close()

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
		Objects: []*Vendor{},
		Mapping: map[int]*Vendor{},
	}

	// will be in API
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

	if len(newLinks.Links) == 0 {
		fmt.Println("To start, add some links. Usage: LinkStruct.Push(name string, url string)")
		return
	}
	performance_test_start := time.Now()
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

			var parsed Vendor
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
				parsed.UpdatedAt = time.Now().UnixMilli()
				parsed.CustomName = newLinks.Names[i]
				newLinks.Objects = append(newLinks.Objects, &parsed)
				newLinks.Mapping[i] = &parsed
				//m.Unlock()
			}

			wg.Done()
		}(i, &wg)
	}

	wg.Wait()
	fmt.Println("\nGetting and parsing xml took", time.Since(performance_test_start))

	// We return if nothing new is found
	if len(newLinks.Mapping) == 0 {
		fmt.Println("\nChecked", len(newLinks.Links), "links. Updated", len(newLinks.Mapping), "of them. No new feeds.")
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

	// User interaction part
	fmt.Println("\nChecked", len(newLinks.Links), "links. Updated", len(newLinks.Mapping), "of them.")
	fmt.Println("\nCheck for newly created file named", datafilename+jsonformat)
	newLinks.PrintDialogue()
}