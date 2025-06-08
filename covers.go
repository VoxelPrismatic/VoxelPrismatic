package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"strings"

	"golang.org/x/net/html"
)

const COLUMNS = 5

type Album struct {
	Cover  string
	Artist string
	Name   string
	Url    string
}

func (a Album) String() string {
	if a.Url == "" {
		return "<td></td>"
	}
	return fmt.Sprintf(`
<td>
	<a href="%s" target="_blank">
		<img src="%s" width="128px"/><br/>
		<sup>%s</sup>
	</a>
</td>
`, a.Url, a.Cover, a.Name)
}

func main() {
	albums := map[string]Album{}
	f, err := os.ReadFile("./covers.json")
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(f, &albums)
	if err != nil {
		panic(err)
	}

	flag.Func("add", "add url", func(url string) error {
		if _, ok := albums[url]; ok {
			return nil
		}
		album, err := fetchBandcamp(url)
		if err != nil {
			return err
		}
		albums[album.Url] = album
		return nil
	})

	flag.Func("del", "remove url", func(url string) error {
		if _, ok := albums[url]; !ok {
			album, err := fetchBandcamp(url)
			if err != nil {
				return err
			}
			url = album.Url
		}
		delete(albums, url)
		return nil
	})

	flag.Parse()

	for k, v := range albums {
		if v.Url == "" {
			delete(albums, k)
		}
	}

	out, err := json.Marshal(&albums)
	if err != nil {
		panic(err)
	}

	err = os.WriteFile("./covers.json", out, 0o755)
	if err != nil {
		panic(err)
	}

	toList := make([]Album, len(albums))
	i := 0
	for _, v := range albums {
		toList[i] = v
		i++
	}

	rand.Shuffle(i-1, func(i, j int) {
		toList[i], toList[j] = toList[j], toList[i]
	})

	count := len(toList) / COLUMNS
	excess := len(toList) % COLUMNS
	if excess != 0 {
		count++
	}
	rows := make([][COLUMNS]Album, count)
	for i, v := range toList {
		rows[i/COLUMNS][i%COLUMNS] = v
	}

	last := make([]Album, COLUMNS)
	for i, v := range rows[count-1] {
		last[i] = v
	}
	fmt.Println(excess)
	for range (COLUMNS - excess) / 2 {
		pop := last[COLUMNS-1]
		last = slices.Concat([]Album{pop}, last[:COLUMNS-1])
	}
	for i, v := range last {
		rows[count-1][i] = v
	}

	fmt.Println("<table>")
	for _, row := range rows {
		fmt.Println("<tr>")
		for _, a := range row {
			fmt.Println(a)
		}
		fmt.Println("</tr>")
	}

	fmt.Println("</table>")
}

var parens = regexp.MustCompile(`\(.*?\)`)

func fetchBandcamp(url string) (Album, error) {
	ret := Album{}
	cmd := exec.Command("curl", "-s", url)
	stdout, err := cmd.CombinedOutput()
	if err != nil {
		return ret, err
	}

	doc, err := html.Parse(bytes.NewBuffer(stdout))
	if err != nil {
		return ret, err
	}

	for node := range doc.Descendants() {
		attr := attrs(node)
		if node.Data != "meta" {
			continue
		}
		switch attr["property"] {
		case "og:title":
			parts := strings.SplitN(attr["content"], ", by", 2)
			ret.Name = strings.TrimSpace(parens.ReplaceAllString(parts[0], ""))
			ret.Artist = strings.TrimSpace(parts[1])
		case "og:url":
			ret.Url = attr["content"]
		case "og:image":
			ret.Cover = attr["content"]
		}
	}

	return ret, nil
}

func attrs(node *html.Node) map[string]string {
	ret := map[string]string{}
	for _, attr := range node.Attr {
		ret[attr.Key] = attr.Val
	}
	return ret
}
