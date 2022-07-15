package main

import (
	//"github.com/jmcvetta/neoism"
	"os"
	"net/http"
	"golang.org/x/net/html"
	"strings"
	"fmt"
	"log"
	"io/ioutil"
	"errors"
	"bytes"
	"strconv"
	"encoding/json"
	"github.com/jmcvetta/neoism"
	"bufio"
	"time"
)

const CALAIS_KEY = "tK8k9MKKGcRzpfwfKqZA7Ye039daJqWQ"
const GOOGLE_KEY = ""

// useful types
type Article struct {
	URL string
	Title string
	Body string
}

var extractors = map[string]func(string) (string, string){
					"nytimes.com": nytimes_extract,
					"washingtonpost.com": wapo_extract,
					"ap.org": aporg_extract,
					"apnews.com": apnews_extract,
					"themoscowtimes.com": moscowtimes_extract,
					"usatoday.com": usatoday_extract,
					"www.cnn.com": cnn_extract,
					"money.cnn.com": cnnmoney_extract,
					"cbsnews.com": cbsnews_extract,
					"politico.eu": politicoeu_extract,
					"politico.com": politico_extract,
					"reuters.com": reuters_extract,
					"thehill.com": thehill_extract,
					"theguardian.com": theguardian_extract,
					"newsweek.com": newsweek_extract,
					"npr.org": npr_extract,
					"forbes.com": forbes_extract,
					"telegraph.co.uk": telegraph_extract,
					"slate.com": slate_extract,
					"independent.co.uk": independent_extract,
					"huffingtonpost.com": huffpo_extract,
					"motherjones.com": motherjones_extract,
					"mcclatchydc.com": mcclatchy_extract,
					"japantimes.co.jp": japantimes_extract,
					"mirror.co.uk": mirror_extract,
					"nbcnews.com": nbcnews_extract,
					"businessinsider.com": businessinsider_extract,
					"talkingpointsmemo.com": tpm_extract,
					"abcnews.go.com": abcnews_extract,
					"foxnews.com": foxnews_extract,
					"thedailybeast.com": dailybeast_extract,
					"russiahouse.org": russiahouse_extract,
					}

func main() {

	file, err := os.Open("../data/links_djtwatchdog.txt")
	if err != nil {
		fmt.Println("FATAL: Cannot load links file")
	} else {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			process(scanner.Text(), "russiahouse.org")
		}

		if err := scanner.Err(); err != nil {
			log.Fatal(err)
		}
	}
}

func process(url string, only string) {

	// URL from Args
/*	url = strings.Split(url, "?")[0]
	url = strings.Split(url, "&sa=")[0]
	url = strings.Split(url, "%23")[0]

	// probably want to unescape but not yet
	url = strings.Replace(url, "%3d", "=", -1)

	url = strings.Replace(url, "https://", "http://", 0)
*/
	if only!="" && !strings.Contains(url, only) {
		return
	}

	client := http_client()

	/*
	res1 := []struct {
		Path   string 		`json:"p"` 		// `json` tag matches column name in query
	}{}

	cq0 := neoism.CypherQuery{
		Statement: "MATCH (f:Person {name: {fromname}}), (t:Person {name: {toname}}), p = shortestPath((f)<-[*]->(t)) RETURN p",
		Parameters: neoism.Props{"fromname": "Felix Sater", "toname": "Tamir Sapir"},
		Result:     &res1,
	}

	db.Cypher(&cq0)

	fmt.Println(len(res1))

	for i := 0; i < len(res1); i++ {
		fmt.Println(res1[i].Path)
	}
	*/

	//fmt.Println("Roctobot Acessing: ", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatalln("FATAL: ", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; U; Linux i686; en-US; rv:1.9.0.1) Gecko/2008071615 Fedora/3.0.1-1.fc9 Firefox/3.0.1")
	resp, err := client.Do(req)

	if err != nil {
		fmt.Println("ERROR: ", err)
	} else {

		defer resp.Body.Close()

		if resp.StatusCode!=200 {
			fmt.Println("FATAL: Status Code: ", resp.StatusCode)
			return
		}

		body_bytes, err2 := ioutil.ReadAll(resp.Body)
		body_string := string(body_bytes[:])

		//fmt.Println(body_string);

		if err2 != nil {
			fmt.Println("ERROR: ", err2)
		} else {
			var body_html string = ""
			var title string = ""

			for k, _ := range extractors {
				if strings.Contains(url, k) {
					//fmt.Println(k)
					body_html, title = extractors[k](body_string)
				}
			}

			article := Article{
				URL: url,
				Title: title,
				Body: body_html,
			}

			// route the result to OpenCalais
			opencalais_extract(article)
		}
	}

}

func base_extract (article_tag string, article_attr string, article_attr_value string, article_filter_attr_value string, headline_attr string, headline_attr_value string, body string) (string, string) {
	var return_string string = ""
	var title string = ""

	doc, err := html.Parse(strings.NewReader(body))

	if err != nil {
		fmt.Println("ERROR: Parsing error: ", err)
	} else {
		var exact = strings.Contains(article_attr, "!")

		if exact {
			article_attr = strings.Replace(article_attr, "!", "", -1)
		}

		var f func(*html.Node)
		f = func(n *html.Node) {
			if n.Type == html.ElementNode  {
				var valid_headline = false

				if strings.Contains(headline_attr, ">") { // shortcut to raw
					//fmt.Println(headline_attr, n.Data)
					if headline_attr==">" + n.Data {
						valid_headline = true
					}
				}

				for _, a := range n.Attr {
					if a.Key == article_attr && n.Data == article_tag {
						if (!exact && strings.Contains(a.Val, article_attr_value)) || (exact && a.Val==article_attr_value) {
							//fmt.Println("Found Match: ", exact, "/", a.Val, "/", article_attr_value)
							if article_filter_attr_value == "" || (article_filter_attr_value != "" && !strings.Contains(a.Val, article_filter_attr_value)) {
								buf := bytes.NewBufferString("")
								html.Render(buf, n)
								return_string += string(buf.Bytes()[:])
								//fmt.Println("Did NOT filter out: ", string(buf.Bytes()[:]))
								//fmt.Println("found unfiltered match")
							} else {
								//fmt.Println("DEBUG: Filtering out due to filter_attr_value match")
							}
						}
					} else {
						if a.Key == headline_attr && strings.Contains(a.Val, headline_attr_value) {
							valid_headline = true
						}
					}
				}

				if valid_headline && title == "" {
					defer func() {
						if r := recover(); r != nil {
							//fmt.Println("UNABLE TO PARSE HEADLINE", r)
							title = "Unknown"
						}
					}()

					buf := bytes.NewBufferString("")
					html.Render(buf, n.FirstChild)
					title = html.UnescapeString(string(buf.Bytes()[:]))

					if strings.TrimSpace(title) == "" {
						title = ""
					}
				}
			}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				f(c)
			}
		}
		f(doc)
	}
	//fmt.Println("HEADLINE: ", title)
	return return_string, title
}

func opencalais_extract (article Article)  {
	//fmt.Println("Article body: " + article.Body)
	calais_url := "https://api.thomsonreuters.com/permid/calais"
	client := http_client()
	req, err := http.NewRequest("POST", calais_url, bytes.NewBufferString(article.Body))

	req.Header.Add("Content-Type", "text/html")
	req.Header.Add("x-ag-access-token", CALAIS_KEY)
	req.Header.Add("omitOutputtingOriginalText", "true")
	req.Header.Add("outputFormat", "application/json")
	req.Header.Add("x-calais-contentClass", "news")
	req.Header.Add("x-calais-language", "English")
	req.Header.Add("Content-Length", strconv.Itoa(len(article.Body)))

	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; U; Linux i686; en-US; rv:1.9.0.1) Gecko/2008071615 Fedora/3.0.1-1.fc9 Firefox/3.0.1")
	resp, err := client.Do(req)
	//fmt.Println("accessing opencalais")
	if err != nil {
		fmt.Println("ERROR1: ", err)
	} else {

		defer resp.Body.Close()
		body_bytes, err2 := ioutil.ReadAll(resp.Body)

		if (err2!=nil) {
			fmt.Println("ERROR: ", err2)
		} else {
			body_response := string(body_bytes[:])
			m := map[string]interface{}{}
			err := json.Unmarshal([]byte(body_response), &m)

			if (err!=nil) {
				fmt.Println("ERROR: invalid parsing effort ", err )
				fmt.Println(body_response)
			} else {
				//fmt.Println("no error but ... ", body_response)
				//fmt.Println("no error from opencalais")
				curiosity_map := map[string]interface{}{}
				for k, v := range m {
					//fmt.Println("key[%s]", k)
					vd := v.(map[string]interface{})
					for _, b := range [...]string{"Company", "Person"} {
						if b == vd["_type"] {
							curiosity_map[k] = vd
						}
					}
				}

				neo4j_injest_curiosity_map(article, curiosity_map)
			}
		}
	}
	// to avoid rate limit issues with OpenCalais
	time.Sleep(2000 * time.Millisecond)
}

func neo4j_injest_curiosity_map (article Article, curiosity_map map[string]interface{}) {
	//fmt.Println("injesting semantic map")

	db, _ := neoism.Connect("http://neo4j:youwontknowwhototrust@localhost:7474/db/data")

	//db.Session.Log = true

	cq0 := neoism.CypherQuery{
		Statement: 	"MERGE (m:Article {url:{url}, title:{title}}) ",
		// Use parameters instead of constructing a query string
		Parameters: neoism.Props{"url": article.URL, "title": article.Title},
		Result:     nil,
	}
	db.Cypher(&cq0)

	// add all people, places and things
	BlackListLoop:
	for ock, v := range curiosity_map {

		vd := v.(map[string]interface{})
		var vdname = strings.ToUpper(vd["name"].(string))
		var vdtype = vd["_type"].(string)

		// Check THE_BLACKLIST

		for _, v := range THE_BLACKLIST {
			if v == vdname {
				continue BlackListLoop
			}
		}

		// Check THE_ALIASES
		if val, ok := THE_ALIASES[vdname]; ok {
			vdname = val;
		}

		if vdtype=="Person" {
			// remove initials if any
			var name = strings.Split(vdname, " ")

			// remove pesky initials
			if len(name) == 3 && strings.Contains(name[1], ".") {
				vdname = name[0] + " " + name[2]
			}
		}

		// Remove punctuation
		vdname = strings.Replace(vdname, ".", "", -1)
		vdname = strings.Replace(vdname, ",", "", -1)

		fmt.Println(vdname + "," + vdtype +  "+" + article.URL)

		cq1 := neoism.CypherQuery{
			Statement: 	"MERGE (m:" + vdtype + " {key:{key}, name:{name}})",
			// Use parameters instead of constructing a query string
			Parameters: neoism.Props{"key": ock, "name": vdname, "type": vdtype},
			Result:     nil,
		}
		db.Cypher(&cq1)

		cq2 := neoism.CypherQuery{
			Statement: 	"MATCH (m:" + vdtype + " {key:{key}}), (n:Article {url:{url}}) " +
					"MERGE (m)-[:MENTIONED_IN]-(n) " +
					"RETURN m",
			// Use parameters instead of constructing a query string
			Parameters: neoism.Props{"key": ock, "url": article.URL},
			Result:     nil,
		}
		db.Cypher(&cq2)
	}
}

func http_client () http.Client {
	// Resolve URL up to 12 redirects.
	return http.Client{
		CheckRedirect: func() func(req *http.Request, via []*http.Request) error {
			redirects := 0
			return func(req *http.Request, via []*http.Request) error {
				if redirects > 24 {
					return errors.New("stopped after 24 redirects")
				}

				redirects++

				// Collect security cookies along the way
				for i :=0; i<len(req.Response.Cookies()); i++ {
					req.AddCookie(req.Response.Cookies()[i]);
				}
				return nil
			}
		}(),
	}
}

// EXTRACTORS


func nytimes_extract (body string) (string, string) {
	return base_extract ("div", "class", "story-body","story-body-text", "id", "headline", body)
}

func wapo_extract (body string) (string, string) {
	return base_extract ("article", "itemprop", "articleBody","no-op-never-reach", "itemprop", "headline", body)
}

func aporg_extract (body string) (string, string) {
	return base_extract ("div", "class", "entry-content","", "class", "story-heading", body)
}

func apnews_extract (body string) (string, string) {
	return base_extract ("div", "class", "articleBody","", "class", "topTitle", body)
}

func moscowtimes_extract (body string) (string, string) {
	return base_extract ("div", "class!", "emerge","", "class", "title", body)
}

func usatoday_extract (body string) (string, string) {
	return base_extract ("div", "itemprop", "articleBody","", "class", "asset-headline", body)
}

func cnn_extract (body string) (string, string) {
	return base_extract ("div", "class", "zn-body__paragraph","", "class", "pg-headline", body)
}

func cnnmoney_extract (body string) (string, string) {
	return base_extract ("div", "id", "storytext","", "class", "article-title", body)
}

func cbsnews_extract (body string) (string, string) {
	return base_extract ("div", "itemprop", "articleBody","", "itemprop", "headline", body)
}

func politicoeu_extract (body string) (string, string) {
	return base_extract ("div", "class", "story-text","", "class", "ev-magazine-layout-title", body)
}

func politico_extract (body string) (string, string) {
	return base_extract ("div", "class", "story-text","", "class", " ", body)
}

func reuters_extract (body string) (string, string) {
	return base_extract ("span", "id", "article-text","", "class", "article-headline", body)
}

func thehill_extract (body string) (string, string) {
	return base_extract ("div", "property", "content:encoded","", "class", "title", body)
}

func theguardian_extract (body string) (string, string) {
	return base_extract ("div", "itemprop", "articleBody","", "itemprop", "headline", body)
}

func newsweek_extract (body string) (string, string) {
	return base_extract ("div", "itemprop", "articleBody","", "itemprop", "headline", body)
}

func forbes_extract (body string) (string, string) {
	return base_extract ("div", "itemprop", "articleBody","", "itemprop", "headline", body)
}

func telegraph_extract (body string) (string, string) {
	return base_extract ("div", "class", "article-body-text","", "itemprop", "headline", body)
}

func independent_extract (body string) (string, string) {
	return base_extract ("div", "itemprop", "articleBody","", "itemprop", "headline", body)
}

func slate_extract (body string) (string, string) {
	return base_extract ("div", "class", "parbase section","", "class", "hed", body)
}

func huffpo_extract (body string) (string, string) {
	return base_extract ("div", "class", "entry__text","", "class", "headline__title", body)
}

func motherjones_extract (body string) (string, string) {
	return base_extract ("div", "id", "node-body-top","", "class", "title", body)
}

func mcclatchy_extract (body string) (string, string) {
	return base_extract ("div", "id", "content-body-","", "class", "title", body)
}

func mirror_extract (body string) (string, string) {
	return base_extract ("div", "class", "article-body","", "itemprop", "headline", body)
}

func nbcnews_extract (body string) (string, string) {
	return base_extract ("div", "class", "article-body","", ">h1", "", body)
}

func japantimes_extract (body string) (string, string) {
	return base_extract ("article", "role", "main","", ">h1", "", body)
}

func npr_extract (body string) (string, string) {
	return base_extract ("div", "itemprop", "articleBody","", ">h1", "", body)
}

func businessinsider_extract (body string) (string, string) {
	return base_extract ("div", "class", "post-content","", ">h1", "", body)
}

func tpm_extract (body string) (string, string) {
	return base_extract ("div", "class", "FeatureBody","", ">h1", "", body)
}

func abcnews_extract (body string) (string, string) {
	return base_extract ("div", "class", "article-copy","", "class", "article-header", body)
}

func foxnews_extract (body string) (string, string) {
	return base_extract ("div", "class", "article-text","", ">h1", "", body)
}

func dailybeast_extract (body string) (string, string) {
	return base_extract ("div", "class", "BodyNodes","", "class", "Title", body)
}

func seekgod_extract (body string) (string, string) {
	return base_extract ("table", "width", "711","", "", "", body)
}

func russiahouse_extract (body string) (string, string) {
	return base_extract("div", "id", "gallery", "", ">h1", "", body)
}

// DATA stuff
// Stuff that clearly doesn't belong
var THE_ALIASES = map[string]string {
	"ERIC": "ERIC TRUMP",
	"TRUMP": "DONALD TRUMP",
	"DONALD JR": "DONALD TRUMP JR",
	"IVANKA": "IVANKA TRUMP",
	"CONWAY": "KELLYANNE CONWAY",
	"THE BANK OF CYPRUS": "BANK OF CYPRUS",
	"EXXONMOBIL": "EXXON",
	"EXXON MOBIL": "EXXON",
	"REAGAN": "RONALD REAGAN",
	"PUTIN": "VLADIMIR PUTIN",
	"CENTRAL INTELLIGENCE AGENCY": "CIA",
	"HILLARY RODHAM CLINTON": "HILLARY CLINTON",
	"DONALD TRUMP BREITBART": "DONALD TRUMP",
	"GAZPROMBANK": "GAZPROM",
	"BURT": "RICHARD BURT",
	"MANAFORT": "PAUL MANAFORT",
	"THE TRUMP ORGANIZATION": "TRUMP ORGANIZATION",
	"TRUMP ORGANIZATION & RELATED GROUP": "TRUMP ORGANIZATION",
	"TORSHIN": "ALEXANDER TORSHIN",
	"MELANIA": "MELANIA TRUMP",
	"MELANIJA TRUMP": "MELANIA TRUMP",
	"RICHARD V": "RICHARD VIGUERIE",
}

// Stuff that clearly doesn't belong
var THE_BLACKLIST = [...]string {
	"GETTY",
	"GETTY IMAGES",
	"THE NEW YORK TIMES",
	"THE TIMES",
	"NY TIMES",
	"WASHINGTON POST",
	"TWITTER",
	"LINKEDIN",
	"CNN",
	"THE WASHINGTON POST",
	"FACEBOOK",
	"BUZZFEED",
	"MSNBC",
	"THE ASSOCIATED PRESS",
	"YAHOO",
	"OBSERVER MEDIA GROUP",
	"YOUTUBE",
	"USA TODAY",
	"THE NEW YOURK OBSERVER",
	"OBSERVER MEDIA GROUP",
	"ABC",
	"WALL STREET JOURNAL",
	"FOX NEWS",
	"REUTERS",
	"AP",
	"THOMSON REUTERS",
	"MCCLATCHY",
	"THE GUARDIAN",
	"NBC",
	"NEW YORK TIMES",
	"POLITICO",
	"RUSSIAN",
	"THE FINANCIAL TIMES",
	"TELEGRAPH CORPORATION",
	"REUTERS U.S.",
	"NEW YORK MEDIA",
	"NEWSWEEK",
	"GOP SENATE",
	"RUSSIAN GOVERNMENT",
	"BBC",
	"BLOOMBERG",
	"CNNMONEY",
	"CBS",
	"WASHINGTON TIMES",
	"LOS ANGELES TIMES",
	"NEW YORK POST",
	"REUTERS/CARLOS BARRIA",
	"MOTHER JONES",
	"SKYPE",
}

