package hlsdl

import (
	"bufio"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/grafov/m3u8"
)

func parseHlsSegments(hlsURL string, headers map[string]string, f func(string) (string, error)) ([]*Segment, error) {
	baseURL, err := url.Parse(hlsURL)
	if err != nil {
		return nil, errors.New("Invalid m3u8 url")
	}

	p, t, err := getM3u8ListType(hlsURL, headers)
	if err != nil {
		return nil, err
	}

	// If url is of type master, obtain the media playlist
	if t == m3u8.MASTER {
		masterpl := p.(*m3u8.MasterPlaylist)

		variant, err := showOptionsMasterFormat(masterpl)
		if err != nil {
			return nil, err
		}
		newURL, err := getURLMediaFormatBase(hlsURL, f)
		if err != nil {
			return nil, err
		}

		urlFinal := newURL + variant.URI
		return parseHlsSegments(urlFinal, headers, f)
	}

	if t != m3u8.MEDIA {
		return nil, errors.New("No support the m3u8 format")
	}

	mediaList := p.(*m3u8.MediaPlaylist)
	segments := []*Segment{}
	for _, seg := range mediaList.Segments {
		if seg == nil {
			continue
		}

		if !strings.Contains(seg.URI, "http") {
			segmentURL, err := baseURL.Parse(seg.URI)
			if err != nil {
				return nil, err
			}

			seg.URI = segmentURL.String()
		}

		if seg.Key == nil && mediaList.Key != nil {
			seg.Key = mediaList.Key
		}

		if seg.Key != nil && !strings.Contains(seg.Key.URI, "http") {
			keyURL, err := baseURL.Parse(seg.Key.URI)
			if err != nil {
				return nil, err
			}

			seg.Key.URI = keyURL.String()
		}

		segment := &Segment{MediaSegment: seg}
		segments = append(segments, segment)
	}

	return segments, nil
}

func newRequest(url string, headers map[string]string) (*http.Request, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	return req, nil
}

func getM3u8ListType(url string, headers map[string]string) (m3u8.Playlist, m3u8.ListType, error) {

	req, err := newRequest(url, headers)
	if err != nil {
		return nil, 0, err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, 0, errors.New(res.Status)
	}

	p, t, err := m3u8.DecodeFrom(res.Body, false)
	if err != nil {
		return nil, 0, err
	}

	return p, t, nil
}

func showOptionsMasterFormat(masterpl *m3u8.MasterPlaylist) (*m3u8.Variant, error) {
	fmt.Println("Choose an option resolution:")
	for i, option := range masterpl.Variants {
		if option.Resolution != "" {
			fmt.Printf("%d. %s\n", i+1, option.Resolution)
		}
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter option number: ")
	input, _ := reader.ReadString('\n')

	input = strings.TrimSpace(input)
	optionNum, err := strconv.Atoi(input)
	if err != nil {
		fmt.Println("Invalid input:", input)
		return nil, err
	}

	if optionNum < 1 || optionNum > len(masterpl.Variants) {
		fmt.Println("Invalid option number:", optionNum)
		return nil, err
	}

	return masterpl.Variants[optionNum-1], nil
}

func getURLMediaFormatBase(hlsURL string, f func(string) (string, error)) (string, error) {
	if f != nil {
		return f(hlsURL)
	}

	u, err := url.Parse(hlsURL)
	if err != nil {
		fmt.Println("Error parsing URL:", err)
		return "", err
	}

	dir := path.Dir(u.Path)

	_, file := path.Split(dir)
	if idx := strings.Index(file, "?"); idx != -1 {
		file = file[:idx]
		dir = path.Join(path.Dir(dir), file)
	}

	u.Path = dir
	u.RawQuery = ""
	newURL := u.String()
	return newURL, nil
}
