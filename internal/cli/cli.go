package cli

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-xmlfmt/xmlfmt"
	"github.com/harmoniemand/gomodstarguard"
	"github.com/harmoniemand/gomodstarguard/internal/filesearch"
	"github.com/mitchellh/go-homedir"
	"github.com/phayes/checkstyle"
	"gopkg.in/yaml.v2"
)

const (
	errFindingHomedir    = "unable to find home directory, %w"
	errReadingConfigFile = "could not read config file: %w"
	errParsingConfigFile = "could not parse config file: %w"
)

var (
	configFile           = ".gomodstarguard.yaml"
	logger               = log.New(os.Stderr, "", 0)
	errFindingConfigFile = fmt.Errorf("could not find config file")
)

func Run() int {
	var (
		args           []string
		help           bool
		noTest         bool
		report         string
		reportFile     string
		issuesExitCode int
		cwd, _         = os.Getwd()
	)

	flag.BoolVar(&help, "h", false, "Show this help text")
	flag.BoolVar(&help, "help", false, "")
	flag.BoolVar(&noTest, "n", false, "Don't lint test files")
	flag.BoolVar(&noTest, "no-test", false, "")
	flag.StringVar(&report, "r", "", "Report results to one of the following formats: checkstyle. "+
		"A report file destination must also be specified")
	flag.StringVar(&report, "report", "", "")
	flag.StringVar(&reportFile, "f", "", "Report results to the specified file. A report type must also be specified")
	flag.StringVar(&reportFile, "file", "", "")
	flag.IntVar(&issuesExitCode, "i", 2, "Exit code when issues were found")
	flag.IntVar(&issuesExitCode, "issues-exit-code", 2, "")
	flag.Parse()

	report = strings.TrimSpace(strings.ToLower(report))

	if help {
		showHelp()
		return 0
	}

	if report != "" && report != "checkstyle" {
		logger.Fatalf("error: invalid report type '%s'", report)
	}

	if report != "" && reportFile == "" {
		logger.Fatalf("error: a report file must be specified when a report is enabled")
	}

	if report == "" && reportFile != "" {
		logger.Fatalf("error: a report type must be specified when a report file is enabled")
	}

	args = flag.Args()
	if len(args) == 0 {
		args = []string{"./..."}
	}

	config, err := GetConfig(configFile)
	if err != nil {
		logger.Fatalf("error: %s", err)
	}

	filteredFiles := filesearch.Find(cwd, noTest, args)

	results := runProcessor(config, filteredFiles)

	return reportIssues(results, report, reportFile, issuesExitCode)
}

func reportIssues(results []gomodstarguard.Issue, report string, reportFile string, issuesExitCode int) int {
	logger.Printf("info: found %d issues", len(results))
	logger.Println()

	if report == "checkstyle" {
		err := WriteCheckstyle(reportFile, results)
		if err != nil {
			logger.Fatalf("error: %s", err)
		}
	}

	for _, r := range results {
		fmt.Println(r.String())
	}

	if len(results) > 0 {
		return issuesExitCode
	}

	return 0
}

func runProcessor(config *gomodstarguard.Configuration, filteredFiles []string) []gomodstarguard.Issue {
	stargazer, err := gomodstarguard.NewStargazer(config)
	if err != nil {
		logger.Fatalf("error: %s", err)
	}

	processor, err := gomodstarguard.NewProcessor(config, stargazer)
	if err != nil {
		logger.Fatalf("error: %s", err)
	}

	results := processor.ProcessFiles(filteredFiles)

	return results
}

// GetConfig from YAML file.
func GetConfig(configFile string) (*gomodstarguard.Configuration, error) {
	config := gomodstarguard.Configuration{}

	home, err := homedir.Dir()
	if err != nil {
		return nil, fmt.Errorf(errFindingHomedir, err)
	}

	cfgFile := ""
	homeDirCfgFile := filepath.Join(home, configFile)

	switch {
	case fileExists(configFile):
		cfgFile = configFile
	case fileExists(homeDirCfgFile):
		cfgFile = homeDirCfgFile
	default:
		return nil, fmt.Errorf("%w: %s %s", errFindingConfigFile, configFile, homeDirCfgFile)
	}

	data, err := os.ReadFile(cfgFile)
	if err != nil {
		return nil, fmt.Errorf(errReadingConfigFile, err)
	}

	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf(errParsingConfigFile, err)
	}

	return &config, nil
}

// showHelp text for command line.
func showHelp() {
	helpText := `Usage: gomodguard <file> [files...]
Also supports package syntax but will use it in relative path, i.e. ./pkg/...
Flags:`
	fmt.Println(helpText)
	flag.PrintDefaults()
}

// WriteCheckstyle takes the results and writes them to a checkstyle formated file.
func WriteCheckstyle(checkstyleFilePath string, results []gomodstarguard.Issue) error {
	check := checkstyle.New()

	for i := range results {
		file := check.EnsureFile(results[i].FileName)
		file.AddError(checkstyle.NewError(results[i].LineNumber, 1, checkstyle.SeverityError, results[i].Reason,
			"gomodguard"))
	}

	checkstyleXML := fmt.Sprintf("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n%s", check.String())

	err := os.WriteFile(checkstyleFilePath, []byte(xmlfmt.FormatXML(checkstyleXML, "", "  ")), 0644) //nolint:gosec
	if err != nil {
		return err
	}

	return nil
}

// fileExists returns true if the file path provided exists.
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}

	return !info.IsDir()
}
