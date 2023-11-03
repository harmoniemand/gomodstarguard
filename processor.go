package gomodstarguard

import (
	"fmt"
	"go/parser"
	"go/token"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"golang.org/x/net/html"
)

var (
	logger = log.New(os.Stderr, "", 0)
)

type Configuration struct {
	Warn      int `yaml:"warn"`
	Error     int `yaml:"error"`
	Exeptions []struct {
		Repository string `yaml:"repository"`
		Reason     string `yaml:"reason"`
	} `yaml:"exeptions"`
}

type Processor struct {
	Config *Configuration
}

func NewProcessor(config *Configuration) (*Processor, error) {
	return &Processor{
		Config: config,
	}, nil
}

func (p *Processor) ProcessFiles(filenames []string) (issues []Issue) {
	for _, filename := range filenames {
		data, err := os.ReadFile(filename)
		if err != nil {
			issues = append(issues, Issue{
				FileName:   filename,
				LineNumber: 0,
				Reason:     fmt.Sprintf("unable to read file, file cannot be linted (%s)", err.Error()),
			})

			continue
		}

		issues = append(issues, p.process(filename, data)...)
	}

	return issues
}

func (p *Processor) process(filename string, data []byte) (issues []Issue) {
	fileSet := token.NewFileSet()

	file, err := parser.ParseFile(fileSet, filename, data, parser.ParseComments)
	if err != nil {
		issues = append(issues, Issue{
			FileName:   filename,
			LineNumber: 0,
			Reason:     fmt.Sprintf("invalid syntax, file cannot be linted (%s)", err.Error()),
		})

		return
	}

	imports := file.Imports
	for n := range imports {
		importedPkg := strings.TrimSpace(strings.Trim(imports[n].Path.Value, "\""))

		// To check if importedPkg is a valid URL:
		if p.isValidURL(importedPkg) {
			logger.Println("Imported package: ", importedPkg)

			url := "https://" + importedPkg
			logger.Println("URL: ", url)
			res, err := http.Get(url) //nolint:gosec

			if err != nil {
				logger.Println("Error: ", err)
				issues = append(issues, p.addError(fileSet, imports[n].Pos(), "unable to load stars from URL"))
				continue
			}

			content, err := io.ReadAll(res.Body)
			if err != nil {
				logger.Println("Error: ", err)
				issues = append(issues, p.addError(fileSet, imports[n].Pos(), "unable to load stars from URL"))
				continue
			}

			doc, err := html.Parse(strings.NewReader(string(content)))
			if err != nil {
				logger.Println("Error: ", err)
				issues = append(issues, p.addError(fileSet, imports[n].Pos(), "unable to load stars from URL"))
				continue
			}

			starsElement := p.getElementById(doc, "repo-stars-counter-star")
			logger.Println("stars: ", starsElement.Data)

		}

		// blockReasons := p.isBlockedPackageFromModFile(importedPkg)
		// if blockReasons == nil {
		// 	continue
		// }

		// for _, blockReason := range blockReasons {
		// 	issues = append(issues, p.addError(fileSet, imports[n].Pos(), blockReason))
		// }
	}

	return issues
}

func (p *Processor) addError(fileset *token.FileSet, pos token.Pos, reason string) Issue {
	position := fileset.Position(pos)

	return Issue{
		FileName:   position.Filename,
		LineNumber: position.Line,
		Position:   position,
		Reason:     reason,
	}
}

func (p *Processor) isValidURL(input string) bool {
	return strings.HasPrefix(input, "github.com")
}

func (p *Processor) GetAttribute(n *html.Node, key string) (string, bool) {
	for _, attr := range n.Attr {
		if attr.Key == key {
			return attr.Val, true
		}
	}
	return "", false
}

func (p *Processor) checkId(n *html.Node, id string) bool {
	if n.Type == html.ElementNode {
		s, ok := p.GetAttribute(n, "id")
		if ok && s == id {
			return true
		}
	}
	return false
}

func (p *Processor) traverse(n *html.Node, id string) *html.Node {
	if p.checkId(n, id) {
		return n
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		result := p.traverse(c, id)
		if result != nil {
			return result
		}
	}

	return nil
}

func (p *Processor) getElementById(n *html.Node, id string) *html.Node {
	return p.traverse(n, id)
}
