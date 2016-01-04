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

/*
Package gitignore provides a Checker that can be used to determine
whether a specific file is excluded by a .gitignore file.

This package targets to support the full gitignore pattern syntax
documented here: https://git-scm.com/docs/gitignore

Multiple .gitignore files with multiple matching patterns
are supported. A cache is used to prevent loading the same
.gitignore file again when checking different paths.
*/
package gitignore

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
)

const (
	GitIgnoreFilename = ".gitignore"
	GitFoldername     = ".git"
)

// Checker allows to check whether a given file is excluded by the
// relevant .gitignore files for a given base path and holds
// a cache of already parsed .gitignore files.
type Checker struct {
	basePath       string
	gitIgnores     []*gitIgnore
	gitIgnoreCache *GitIgnoreCache
}

// gitIgnore holds all patterns of a specific .gitignore file.
type gitIgnore struct {
	basePath string
	patterns []patternMatcher
}

// GitIgnoreCache holds already parsed .gitignore files.
type GitIgnoreCache struct {
	cache map[string]*gitIgnore
	mu    sync.RWMutex
}

// basePattern holds the properties of a gitignore pattern.
type basePattern struct {
	// the base path of the corresponding .gitignore file
	basePath string
	// only match directories (pattern ends with "/")
	matchDirOnly bool
	// include (pattern starts with "!")
	negated bool
	// match against root of basePath (pattern starts with "/")
	leadingSlash bool
	// the normalized content of the pattern
	content string
}

// simplePattern describes a pattern matching filenames.
// The pattern must not contain special characters.
type simplePattern struct {
	basePattern
}

// filePattern describes a pattern with special characters matching filenames.
// The pattern may contain special characters.
type filePattern struct {
	basePattern
}

// pathPattern describes a pattern matching a partial or complete file path.
type pathPattern struct {
	basePattern
	// depth describes the depth (number of folders) of the pattern
	depth int
}

// regexPattern describes a regex pattern matching the file path.
type regexPattern struct {
	basePattern
	re *regexp.Regexp
}

// patternMatcher is an interface for all pattern types.
type patternMatcher interface {
	Matches(string, os.FileInfo) bool
	Negated() bool
}

// NewChecker returns a new Checker instance.
func NewChecker() *Checker {
	c := &Checker{}
	c.gitIgnoreCache = NewGitIgnoreCache()
	return c
}

// NewCheckerWithCache returns a new Checker instance that uses the given cache.
func NewCheckerWithCache(cache *GitIgnoreCache) *Checker {
	c := &Checker{}
	c.gitIgnoreCache = cache
	return c
}

// Check returns whether the specified path is excluded by a .gitignore file.
func (c *Checker) Check(path string, fi os.FileInfo) bool {
	res := false
	for _, gi := range c.gitIgnores {
		if ignore, matched := gi.check(path, fi); matched {
			res = ignore
			break
		}
	}
	return res
}

// LoadBasePath initializes the Checker instance with a new base path
// and loads all relevant .gitignore files. Already known .gitignore
// files are taken from the cache.
//
// This function re-initializes the whole Checker, thus it is not
// thread-safe to call this function while using the Check() function
// of the same instance.
func (c *Checker) LoadBasePath(path string) error {
	curPath, err := filepath.Abs(path)
	if err != nil || curPath == "" {
		return err
	}

	c.gitIgnores = []*gitIgnore{}

	lastPath := ""
	for curPath != lastPath {
		ignoreFile := filepath.Join(curPath, GitIgnoreFilename)
		if _, err := os.Stat(ignoreFile); err == nil {
			var gi *gitIgnore
			gi, err = c.gitIgnoreCache.get(ignoreFile)
			if err != nil {
				return err
			}
			c.gitIgnores = append(c.gitIgnores, gi)
		}
		lastPath = curPath
		curPath = filepath.Dir(curPath)
	}

	return nil
}

// newGitIgnore returns a gitIgnore instance for the given .gitignore file.
func newGitIgnore(path string) (*gitIgnore, error) {
	basePath := filepath.Dir(path)
	var gi *gitIgnore = &gitIgnore{basePath: basePath}
	err := gi.loadIgnoreFile(path)
	return gi, err
}

// check checks whether the given path is excluded by the gitIgnore instance.
func (gi gitIgnore) check(path string, fi os.FileInfo) (ignore bool, matched bool) {
	fullpath, _ := filepath.Abs(path)
	if len(fullpath) <= len(gi.basePath) || !strings.HasPrefix(fullpath, gi.basePath) {
		return false, false
	}

	testpath := fullpath[len(gi.basePath)+1:]
	for i := len(gi.patterns) - 1; i >= 0; i-- {
		p := gi.patterns[i]
		if p.Matches(testpath, fi) {
			ignore = !p.Negated()
			matched = true
			return
		}
	}
	return false, false
}

// loadIgnoreFile loads a .gitignore file and processes
// all found patterns.
func (c *gitIgnore) loadIgnoreFile(path string) error {
	basePath := filepath.Dir(path)
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		c.addPattern(scanner.Text(), basePath)
	}
	if err = scanner.Err(); err != nil {
		return err
	}

	return nil
}

// addPattern parses the given pattern and adds it to
// the gitIgnore instance.
func (c *gitIgnore) addPattern(pattern string, basePath string) {
	negated := false
	matchDirOnly := false
	leadingSlash := false

	if strings.Trim(pattern, " ") == "" {
		return
	}
	if strings.HasPrefix(pattern, "#") {
		return
	}

	if strings.HasPrefix(pattern, "!") {
		negated = true
		pattern = pattern[1:]
	} else if strings.HasPrefix(pattern, `\!`) {
		pattern = pattern[1:]
	}
	if strings.HasPrefix(pattern, "/") {
		leadingSlash = true
		pattern = pattern[1:]
	}
	if strings.HasSuffix(pattern, "/") {
		matchDirOnly = true
		pattern = pattern[:len(pattern)-1]
	}

	var p patternMatcher
	var base basePattern
	base = basePattern{
		basePath:     basePath,
		content:      pattern,
		negated:      negated,
		leadingSlash: leadingSlash,
		matchDirOnly: matchDirOnly,
	}
	if strings.Contains(pattern, "**") {
		p = newRegexPattern(base)
	} else {
		if strings.Contains(pattern, "/") || leadingSlash {
			p = newPathPattern(base)
		} else {
			if strings.ContainsAny(pattern, "*?[") {
				p = newFilePattern(base)
			} else {
				p = newSimplePattern(base)
			}
		}
	}
	c.patterns = append(c.patterns, p)
}

// NewGitIgnoreCache creates and returns a new gitignore cache.
func NewGitIgnoreCache() *GitIgnoreCache {
	c := &GitIgnoreCache{}
	c.cache = make(map[string]*gitIgnore)
	return c
}

// get returns the matching GitIgnore instance from the cache or
// creates a new one and stores it in the cache.
func (c *GitIgnoreCache) get(path string) (*gitIgnore, error) {
	c.mu.RLock()
	if gi, ok := c.cache[path]; ok {
		c.mu.RUnlock()
		return gi, nil
	}
	c.mu.RUnlock()
	gi, err := newGitIgnore(path)
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	c.cache[path] = gi
	c.mu.Unlock()
	return gi, nil
}

func (p basePattern) Negated() bool {
	return p.negated
}

func newSimplePattern(base basePattern) patternMatcher {
	return simplePattern{base}
}

func (p simplePattern) Matches(path string, fi os.FileInfo) bool {
	if p.matchDirOnly && !fi.IsDir() {
		return false
	}
	filename := fi.Name()
	return filename == p.content
}

func newFilePattern(base basePattern) patternMatcher {
	return filePattern{base}
}

func (p filePattern) Matches(path string, fi os.FileInfo) bool {
	if p.matchDirOnly && !fi.IsDir() {
		return false
	}
	filename := fi.Name()
	res, err := filepath.Match(p.content, filename)
	if err != nil {
		return false
	}
	return res
}

func newPathPattern(base basePattern) patternMatcher {
	depth := 0
	if !base.leadingSlash {
		depth = strings.Count(base.content, "/")
	}
	p := pathPattern{base, depth}
	return p
}

func (p pathPattern) Matches(path string, fi os.FileInfo) bool {
	if p.matchDirOnly && !fi.IsDir() {
		return false
	}
	if runtime.GOOS == "windows" {
		path = filepath.ToSlash(path)
	}
	if p.leadingSlash {
		res, err := filepath.Match(p.content, path)
		if err != nil {
			return false
		}
		return res
	} else {
		slashes := 0
		pos := 0
		for pos = len(path) - 1; pos >= 0; pos-- {
			if path[pos:pos+1] == "/" {
				slashes++
				if slashes > p.depth {
					break
				}
			}
		}
		if slashes < p.depth {
			return false
		}
		checkpath := path[pos+1:]
		res, err := filepath.Match(p.content, checkpath)
		if err != nil {
			return false
		}
		return res
	}
}

func newRegexPattern(base basePattern) patternMatcher {
	matchStart := false
	matchEnd := false
	content := base.content
	if strings.HasPrefix(content, "**/") {
		content = content[3:]
	} else {
		matchStart = true
	}
	if strings.HasSuffix(content, "/**") {
		content = content[:len(content)-3]
	} else {
		matchEnd = true
	}

	parts := strings.Split(content, "**")
	for i, _ := range parts {
		parts[i] = regexp.QuoteMeta(parts[i])
	}
	pattern := strings.Join(parts, ".*?")
	if matchStart {
		pattern = "^" + pattern
	}
	if matchEnd {
		pattern = pattern + "$"
	}

	re := regexp.MustCompile(pattern)
	p := regexPattern{base, re}
	return p
}

func (p regexPattern) Matches(path string, fi os.FileInfo) bool {
	if p.matchDirOnly && !fi.IsDir() {
		return false
	}
	if runtime.GOOS == "windows" {
		path = filepath.ToSlash(path)
	}
	return p.re.MatchString(path)
}
