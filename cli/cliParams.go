package cli

import (
	"flag"
	"log"
	"os"
	"regexp"
	"strings"
)

type Params struct {
	SearchStrings [][]byte
	searchStrings ArrayFlags
	regexpStrings ArrayFlags
	RegexpStrings []*regexp.Regexp
	Force         bool
	Line          bool
	Inverse       bool
	X             bool
	Mask          string
	InputDir      string
	OutputDir     string
	Delimiter     []byte
}

func (it *Params) Load() {
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("working dir: %v", dir)

	var searchString ArrayFlags
	flag.Var(&searchString, "s", "String to search")

	var regexString ArrayFlags
	flag.Var(&regexString, "r", "Regexp to search")

	var force bool
	flag.BoolVar(&force, "f", false, "Force directory recreation")

	var line bool
	flag.BoolVar(&line, "l", false, "Use line-by-line parser")

	var dry bool
	flag.BoolVar(&dry, "x", false, "Skip write results")

	var inverse bool
	flag.BoolVar(&inverse, "i", false, "Inverse search")

	var inputDir string
	flag.StringVar(&inputDir, "src", dir, "Input directory")

	var outputDir string
	flag.StringVar(&outputDir, "out", "", "Output directory")

	var fileMask string
	flag.StringVar(&fileMask, "m", "*", "File mask to filter input files")

	var delimiter string
	flag.StringVar(&delimiter, "d", "####", "Log records delimiter")

	flag.Parse()

	it.searchStrings = searchString
	it.SearchStrings = make([][]byte, len(searchString))
	for i, s := range searchString {
		it.SearchStrings[i] = []byte(s)
	}
	it.regexpStrings = regexString
	it.RegexpStrings = make([]*regexp.Regexp, len(regexString))
	for i, s := range regexString {
		it.RegexpStrings[i] = regexp.MustCompile(s)
	}
	it.Force = force
	it.Line = line
	it.Inverse = inverse
	it.Mask = fileMask
	it.X = dry
	it.InputDir = inputDir
	if len(outputDir) > 0 {
		it.OutputDir = outputDir
	} else {
		it.OutputDir = it.resultDir()
	}
	it.Delimiter = []byte(delimiter)
}

func (it *Params) resultDir() string {
	slice := append([]string{}, it.searchStrings...)
	slice = append(slice, it.regexpStrings...)
	pattern := regexp.MustCompile(`[/\\\s:*?"<>|]+`)
	return pattern.ReplaceAllString(strings.Join(slice, "_"), "_")
}

func (it *Params) Print() {
	log.Printf("========================================")
	log.Printf("Start params")
	log.Printf("strings: %v", it.searchStrings.String())
	log.Printf("regexps: %v", it.regexpStrings.String())
	log.Printf("force: %v", it.Force)
	log.Printf("inverse: %v", it.Inverse)
	log.Printf("dry-run: %v", it.X)
	log.Printf("line: %v", it.Line)
	log.Printf("mask: %v", it.Mask)
	log.Printf("input dir: %v", it.InputDir)
	log.Printf("output dir: %v", it.OutputDir)
	log.Printf("delimiter: %v", string(it.Delimiter))
	log.Printf("========================================")
}
