package gomodstarguard

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"regexp"
	"strings"
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
	Config    *Configuration
	Stargazer *Stargazer
}

func NewProcessor(config *Configuration, stargazer *Stargazer) (*Processor, error) {
	return &Processor{
		Config:    config,
		Stargazer: stargazer,
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

		return issues
	}

	imports := file.Imports
	for n := range imports {
		importedPkg := strings.TrimSpace(strings.Trim(imports[n].Path.Value, "\""))

		// To check if importedPkg is a valid URL:
		if p.isValidURL(importedPkg) {
			re := regexp.MustCompile(`^github.com/[^/]+/[^/]+`)
			match := re.FindStringSubmatch(importedPkg)

			repoBaseURL := match[0]

			url := "https://" + repoBaseURL
			stars, err := p.Stargazer.GetStars(url)

			if err != nil {
				issues = append(issues, p.addError(fileSet, imports[n].Pos(), err.Error()))
				continue
			}

			// logger.Println("stars: ", stars)

			if stars < p.Config.Error {
				reason := fmt.Sprintf("github stars of %s is less than the error threshold (%d)", importedPkg, p.Config.Error)
				issues = append(issues, p.addError(fileSet, imports[n].Pos(), reason))

				continue
			}

			if stars < p.Config.Warn {
				reason := fmt.Sprintf("github stars of %s is less than the warn threshold (%d)", importedPkg, p.Config.Warn)
				issues = append(issues, p.addError(fileSet, imports[n].Pos(), reason))

				continue
			}
		}
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
