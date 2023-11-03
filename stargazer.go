package gomodstarguard

import (
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/net/html"
)

type GithubQueryError struct {
	Err           error
	RepositoryURL string
}

func (e GithubQueryError) Error() string {
	return "Error on Querying Github for " + e.RepositoryURL + " - " + e.Err.Error()
}

type StarParserError struct {
	Err           error
	RepositoryURL string
}

func (e StarParserError) Error() string {
	return "Error on Parsing Stars for " + e.RepositoryURL + " - " + e.Err.Error()
}

type Stargazer struct {
	Config *Configuration
}

func NewStargazer(config *Configuration) (*Stargazer, error) {
	return &Stargazer{
		Config: config,
	}, nil
}

func (s *Stargazer) GetStars(repositoryURL string) (int, error) {
	content, ghErr := s.loadHTML(repositoryURL)
	if ghErr != nil {
		return 0, &GithubQueryError{
			Err: ghErr,

			RepositoryURL: repositoryURL,
		}
	}

	doc, err := html.Parse(strings.NewReader(content))
	if err != nil {
		return 0, &GithubQueryError{
			Err:           err,
			RepositoryURL: repositoryURL,
		}
	}

	starsElement := s.getElementByID(doc, "repo-stars-counter-star")
	if starsElement == nil {
		return 0, &StarParserError{
			Err:           errors.New("could not find stars element in fetched HTML"),
			RepositoryURL: repositoryURL,
		}
	}

	stars, parseErr := s.parseStarNumber(starsElement.FirstChild.Data)
	if parseErr != nil {
		return 0, &StarParserError{
			Err:           parseErr,
			RepositoryURL: repositoryURL,
		}
	}

	return stars, nil
}

func (s *Stargazer) parseStarNumber(stars string) (int, error) {
	var multiplier float64 = 1

	if stars[len(stars)-1:] == "k" {
		multiplier = 1000
	}

	starNumberStr := strings.ReplaceAll(stars, "k", "")

	starsFloat, err := strconv.ParseFloat(starNumberStr, 32)
	if err != nil {
		return 0, &StarParserError{
			Err:           err,
			RepositoryURL: "",
		}
	}

	return int(starsFloat * multiplier), nil
}

func (s *Stargazer) loadHTML(url string) (string, *GithubQueryError) {
	res, err := http.Get(url) //nolint:gosec // TODO: add context

	if err != nil {
		return "", &GithubQueryError{
			Err:           err,
			RepositoryURL: url,
		}
	}

	content, err := io.ReadAll(res.Body)
	if err != nil {
		return "", &GithubQueryError{
			Err:           err,
			RepositoryURL: url,
		}
	}

	err = res.Body.Close()
	if err != nil {
		return "", &GithubQueryError{
			Err:           err,
			RepositoryURL: url,
		}
	}

	return string(content), nil
}

func (s *Stargazer) getAttribute(n *html.Node, key string) (string, bool) {
	for _, attr := range n.Attr {
		if attr.Key == key {
			return attr.Val, true
		}
	}

	return "", false
}

func (s *Stargazer) checkID(n *html.Node, id string) bool {
	if n.Type == html.ElementNode {
		s, ok := s.getAttribute(n, "id")
		if ok && s == id {
			return true
		}
	}

	return false
}

func (s *Stargazer) traverse(n *html.Node, id string) *html.Node {
	if s.checkID(n, id) {
		return n
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		result := s.traverse(c, id)
		if result != nil {
			return result
		}
	}

	return nil
}

func (s *Stargazer) getElementByID(n *html.Node, id string) *html.Node {
	return s.traverse(n, id)
}
