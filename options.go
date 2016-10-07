// sift
// Copyright (C) 2014-2016 Sven Taute
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, version 3 of the License.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/crypto/ssh/terminal"
)

type Options struct {
	BinarySkip          bool   `long:"binary-skip" description:"skip files that seem to be binary"`
	BinaryAsText        bool   `short:"a" long:"binary-text" description:"process files that seem to be binary as text"`
	Blocksize           string `long:"blocksize" description:"blocksize in bytes (with optional suffix K|M)"`
	Color               string
	ColorFunc           func()   `long:"color" description:"enable colored output (default: auto)" json:"-"`
	NoColorFunc         func()   `long:"no-color" description:"disable colored output" json:"-"`
	ConfigFile          string   `long:"conf" description:"load config file FILE" value-name:"FILE" json:"-"`
	Context             int      `short:"C" long:"context" description:"show NUM context lines" value-name:"NUM" json:"-"`
	ContextAfter        int      `short:"A" long:"context-after" description:"show NUM context lines after match" value-name:"NUM" json:"-"`
	ContextBefore       int      `short:"B" long:"context-before" description:"show NUM context lines before match" value-name:"NUM" json:"-"`
	Cores               int      `short:"j" long:"cores" description:"limit used CPU Cores (default: 0 = all)" default-mask:"-"`
	Count               bool     `short:"c" long:"count" description:"print count of matches per file" json:"-"`
	IncludeDirs         []string `long:"dirs" description:"recurse only into directories whose name matches GLOB" value-name:"GLOB" default-mask:"-"`
	ErrShowLineLength   bool     `long:"err-show-line-length" description:"show all line length errors"`
	ErrSkipLineLength   bool     `long:"err-skip-line-length" description:"skip line length errors"`
	ExcludeDirs         []string `long:"exclude-dirs" description:"do not recurse into directories whose name matches GLOB" value-name:"GLOB" default-mask:"-"`
	IncludeExtensions   string   `short:"x" long:"ext" description:"limit search to specific file extensions (comma-separated)" default-mask:"-"`
	ExcludeExtensions   string   `short:"X" long:"exclude-ext" description:"exclude specific file extensions (comma-separated)" default-mask:"-"`
	IncludeFiles        []string `long:"files" description:"search only files whose name matches GLOB" value-name:"GLOB" default-mask:"-"`
	ExcludeFiles        []string `long:"exclude-files" description:"do not select files whose name matches GLOB while recursing" value-name:"GLOB" default-mask:"-"`
	IncludePath         string   `long:"path" description:"search only files whose path matches PATTERN" value-name:"PATTERN" default-mask:"-"`
	IncludeIPath        string   `long:"ipath" description:"search only files whose path matches PATTERN (case insensitive)" value-name:"PATTERN" default-mask:"-"`
	ExcludePath         string   `long:"exclude-path" description:"do not search files whose path matches PATTERN" value-name:"PATTERN" default-mask:"-"`
	ExcludeIPath        string   `long:"exclude-ipath" description:"do not search files whose path matches PATTERN (case insensitive)" value-name:"PATTERN" default-mask:"-"`
	IncludeTypes        string   `short:"t" long:"type" description:"limit search to specific file types (comma-separated, see --list-types)" default-mask:"-"`
	ExcludeTypes        string   `short:"T" long:"no-type" description:"exclude specific file types (comma-separated, see --list-types)" default-mask:"-"`
	AddCustomTypes      []string `long:"add-type" description:"add custom type (see --list-types for format)" default-mask:"-" json:"-"`
	DelCustomTypes      []string `long:"del-type" description:"remove custom type" default-mask:"-" json:"-"`
	CustomTypes         map[string]string
	FieldSeparator      string   `long:"field-sep" description:"column separator (default: \":\")" default-mask:"-"`
	FilesWithMatches    bool     `short:"l" long:"files-with-matches" description:"list files containing matches"`
	FilesWithoutMatch   bool     `short:"L" long:"files-without-match" description:"list files containing no match"`
	FollowSymlinks      bool     `long:"follow" description:"follow symlinks"`
	Git                 bool     `long:"git" description:"respect .gitignore files and skip .git directories"`
	GroupByFile         bool     `long:"group" description:"group output by file (default: off)"`
	NoGroupByFile       func()   `long:"no-group" description:"do not group output by file" json:"-"`
	IgnoreCase          bool     `short:"i" long:"ignore-case" description:"case insensitive (default: off)"`
	NoIgnoreCase        func()   `short:"I" long:"no-ignore-case" description:"disable case insensitive" json:"-"`
	SmartCase           bool     `short:"s" long:"smart-case" description:"case insensitive unless pattern contains uppercase characters (default: off)"`
	NoSmartCase         func()   `short:"S" long:"no-smart-case" description:"disable smart case" json:"-"`
	NoConfig            bool     `long:"no-conf" description:"do not load config files" json:"-"`
	InvertMatch         bool     `short:"v" long:"invert-match" description:"select non-matching lines" json:"-"`
	Limit               int64    `long:"limit" description:"only show first NUM matches per file" value-name:"NUM" default-mask:"-"`
	Literal             bool     `short:"Q" long:"literal" description:"treat pattern as literal, quote meta characters"`
	Multiline           bool     `short:"m" long:"multiline" description:"multiline parsing (default: off)"`
	NoMultiline         func()   `short:"M" long:"no-multiline" description:"disable multiline parsing" json:"-"`
	OnlyMatching        bool     `long:"only-matching" description:"only show the matching part of a line" json:"-"`
	Output              string   `short:"o" long:"output" description:"write output to the specified file or network connection" value-name:"FILE|tcp://HOST:PORT" json:"-"`
	OutputLimit         int      `long:"output-limit" description:"limit output length per found match" default-mask:"-"`
	OutputSeparator     string   `long:"output-sep" description:"output separator (default: \"\\n\")" default-mask:"-" json:"-"`
	OutputUnixPath      bool     `long:"output-unixpath" description:"output file paths in unix format ('/' as path separator)"`
	Patterns            []string `short:"e" long:"regexp" description:"add pattern PATTERN to the search" value-name:"PATTERN" default-mask:"-" json:"-"`
	PatternFile         string   `short:"f" long:"regexp-file" description:"search for patterns contained in FILE (one per line)" value-name:"FILE" default-mask:"-" json:"-"`
	PrintConfig         bool     `long:"print-config" description:"print config for loaded configs + given command line arguments" json:"-"`
	Quiet               bool     `short:"q" long:"quiet" description:"suppress output, exit with return code zero if any match is found" json:"-"`
	Recursive           bool     `short:"r" long:"recursive" description:"recurse into directories (default: on)"`
	NoRecursive         func()   `short:"R" long:"no-recursive" description:"do not recurse into directories" json:"-"`
	Replace             string   `long:"replace" description:"replace numbered or named (?P<name>pattern) capture groups. Use ${1}, ${2}, $name, ... for captured submatches" json:"-"`
	ShowFilename        string
	ShowFilenameFunc    func() `long:"filename" description:"enforce printing the filename before results (default: auto)" json:"-"`
	NoShowFilenameFunc  func() `long:"no-filename" description:"disable printing the filename before results" json:"-"`
	ShowLineNumbers     bool   `short:"n" long:"line-number" description:"show line numbers (default: off)"`
	NoShowLineNumbers   func() `short:"N" long:"no-line-number" description:"do not show line numbers" json:"-"`
	ShowColumnNumbers   bool   `long:"column" description:"show column numbers"`
	NoShowColumnNumbers func() `long:"no-column" description:"do not show column numbers" json:"-"`
	ShowByteOffset      bool   `long:"byte-offset" description:"show the byte offset before each output line"`
	NoShowByteOffset    func() `long:"no-byte-offset" description:"do not show the byte offset before each output line" json:"-"`
	Stats               bool   `long:"stats" description:"show statistics"`
	TargetsOnly         bool   `long:"targets" description:"only list selected files, do not search"`
	ListTypes           bool   `long:"list-types" description:"list available file types" json:"-" default-mask:"-"`
	Version             func() `short:"V" long:"version" description:"show version and license information" json:"-"`
	WordRegexp          bool   `short:"w" long:"word-regexp" description:"only match on ASCII word boundaries"`
	WriteConfig         bool   `long:"write-config" description:"save config for loaded configs + given command line arguments" json:"-"`
	Zip                 bool   `short:"z" long:"zip" description:"search content of compressed .gz files (default: off)"`
	NoZip               func() `short:"Z" long:"no-zip" description:"do not search content of compressed .gz files" json:"-"`

	FileConditions struct {
		FileMatches     []string `long:"file-matches" description:"only show matches if file also matches PATTERN" value-name:"PATTERN"`
		LineMatches     []string `long:"line-matches" description:"only show matches if line NUM matches PATTERN" value-name:"NUM:PATTERN"`
		RangeMatches    []string `long:"range-matches" description:"only show matches if lines X-Y match PATTERN" value-name:"X:Y:PATTERN"`
		NotFileMatches  []string `long:"not-file-matches" description:"only show matches if file does not match PATTERN" value-name:"PATTERN"`
		NotLineMatches  []string `long:"not-line-matches" description:"only show matches if line NUM does not match PATTERN" value-name:"NUM:PATTERN"`
		NotRangeMatches []string `long:"not-range-matches" description:"only show matches if lines X-Y do not match PATTERN" value-name:"X:Y:PATTERN"`
	} `group:"File Condition options" json:"-"`

	MatchConditions struct {
		Preceded            []string `long:"preceded-by" description:"only show matches preceded by PATTERN" value-name:"PATTERN"`
		Followed            []string `long:"followed-by" description:"only show matches followed by PATTERN" value-name:"PATTERN"`
		Surrounded          []string `long:"surrounded-by" description:"only show matches surrounded by PATTERN" value-name:"PATTERN"`
		PrecededWithin      []string `long:"preceded-within" description:"only show matches preceded by PATTERN within NUM lines" value-name:"NUM:PATTERN"`
		FollowedWithin      []string `long:"followed-within" description:"only show matches followed by PATTERN within NUM lines" value-name:"NUM:PATTERN"`
		SurroundedWithin    []string `long:"surrounded-within" description:"only show matches surrounded by PATTERN within NUM lines" value-name:"NUM:PATTERN"`
		NotPreceded         []string `long:"not-preceded-by" description:"only show matches not preceded by PATTERN" value-name:"PATTERN"`
		NotFollowed         []string `long:"not-followed-by" description:"only show matches not followed by PATTERN" value-name:"PATTERN"`
		NotSurrounded       []string `long:"not-surrounded-by" description:"only show matches not surrounded by PATTERN" value-name:"PATTERN"`
		NotPrecededWithin   []string `long:"not-preceded-within" description:"only show matches not preceded by PATTERN within NUM lines" value-name:"NUM:PATTERN"`
		NotFollowedWithin   []string `long:"not-followed-within" description:"only show matches not followed by PATTERN within NUM lines" value-name:"NUM:PATTERN"`
		NotSurroundedWithin []string `long:"not-surrounded-within" description:"only show matches not surrounded by PATTERN within NUM lines" value-name:"NUM:PATTERN"`
	} `group:"Match Condition options" json:"-"`
}

func getHomeDir() string {
	var home string
	if runtime.GOOS == "windows" {
		home = os.Getenv("USERPROFILE")
	} else {
		home = os.Getenv("HOME")
	}
	if home == "" {
		if u, err := user.Current(); err == nil {
			home = u.HomeDir
		}
	}
	return home
}

// findLocalConfig returns the path to the local config file.
// It searches the current directory and all parent directories for a config file.
// If no config file is found, findLocalConfig returns an empty string.
func findLocalConfig() string {
	curdir, err := os.Getwd()
	if err != nil {
		curdir = "."
	}
	path, err := filepath.Abs(curdir)
	if err != nil || path == "" {
		return ""
	}
	lp := ""
	for path != lp {
		confpath := filepath.Join(path, SiftConfigFile)
		if _, err := os.Stat(confpath); err == nil {
			return confpath
		}
		lp = path
		path = filepath.Dir(path)
	}
	return ""
}

// func listTypes list the available types (built-in and custom) and exits.
func listTypes() {
	fmt.Println("The following list shows all file types supported.")
	fmt.Println("Use --type/--no-type to include/exclude file types.")
	fmt.Println("")
	var types []string
	for t := range global.fileTypesMap {
		types = append(types, t)
	}
	sort.Strings(types)
	for _, e := range types {
		t := global.fileTypesMap[e]
		var shebang string
		if t.ShebangRegex != nil {
			shebang = fmt.Sprintf("or first line matches /%s/", t.ShebangRegex)
		}
		fmt.Printf("%-15s:%s %s\n", e, strings.Join(t.Patterns, " "), shebang)
	}
	fmt.Println("")
	fmt.Println(`Custom types can be added with --add-type.`)
	fmt.Println(`Example matching *.rb, *.erb, Rakefile and all files whose first line matches the regular expression /\bruby\b/:`)
	fmt.Println(`sift --add-type 'ruby=*.rb,*.erb,Rakefile;\bruby\b'`)
	fmt.Println(`Write the definition to the config file:`)
	fmt.Println(`sift --add-type 'ruby=*.rb,*.erb,Rakefile;\bruby\b' --write-config`)
	fmt.Println(`Remove the definition from the config file:`)
	fmt.Println(`sift --del-type ruby --write-config`)
	fmt.Println("")
	os.Exit(0)
}

// LoadDefaults sets default options.
func (o *Options) LoadDefaults() {
	o.Cores = runtime.NumCPU()
	o.OutputSeparator = ""
	o.FieldSeparator = ":"
	o.ShowFilename = "auto"
	o.Color = "auto"
	o.Recursive = true
	o.CustomTypes = make(map[string]string)

	o.ColorFunc = func() {
		o.Color = "on"
	}
	o.NoColorFunc = func() {
		o.Color = "off"
	}
	o.NoIgnoreCase = func() {
		o.IgnoreCase = false
	}
	o.NoSmartCase = func() {
		o.SmartCase = false
	}
	o.NoGroupByFile = func() {
		o.GroupByFile = false
	}
	o.NoMultiline = func() {
		o.Multiline = false
	}
	o.NoRecursive = func() {
		o.Recursive = false
	}
	o.ShowFilenameFunc = func() {
		o.ShowFilename = "on"
	}
	o.NoShowFilenameFunc = func() {
		o.ShowFilename = "off"
	}
	o.NoShowLineNumbers = func() {
		o.ShowLineNumbers = false
	}
	o.NoShowColumnNumbers = func() {
		o.ShowColumnNumbers = false
	}
	o.NoShowByteOffset = func() {
		o.ShowByteOffset = false
	}
	o.NoZip = func() {
		o.Zip = false
	}
	o.Version = func() {
		fmt.Printf("sift %s (%s/%s)\n", SiftVersion, runtime.GOOS, runtime.GOARCH)
		fmt.Println("Copyright (C) 2014-2016 Sven Taute")
		fmt.Println("")
		fmt.Println("This program is free software: you can redistribute it and/or modify")
		fmt.Println("it under the terms of the GNU General Public License as published by")
		fmt.Println("the Free Software Foundation, version 3 of the License.")
		fmt.Println("")
		fmt.Println("This program is distributed in the hope that it will be useful,")
		fmt.Println("but WITHOUT ANY WARRANTY; without even the implied warranty of")
		fmt.Println("MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the")
		fmt.Println("GNU General Public License for more details.")
		fmt.Println("")
		fmt.Println("You should have received a copy of the GNU General Public License")
		fmt.Println("along with this program. If not, see <http://www.gnu.org/licenses/>.")
		os.Exit(0)
	}
}

// loadConfigFile loads options from the given config file.
func (o *Options) loadConfigFile(configFilePath string, label string) {
	configFile, err := ioutil.ReadFile(configFilePath)
	if err == nil && len(configFile) > 0 {
		if err := json.Unmarshal(configFile, &o); err != nil {
			errorLogger.Printf("cannot parse %s '%s': %s\n", label, configFilePath, err)
		}
	}
	if err != nil {
		errorLogger.Printf("cannot open %s '%s': %s\n", label, configFilePath, err)
	}
}

// LoadConfigs tries to load options from sift config files.
// if noConf is true, only a config file set via option --conf will be parsed.
func (o *Options) LoadConfigs(noConf bool, configFileArg string) {
	if !noConf {
		// load config from global sift config if file exists
		if homedir := getHomeDir(); homedir != "" {
			configFilePath := filepath.Join(homedir, SiftConfigFile)
			if _, err := os.Stat(configFilePath); err == nil {
				o.loadConfigFile(configFilePath, "global config")
			}
		}

		// load config from local sift config if file exists
		if configFilePath := findLocalConfig(); configFilePath != "" {
			if _, err := os.Stat(configFilePath); err == nil {
				o.loadConfigFile(configFilePath, "local config")
			}
		}
	}

	// load config from config option
	if configFileArg != "" {
		o.loadConfigFile(configFileArg, "config")
	}
}

// Apply processes user provided options
func (o *Options) Apply(patterns []string, targets []string) error {
	if err := o.processTypes(); err != nil {
		return err
	}

	if err := o.checkFormats(); err != nil {
		return err
	}

	if err := o.processConditions(); err != nil {
		return err
	}

	if err := o.checkCompatibility(patterns, targets); err != nil {
		return err
	}

	// handle print-config and write-config before auto detection to prevent
	// auto detected values from being written to the config file
	if err := o.processConfigOptions(); err != nil {
		return err
	}

	o.performAutoDetections(patterns, targets)

	if o.Quiet {
		global.outputFile = ioutil.Discard
	}

	if o.OnlyMatching {
		o.Replace = `$0`
	}

	for i := range patterns {
		patterns[i] = o.preparePattern(patterns[i])
	}

	runtime.GOMAXPROCS(o.Cores)
	return nil
}

// processTypes processes custom types defined on the command line
// or in the config file.
func (o *Options) processTypes() error {
	for _, e := range o.DelCustomTypes {
		if _, ok := o.CustomTypes[e]; !ok {
			return fmt.Errorf("No custom type definition for '%s' found", e)
		}
		delete(o.CustomTypes, e)
	}

	for _, e := range o.AddCustomTypes {
		s := strings.SplitN(e, "=", 2)
		if len(s) != 2 {
			return fmt.Errorf("wrong format for type definition '%s'", e)
		}
		o.CustomTypes[s[0]] = s[1]
	}

	// parse type definition, e.g. '*.pl,*.pm;\bperl\b'
	for name, e := range o.CustomTypes {
		var ft FileType
		s := strings.SplitN(e, ";", 2)
		if len(s) == 2 && s[1] != "" {
			re, err := regexp.Compile(s[1])
			if err != nil {
				return fmt.Errorf("cannot parse regular expression '%s' for custom type '%s': %s", s[1], name, err)
			}
			ft.ShebangRegex = re
		}
		patterns := strings.Split(s[0], ",")
		ft.Patterns = patterns
		global.fileTypesMap[name] = ft
	}

	if o.ListTypes {
		listTypes()
	}

	return nil
}

// checkFormats checks options for illegal formats
func (o *Options) checkFormats() error {
	if o.ExcludePath != "" {
		var err error
		global.excludeFilepathRegex, err = regexp.Compile(o.ExcludePath)
		if err != nil {
			return fmt.Errorf("cannot parse exclude filepath pattern '%s': %s\n", o.ExcludePath, err)
		}
	}
	if o.ExcludeIPath != "" {
		var err error
		global.excludeFilepathRegex, err = regexp.Compile("(?i)" + o.ExcludeIPath)
		if err != nil {
			return fmt.Errorf("cannot parse exclude filepath pattern '%s': %s\n", o.ExcludeIPath, err)
		}
	}
	if o.IncludePath != "" {
		var err error
		global.includeFilepathRegex, err = regexp.Compile(o.IncludePath)
		if err != nil {
			return fmt.Errorf("cannot parse filepath pattern '%s': %s\n", o.IncludePath, err)
		}
	}
	if o.IncludeIPath != "" {
		var err error
		global.includeFilepathRegex, err = regexp.Compile("(?i)" + o.IncludeIPath)
		if err != nil {
			return fmt.Errorf("cannot parse filepath pattern '%s': %s\n", o.IncludeIPath, err)
		}
	}

	if len(o.IncludeTypes) > 0 {
		for _, t := range strings.Split(o.IncludeTypes, ",") {
			if _, ok := global.fileTypesMap[t]; !ok {
				return fmt.Errorf("file type '%s' is not specified. See --list-types for a list of available file types", t)
			}
		}
	}
	if len(o.ExcludeTypes) > 0 {
		for _, t := range strings.Split(o.ExcludeTypes, ",") {
			if _, ok := global.fileTypesMap[t]; !ok {
				return fmt.Errorf("file type '%s' is not specified. See --list-types for a list of available file types", t)
			}
		}
	}

	if options.Cores < 0 {
		return fmt.Errorf("the number of cores must be >= 1 (or 0 for 'all')")
	}

	if options.Blocksize != "" {
		re := regexp.MustCompile(`^\d+[kKmM]?$`)
		if !re.MatchString(options.Blocksize) {
			return fmt.Errorf("cannot parse blocksize %q", options.Blocksize)
		}
		var blocksize int
		switch options.Blocksize[len(options.Blocksize)-1:] {
		case "k", "K":
			blocksize, _ = strconv.Atoi(options.Blocksize[0 : len(options.Blocksize)-1])
			InputBlockSize = blocksize * 1024
		case "m", "M":
			blocksize, _ = strconv.Atoi(options.Blocksize[0 : len(options.Blocksize)-1])
			InputBlockSize = blocksize * 1024 * 1024
		default:
			blocksize, _ := strconv.Atoi(options.Blocksize)
			InputBlockSize = blocksize
		}
		if InputBlockSize < 256*1024 {
			return fmt.Errorf("blocksize must be >= 256k")
		}
	}

	if o.OutputSeparator == "" {
		o.OutputSeparator = "\n"
	} else {
		sep, err := strconv.Unquote("\"" + o.OutputSeparator + "\"")
		if err != nil {
			return fmt.Errorf("cannot parse output separator '%s': %s\n", o.OutputSeparator, err)
		}
		o.OutputSeparator = sep
	}

	if o.Output != "" {
		if global.netTcpRegex.MatchString(o.Output) {
			netParams := global.netTcpRegex.FindStringSubmatch(o.Output)
			proto := netParams[1]
			addr := netParams[2]
			conn, err := net.Dial(proto, addr)
			if err != nil {
				return fmt.Errorf("could not connect to '%s'", o.Output)
			}
			global.outputFile = conn
		} else {
			writer, err := os.Create(o.Output)
			if err != nil {
				return fmt.Errorf("cannot open output file '%s' for writing", o.Output)
			}
			global.outputFile = writer
		}
	}

	return nil
}

// preparePattern adjusts a pattern to respect the ignore-case, literal and multiline options
func (o *Options) preparePattern(pattern string) string {
	if o.Literal {
		pattern = regexp.QuoteMeta(pattern)
	}
	if o.IgnoreCase {
		pattern = strings.ToLower(pattern)
	}
	if o.WordRegexp {
		pattern = `\b` + pattern + `\b`
	}
	pattern = "(?m)" + pattern
	if o.Multiline {
		pattern = "(?s)" + pattern
	}
	return pattern
}

// processConditions checks conditions and puts them into global.conditions
func (o *Options) processConditions() error {
	global.conditions = []Condition{}
	conditionDirections := []ConditionType{ConditionPreceded, ConditionFollowed, ConditionSurrounded}

	// parse preceded/followed/surrounded conditions without distance limit
	conditionArgs := [][]string{o.MatchConditions.Preceded, o.MatchConditions.Followed, o.MatchConditions.Surrounded,
		o.MatchConditions.NotPreceded, o.MatchConditions.NotFollowed, o.MatchConditions.NotSurrounded}
	for i := range conditionArgs {
		for _, pattern := range conditionArgs[i] {
			regex, err := regexp.Compile(o.preparePattern(pattern))
			if err != nil {
				return fmt.Errorf("cannot parse condition pattern '%s': %s\n", pattern, err)
			}
			global.conditions = append(global.conditions, Condition{regex: regex, conditionType: conditionDirections[i%3], within: -1, negated: i >= 3})
		}
	}

	// parse preceded/followed/surrounded conditions with distance limit
	conditionArgs = [][]string{o.MatchConditions.PrecededWithin, o.MatchConditions.FollowedWithin, o.MatchConditions.SurroundedWithin,
		o.MatchConditions.NotPrecededWithin, o.MatchConditions.NotFollowedWithin, o.MatchConditions.NotSurroundedWithin}
	for i := range conditionArgs {
		for _, arg := range conditionArgs[i] {
			s := strings.SplitN(arg, ":", 2)
			if len(s) != 2 {
				return fmt.Errorf("wrong format for condition option '%s'\n", arg)
			}
			within, err := strconv.Atoi(s[0])
			if err != nil {
				return fmt.Errorf("cannot parse condition option '%s': '%s' is not a number\n", arg, s[0])
			}
			if within < 0 {
				return fmt.Errorf("distance value must be >= 0\n")
			}
			regex, err := regexp.Compile(o.preparePattern(s[1]))
			if err != nil {
				return fmt.Errorf("cannot parse condition pattern '%s': %s", arg, err)
			}
			global.conditions = append(global.conditions, Condition{regex: regex, conditionType: conditionDirections[i%3], within: int64(within), negated: i >= 3})
		}
	}

	// parse match conditions
	conditionArgs = [][]string{o.FileConditions.FileMatches, o.FileConditions.NotFileMatches}
	for i := range conditionArgs {
		for _, pattern := range conditionArgs[i] {
			regex, err := regexp.Compile(o.preparePattern(pattern))
			if err != nil {
				return fmt.Errorf("cannot parse condition pattern '%s': %s\n", pattern, err)
			}
			global.conditions = append(global.conditions, Condition{regex: regex, conditionType: ConditionFileMatches, negated: i == 1})
		}
	}

	// parse line match conditions
	conditionArgs = [][]string{o.FileConditions.LineMatches, o.FileConditions.NotLineMatches}
	for i := range conditionArgs {
		for _, arg := range conditionArgs[i] {
			s := strings.SplitN(arg, ":", 2)
			if len(s) != 2 {
				return fmt.Errorf("wrong format for condition option '%s'\n", arg)
			}
			lineno, err := strconv.Atoi(s[0])
			if err != nil {
				return fmt.Errorf("cannot parse condition option '%s': '%s' is not a number\n", arg, s[0])
			}
			if lineno < 1 {
				return fmt.Errorf("line number value must be > 0\n")
			}
			regex, err := regexp.Compile(o.preparePattern(s[1]))
			if err != nil {
				return fmt.Errorf("cannot parse condition pattern '%s': %s\n", s[1], err)
			}
			global.conditions = append(global.conditions, Condition{regex: regex, conditionType: ConditionLineMatches, lineRangeStart: int64(lineno), negated: i == 1})
		}
	}

	// parse line range match conditions
	conditionArgs = [][]string{o.FileConditions.RangeMatches, o.FileConditions.NotRangeMatches}
	for i := range conditionArgs {
		for _, arg := range conditionArgs[i] {
			s := strings.SplitN(arg, ":", 3)
			if len(s) != 3 {
				return fmt.Errorf("wrong format for condition option '%s'\n", arg)
			}
			lineStart, err := strconv.Atoi(s[0])
			if err != nil {
				return fmt.Errorf("cannot parse condition option '%s': '%s' is not a number\n", arg, s[0])
			}
			lineEnd, err := strconv.Atoi(s[1])
			if err != nil {
				return fmt.Errorf("cannot parse condition option '%s': '%s' is not a number\n", arg, s[1])
			}
			if lineStart < 1 || lineEnd < 1 {
				return fmt.Errorf("line number value must be > 0\n")
			}
			regex, err := regexp.Compile(o.preparePattern(s[2]))
			if err != nil {
				return fmt.Errorf("cannot parse condition pattern '%s': %s\n", s[2], err)
			}
			global.conditions = append(global.conditions, Condition{regex: regex, conditionType: ConditionRangeMatches, lineRangeStart: int64(lineStart), lineRangeEnd: int64(lineEnd), negated: i == 1})
		}
	}

	return nil
}

// checkCompatibility checks options for incompatible combinations
func (o *Options) checkCompatibility(patterns []string, targets []string) error {
	stdinTargetFound := false
	netTargetFound := false
	for _, target := range targets {
		switch {
		case target == "-":
			stdinTargetFound = true
		case global.netTcpRegex.MatchString(target):
			netTargetFound = true
		}
	}
	if o.Context > 0 {
		o.ContextBefore = o.Context
		o.ContextAfter = o.Context
	}

	if o.InvertMatch && o.Multiline {
		return errors.New("options 'multiline' and 'invert' cannot be used together")
	}
	if netTargetFound && o.InvertMatch {
		return errors.New("option 'invert' is not supported for network targets")
	}
	if o.OutputLimit < 0 {
		return errors.New("value for option 'output-limit' must be >= 0 (0 = no limit)")
	}

	if o.OutputSeparator != "\n" && (o.ContextBefore > 0 || o.ContextAfter > 0) {
		return errors.New("context options are not supported when combined with a non-standard 'output-separator'")
	}

	if (stdinTargetFound || netTargetFound) && (o.ContextBefore > 0 || o.ContextAfter > 0) {
		return errors.New("context options are not supported when reading from STDIN or network")
	}

	if (stdinTargetFound || netTargetFound) && o.TargetsOnly {
		return errors.New("targets option not supported when reading from STDIN or network")
	}

	if (o.ContextBefore != 0 || o.ContextAfter != 0) && (o.Count || o.FilesWithMatches || o.FilesWithoutMatch) {
		return errors.New("context options cannot be combined with count or list option")
	}

	if o.FilesWithMatches && o.FilesWithoutMatch {
		return errors.New("illegal combination of list option")
	}

	if o.Zip && (o.ContextBefore != 0 || o.ContextAfter != 0) {
		return errors.New("context options cannot be used with zip search enabled")
	}

	if o.BinarySkip && o.BinaryAsText {
		return errors.New("options 'binary-skip' and 'binary-text' cannot be used together")
	}

	if o.ErrSkipLineLength && o.ErrShowLineLength {
		return errors.New("options 'err-skip-line-length' and 'err-show-line-length' cannot be used together")
	}

	if o.OnlyMatching && o.Replace != "" {
		return errors.New("options 'only-matching' and 'replace' cannot be used together")
	}

	if o.SmartCase && (len(patterns) > 1 || len(global.conditions) > 0) {
		return errors.New("the smart case option cannot be used with multiple patterns or conditions")
	}

	if o.ExcludePath != "" && o.ExcludeIPath != "" {
		return errors.New("options 'exclude-path' and 'exclude-ipath' cannot be used together")
	}
	if o.IncludePath != "" && o.IncludeIPath != "" {
		return errors.New("options 'path' and 'ipath' cannot be used together")
	}

	return nil
}

// processConfigOptions processes the options --print-config and --write-config
func (o *Options) processConfigOptions() error {
	if o.PrintConfig {
		if homedir := getHomeDir(); homedir != "" {
			globalConfigFilePath := filepath.Join(homedir, SiftConfigFile)
			fmt.Fprintf(os.Stderr, "Global config file path: %s\n", globalConfigFilePath)
		} else {
			errorLogger.Println("could not detect user home directory.")
		}

		localConfigFilePath := findLocalConfig()
		if localConfigFilePath != "" {
			fmt.Fprintf(os.Stderr, "Local config file path: %s\n", localConfigFilePath)
		} else {
			fmt.Fprintf(os.Stderr, "No local config file found.\n")
		}

		conf, err := json.MarshalIndent(o, "", "    ")
		if err != nil {
			return fmt.Errorf("cannot convert config to JSON: %s", err)
		}
		fmt.Println(string(conf))
		os.Exit(0)
	}

	if o.WriteConfig {
		var configFilePath string
		localConfigFilePath := findLocalConfig()
		if localConfigFilePath != "" {
			configFilePath = localConfigFilePath
		} else {
			if homedir := getHomeDir(); homedir != "" {
				configFilePath = filepath.Join(homedir, SiftConfigFile)
			} else {
				return errors.New("could not detect user home directory")
			}
		}
		conf, err := json.MarshalIndent(o, "", "    ")
		if err != nil {
			return fmt.Errorf("cannot convert config to JSON: %s", err)
		}
		if err := ioutil.WriteFile(configFilePath, conf, os.ModePerm); err != nil {
			return fmt.Errorf("cannot write config file: %s", err)
		}
		fmt.Printf("Saved config to '%s'.\n", configFilePath)
		os.Exit(0)
	}

	return nil
}

// performAutoDetections sets options that are set to "auto"
func (o *Options) performAutoDetections(patterns []string, targets []string) {
	stdinTargetFound := false
	netTargetFound := false
	for _, target := range targets {
		switch {
		case target == "-":
			stdinTargetFound = true
		case global.netTcpRegex.MatchString(target):
			netTargetFound = true
		}
	}

	if len(global.conditions) == 0 {
		global.streamingAllowed = true

		if len(targets) == 1 {
			if stdinTargetFound || netTargetFound {
				global.streamingThreshold = 0
				o.GroupByFile = false
			} else {
				stat, err := os.Stat(targets[0])
				if err == nil && stat.Mode()&os.ModeType == 0 {
					global.streamingThreshold = 0
				}
			}
		}
	}

	if o.ShowFilename == "auto" {
		if len(targets) == 1 {
			fileinfo, err := os.Stat(targets[0])
			if err == nil && fileinfo.IsDir() {
				o.ShowFilename = "on"
			} else {
				o.ShowFilename = "off"
			}
		} else {
			o.ShowFilename = "on"
		}
	}

	if o.Color == "auto" {
		// auto activate colored output only if STDOUT is a terminal
		if o.Output == "" {
			if runtime.GOOS != "windows" && terminal.IsTerminal(int(os.Stdout.Fd())) {
				o.Color = "on"
			} else {
				o.Color = "off"
			}
		} else {
			o.Color = "off"
		}
	}

	if o.GroupByFile {
		if !terminal.IsTerminal(int(os.Stdout.Fd())) {
			o.GroupByFile = false
		}
	}

	if !o.IgnoreCase && o.SmartCase {
		if len(patterns) >= 1 {
			if m, _ := regexp.MatchString("[A-Z]", patterns[0]); !m {
				o.IgnoreCase = true
			}
		}
	}

	if o.Cores == 0 {
		o.Cores = runtime.NumCPU()
	}

	if o.Color == "on" {
		global.termHighlightFilename = fmt.Sprintf("\033[%d;%d;%dm", 1, 35, 49)
		global.termHighlightLineno = fmt.Sprintf("\033[%d;%d;%dm", 1, 32, 49)
		global.termHighlightMatch = fmt.Sprintf("\033[%d;%d;%dm", 1, 31, 49)
		global.termHighlightReset = fmt.Sprintf("\033[%d;%d;%dm", 0, 39, 49)
	} else {
		global.termHighlightFilename = ""
		global.termHighlightLineno = ""
		global.termHighlightMatch = ""
		global.termHighlightReset = ""
	}
}
