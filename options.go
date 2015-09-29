// sift
// Copyright (C) 2014-2015 Sven Taute
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
)

type Options struct {
	BinarySkip         bool `long:"binary-skip" description:"skip files that seem to be binary"`
	BinaryAsText       bool `short:"a" long:"binary-text" description:"process files that seem to be binary as text"`
	Color              string
	ColorFunc          func()   `long:"color" description:"enable colored output (default: auto)" json:"-"`
	NoColorFunc        func()   `long:"no-color" description:"disable colored output" json:"-"`
	Context            int      `short:"C" long:"context" description:"show NUM context lines" value-name:"NUM" json:"-"`
	ContextAfter       int      `short:"A" long:"context-after" description:"show NUM context lines after match" value-name:"NUM" json:"-"`
	ContextBefore      int      `short:"B" long:"context-before" description:"show NUM context lines before match" value-name:"NUM" json:"-"`
	Cores              int      `short:"j" long:"cores" description:"limit used CPU Cores (default: 0 = all)" default-mask:"-"`
	Count              bool     `short:"c" long:"count" description:"print count of matches per file" json:"-"`
	IncludeDirs        []string `long:"dirs" description:"recurse only into directories whose name matches GLOB" value-name:"GLOB" default-mask:"-"`
	ExcludeDirs        []string `long:"exclude-dirs" description:"do not recurse into directories whose name matches GLOB" value-name:"GLOB" default-mask:"-"`
	IncludeExtensions  string   `short:"x" long:"ext" description:"limit search to specific file extensions (comma-separated)" default-mask:"-"`
	ExcludeExtensions  string   `short:"X" long:"exclude-ext" description:"exclude specific file extensions (comma-separated)" default-mask:"-"`
	IncludeFiles       []string `long:"files" description:"search only files whose name matches GLOB" value-name:"GLOB" default-mask:"-"`
	ExcludeFiles       []string `long:"exclude-files" description:"do not select files whose name matches GLOB while recursing" value-name:"GLOB" default-mask:"-"`
	IncludePath        string   `long:"path" description:"search only files whose path matches PATTERN" value-name:"PATTERN" default-mask:"-"`
	ExcludePath        string   `long:"exclude-path" description:"do not select files whose path matches PATTERN while recursing" value-name:"PATTERN" default-mask:"-"`
	IncludeTypes       string   `short:"t" long:"type" description:"limit search to specific file types (comma-separated, see --list-types)" default-mask:"-"`
	ExcludeTypes       string   `short:"T" long:"no-type" description:"exclude specific file types (comma-separated, --list-types)" default-mask:"-"`
	FilesWithMatches   bool     `short:"l" long:"files-with-matches" description:"list files containing matches"`
	FilesWithoutMatch  bool     `short:"L" long:"files-without-match" description:"list files containing no match"`
	GroupByFile        bool     `long:"group" description:"group output by file (default: off)"`
	NoGroupByFile      func()   `long:"no-group" description:"do not group output by file" json:"-"`
	IgnoreCase         bool     `short:"i" long:"ignore-case" description:"case insensitive (default: off)"`
	NoIgnoreCase       func()   `short:"I" long:"no-ignore-case" description:"disable case insensitive" json:"-"`
	InvertMatch        bool     `short:"v" long:"invert-match" description:"select non-matching lines" json:"-"`
	Limit              int64    `long:"limit" description:"only show first NUM matches per file" value-name:"NUM" default-mask:"-"`
	Multiline          bool     `short:"m" long:"multiline" description:"multiline parsing (default: off)"`
	NoMultiline        func()   `short:"M" long:"no-multiline" description:"disable multiline parsing" json:"-"`
	Output             string   `short:"o" long:"output" description:"write output to the specified file or network connection" value-name:"FILE|tcp://HOST:PORT" json:"-"`
	OutputLimit        int      `long:"output-limit" description:"limit output length per found match" default-mask:"-"`
	OutputSeparator    string   `long:"output-sep" description:"output separator (default: \"\\n\")" default-mask:"-" json:"-"`
	Patterns           []string `short:"e" long:"regexp" description:"add pattern PATTERN to the search" value-name:"PATTERN" default-mask:"-" json:"-"`
	PatternFile        string   `short:"f" long:"regexp-file" description:"search for patterns contained in FILE (one per line)" value-name:"FILE" default-mask:"-" json:"-"`
	PrintConfig        bool     `long:"print-config" description:"print config for loaded configs + given command line arguments" json:"-"`
	Quiet              bool     `short:"q" long:"quiet" description:"suppress output, exit with return code zero if any match is found" json:"-"`
	Recursive          bool     `short:"r" long:"recursive" description:"recurse into directories (default: on)"`
	NoRecursive        func()   `short:"R" long:"no-recursive" description:"do not recurse into directories" json:"-"`
	Replace            string   `long:"replace" description:"replace matches. Use ${1}, ${2}, $name, ... for captured submatches" json:"-"`
	ShowFilename       string
	ShowFilenameFunc   func() `long:"filename" description:"enforce printing the filename before results (default: auto)" json:"-"`
	NoShowFilenameFunc func() `long:"no-filename" description:"disable printing the filename before results" json:"-"`
	ShowLineNumbers    bool   `short:"n" long:"line-number" description:"show line numbers (default: off)"`
	NoShowLineNumbers  func() `short:"N" long:"no-line-number" description:"do not show line numbers" json:"-"`
	Stats              bool   `long:"stats" description:"show statistics"`
	ListTypes          func() `long:"list-types" description:"list available file types" json:"-" default-mask:"-"`
	Version            func() `short:"V" long:"version" description:"show version and license information" json:"-"`
	WriteConfig        bool   `long:"write-config" description:"save config for loaded configs + given command line arguments" json:"-"`
	Zip                bool   `short:"z" long:"zip" description:"search content of compressed .gz files (default: off)"`
	NoZip              func() `short:"Z" long:"no-zip" description:"do not search content of compressed .gz files" json:"-"`

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

// LoadDefaults sets default options and tries to load options from sift config files.
func (o *Options) LoadDefaults() {
	o.Cores = runtime.NumCPU()
	o.OutputSeparator = ""
	o.ShowFilename = "auto"
	o.Color = "auto"
	o.Recursive = true

	o.ColorFunc = func() {
		o.Color = "on"
	}
	o.NoColorFunc = func() {
		o.Color = "off"
	}
	o.NoIgnoreCase = func() {
		o.IgnoreCase = false
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
	o.NoZip = func() {
		o.Zip = false
	}
	o.Version = func() {
		fmt.Println("sift", SiftVersion)
		fmt.Println("Copyright (C) 2014-2015 Sven Taute")
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

	o.ListTypes = func() {
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
				shebang = fmt.Sprintf(" or first line matches /%s/", t.ShebangRegex)
			}
			fmt.Printf("%-15s%s%s\n", t.Name+":", strings.Join(t.Patterns, " "), shebang)
		}
		os.Exit(0)
	}

	// load config from global sift config if file exists
	if homedir := getHomeDir(); homedir != "" {
		configFilePath := filepath.Join(homedir, SiftConfigFile)
		configFile, err := ioutil.ReadFile(configFilePath)
		if err == nil && len(configFile) > 0 {
			if err := json.Unmarshal(configFile, &o); err != nil {
				errorLogger.Printf("cannot parse global config '%s': %s\n", configFilePath, err)
			}
		}
	}

	// load config from local sift config if file exists
	if configFilePath := findLocalConfig(); configFilePath != "" {
		configFile, err := ioutil.ReadFile(configFilePath)
		if err == nil && len(configFile) > 0 {
			if err := json.Unmarshal(configFile, &o); err != nil {
				errorLogger.Printf("cannot parse local config '%s': %s\n", configFilePath, err)
			}
		}
	}
}

// Apply processes user provided options
func (o *Options) Apply(patterns []string, targets []string) error {
	if err := o.checkFormats(); err != nil {
		return err
	}

	if err := o.processConditions(); err != nil {
		return err
	}

	if err := o.checkCompatibility(targets); err != nil {
		return err
	}

	// handle print-config and write-config before auto detection to prevent
	// auto detected values from being written to the config file
	if err := o.processConfigOptions(); err != nil {
		return err
	}

	o.performAutoDetections(targets)

	if o.Quiet {
		global.outputFile = ioutil.Discard
	}

	for i := range patterns {
		patterns[i] = o.preparePattern(patterns[i])
	}

	runtime.GOMAXPROCS(o.Cores)
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
	if o.IncludePath != "" {
		var err error
		global.includeFilepathRegex, err = regexp.Compile(o.IncludePath)
		if err != nil {
			return fmt.Errorf("cannot parse filepath pattern '%s': %s\n", o.IncludePath, err)
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

// preparePattern adjusts a pattern to respect the ignore-case and multiline options
func (o *Options) preparePattern(pattern string) string {
	if o.IgnoreCase {
		pattern = strings.ToLower(pattern)
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
func (o *Options) checkCompatibility(targets []string) error {
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
func (o *Options) performAutoDetections(targets []string) {
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
		// auto activate colored output only if STDOUT is a device,
		// disable for files and pipes
		if o.Output == "" {
			stat, err := os.Stdout.Stat()
			if err == nil && stat.Mode()&os.ModeDevice != 0 {
				o.Color = "on"
			} else {
				o.Color = "off"
			}
		} else {
			o.Color = "off"
		}
	}

	if o.GroupByFile {
		stat, err := os.Stdout.Stat()
		if err != nil || stat.Mode()&os.ModeDevice == 0 {
			o.GroupByFile = false
		}
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
