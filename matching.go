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
	"bufio"
	"bytes"
	"io"
	"os"
	"regexp"
	"sort"
)

// processReader is the main routine working on an io.Reader
func processReader(reader io.Reader, matchRegexes []*regexp.Regexp, data []byte, testBuffer []byte, target string) error {
	var (
		bufferOffset             int
		err                      error
		isEOF                    bool
		lastInputBlockSize       int
		lastMatch                *Match
		lastRoundMultilineWindow bool
		lastSeekAmount           int
		lastValidMatchRange      int
		linecount                int64 = 1
		matchChan                chan Matches
		matchCount               int64
		offset                   int64
		resultIsBinary           bool
		resultStreaming          bool
		validMatchRange          int
	)
	matches := make([]Match, 0, 16)
	conditionMatches := make([]Match, 0, 16)

	for {
		if isEOF {
			break
		}
		var length int
		lastConditionMatch := len(conditionMatches) - 1

		if options.Multiline {
			if lastRoundMultilineWindow {
				// if the last input block was greater than the sliding window size, that last part has to be processed again
				copy(data[bufferOffset:bufferOffset+InputMultilineWindow], data[lastInputBlockSize-InputMultilineWindow:lastInputBlockSize])
				length, err = reader.Read(data[bufferOffset+InputMultilineWindow:])
				length += bufferOffset
				length += InputMultilineWindow
			} else {
				length, err = reader.Read(data[bufferOffset:])
				length += bufferOffset
			}
			if err != nil {
				if err == io.EOF {
					isEOF = true
				} else {
					return err
				}
			}
			lastInputBlockSize = length

			// if the input block was greater than the sliding window size, only process a part of it
			if !isEOF && length > InputMultilineWindow {
				validMatchRange = length - InputMultilineWindow
				lastRoundMultilineWindow = true
			} else {
				validMatchRange = length
				lastRoundMultilineWindow = false
			}
		} else {
			// single line mode
			length, err = reader.Read(data[bufferOffset:])
			if err != nil {
				if err == io.EOF {
					isEOF = true
				} else {
					return err
				}
			}
			length += bufferOffset
			validMatchRange = length
			lastInputBlockSize = length
		}

		lastValidMatchRange = validMatchRange

		// check if file is binary (0x00 found in first 256 bytes)
		if offset == 0 {
			s := 256
			if length < s {
				s = length
			}
			for i := 0; i < s; i++ {
				if data[i] == 0 {
					resultIsBinary = true
					if options.BinarySkip {
						return nil
					}
					break
				}
			}
		}

		// adjust validMatchRange to end on a newline
		lastSeekAmount = 0
		if !isEOF {
			newLineFound := false
			var pos int
			for pos = validMatchRange - 1; pos >= 0; pos-- {
				if data[pos] == '\n' {
					newLineFound = true
					break
				}
			}
			if newLineFound {
				lastSeekAmount = validMatchRange - 1 - pos
				validMatchRange = validMatchRange - lastSeekAmount
				bufferOffset = 0
			} else {
				if lastInputBlockSize == InputBlockSize {
					return errLineTooLong
				}
				bufferOffset = validMatchRange
				continue
			}
		}

		var testDataPtr []byte
		if options.IgnoreCase {
			bytesToLower(data, testBuffer, length)
			testDataPtr = testBuffer[0:length]
		} else {
			testDataPtr = data[0:length]
		}

		var newMatches Matches
		for _, re := range matchRegexes {
			tmpMatches := getMatches(re, data, testDataPtr, offset, length, validMatchRange, 0, target)
			if len(tmpMatches) > 0 {
				newMatches = append(newMatches, tmpMatches...)
			}
		}

		// sort matches and filter duplicates
		if len(newMatches) > 0 {
			sort.Sort(Matches(newMatches))
			var validMatch bool
			prevMatch := lastMatch
			for i := 0; i < len(newMatches); {
				validMatch = false
				if prevMatch == nil {
					validMatch = true
				} else {
					m := newMatches[i]
					if (!options.Multiline && m.lineEnd > prevMatch.lineEnd) ||
						(options.Multiline && m.start >= prevMatch.end) {
						validMatch = true
					}
				}
				if validMatch {
					prevMatch = &newMatches[i]
					i++
				} else {
					copy(newMatches[i:], newMatches[i+1:])
					newMatches = newMatches[0 : len(newMatches)-1]
				}
			}
		}

		for conditionID, condition := range global.conditions {
			tmpMatches := getMatches(condition.regex, data, testDataPtr, offset, length, validMatchRange, conditionID, target)
			if len(tmpMatches) > 0 {
				conditionMatches = append(conditionMatches, tmpMatches...)
			}
		}
		if len(conditionMatches) > 0 {
			sort.Sort(Matches(conditionMatches))
		}

		if options.ShowLineNumbers || options.ContextBefore > 0 || options.ContextAfter > 0 || len(global.conditions) > 0 {
			linecount = countLines(data, lastConditionMatch, newMatches, conditionMatches, offset, validMatchRange, linecount)
		}

		if len(newMatches) > 0 {
			// if a list option is used exit here if possible
			if (options.FilesWithMatches || options.FilesWithoutMatch) && !options.Count && len(global.conditions) == 0 {
				global.resultsChan <- &Result{target: target, matches: []Match{Match{}}}
				return nil
			}

			lastMatch = &newMatches[len(newMatches)-1]
			if resultStreaming {
				matchChan <- newMatches
			} else {
				matches = append(matches, newMatches...)
				if len(matches) > global.streamingThreshold && global.streamingAllowed {
					resultStreaming = true
					matchChan = make(chan Matches, 16)
					global.resultsChan <- &Result{target: target, matches: matches, streaming: true, matchChan: matchChan, isBinary: resultIsBinary}
					defer func() {
						close(matchChan)
					}()
				}
			}

			matchCount += int64(len(newMatches))
			if options.Limit != 0 && matchCount >= options.Limit {
				break
			}
		}

		// copy the bytes not processed after the last newline to the beginning of the buffer
		if lastSeekAmount > 0 {
			copy(data[bufferOffset:bufferOffset+lastSeekAmount], data[lastValidMatchRange-lastSeekAmount:lastValidMatchRange])
			bufferOffset += lastSeekAmount
		}

		offset += int64(validMatchRange)
	}

	if !resultStreaming {
		global.resultsChan <- &Result{target: target, matches: matches, conditionMatches: conditionMatches, streaming: false, isBinary: resultIsBinary}
	}
	return nil
}

// getMatches gets all matches in the provided data, it is used for normal and condition matches.
//
// data contains the original data.
// testBuffer contains the data to test the regex against (potentially modified, e.g. to support the ignore case option).
// length contains the length of the provided data.
// matches are only valid if they start within the validMatchRange.
func getMatches(regex *regexp.Regexp, data []byte, testBuffer []byte, offset int64, length int, validMatchRange int, conditionID int, target string) Matches {
	var matches Matches
	if allIndex := regex.FindAllIndex(testBuffer, -1); allIndex != nil {
		// for _, index := range allindex {
		for mi := 0; mi < len(allIndex); mi++ {
			index := allIndex[mi]
			start := index[0]
			end := index[1]
			// \s always matches newline, leading to incorrect matches in non-multiline mode
			// analyze match and reject false matches
			if !options.Multiline {
				// remove newlines at the beginning of the match
				for ; start < length && end > start && data[start] == 0x0a; start++ {
				}
				// remove newlines at the end of the match
				for ; end > 0 && end > start && data[end-1] == 0x0a; end-- {
				}
				// check if the corrected match is still valid
				if !regex.Match(testBuffer[start:end]) {
					continue
				}
				// check if the match contains newlines
				if bytes.Contains(data[start:end], []byte{0x0a}) {
					// Rebuild the complete lines to check whether these contain valid matches.
					// In very rare cases, multiple lines may contain a valid match. As multiple
					// matches cannot be processed correctly here, requeue them to be processed again.
					lineStart := start
					lineEnd := end
					for lineStart > 0 && data[lineStart-1] != 0x0a {
						lineStart--
					}
					for lineEnd < length && data[lineEnd] != 0x0a {
						lineEnd++
					}

					lastStart := lineStart
					for pos := lastStart + 1; pos < lineEnd; pos++ {
						if data[pos] == 0x0a || pos == lineEnd-1 {
							if pos == lineEnd-1 && data[pos] != 0x0a {
								pos++
							}
							if idx := regex.FindIndex(testBuffer[lastStart:pos]); idx != nil {
								start = lastStart
								end = pos
								start = lastStart + idx[0]
								end = lastStart + idx[1]
								allIndex = append(allIndex, []int{start, end})
							}
							lastStart = pos + 1
						}
					}
					continue
				}
			}

			lineStart := start
			lineEnd := end
			if options.Multiline && start >= validMatchRange {
				continue
			}
			for lineStart > 0 && data[lineStart-1] != 0x0a {
				lineStart--
			}
			for lineEnd < length && data[lineEnd] != 0x0a {
				lineEnd++
			}

			var contextBefore *string
			var contextAfter *string

			if options.ContextBefore > 0 {
				var contextBeforeStart int
				if lineStart > 0 {
					contextBeforeStart = lineStart - 1
					precedingLinesFound := 0
					for contextBeforeStart > 0 {
						if data[contextBeforeStart-1] == 0x0a {
							precedingLinesFound++
							if precedingLinesFound == options.ContextBefore {
								break
							}
						}
						contextBeforeStart--
					}
					if precedingLinesFound < options.ContextBefore && contextBeforeStart == 0 && offset > 0 {
						contextBefore = getBeforeContextFromFile(target, offset, start)
					} else {
						tmp := string(data[contextBeforeStart : lineStart-1])
						contextBefore = &tmp
					}
				} else {
					if offset > 0 {
						contextBefore = getBeforeContextFromFile(target, offset, start)
					} else {
						contextBefore = nil
					}
				}
			}

			if options.ContextAfter > 0 {
				var contextAfterEnd int
				if lineEnd < length-1 {
					contextAfterEnd = lineEnd
					followingLinesFound := 0
					for contextAfterEnd < length-1 {
						if data[contextAfterEnd+1] == 0x0a {
							followingLinesFound++
							if followingLinesFound == options.ContextAfter {
								contextAfterEnd++
								break
							}
						}
						contextAfterEnd++
					}
					if followingLinesFound < options.ContextAfter && contextAfterEnd == length-1 {
						contextAfter = getAfterContextFromFile(target, offset, end)
					} else {
						tmp := string(data[lineEnd+1 : contextAfterEnd])
						contextAfter = &tmp
					}
				} else {
					contextAfter = getAfterContextFromFile(target, offset, end)
				}
			}

			m := Match{
				conditionID:   conditionID,
				start:         offset + int64(start),
				end:           offset + int64(end),
				lineStart:     offset + int64(lineStart),
				lineEnd:       offset + int64(lineEnd),
				match:         string(data[start:end]),
				line:          string(data[lineStart:lineEnd]),
				contextBefore: contextBefore,
				contextAfter:  contextAfter,
			}

			// handle special case where '^' matches after the last newline
			if int(lineStart) != validMatchRange {
				matches = append(matches, m)
			}
		}
	}
	return matches
}

// countLines counts the linebreaks within the given buffer and calculates the correct line numbers for new matches
func countLines(data []byte, lastConditionMatch int, matches Matches, conditionMatches Matches, offset int64, validMatchRange int, lineCount int64) int64 {
	currentMatch := 0
	currentConditionMatch := lastConditionMatch + 1
	if currentMatch < len(matches) || currentConditionMatch < len(conditionMatches) {
		for i := 0; i < validMatchRange; i++ {
			if data[i] == 0xa {
				for currentMatch < len(matches) && offset+int64(i) >= matches[currentMatch].lineStart {
					matches[currentMatch].lineno = lineCount
					currentMatch++
				}
				for currentConditionMatch < len(conditionMatches) && offset+int64(i) >= conditionMatches[currentConditionMatch].lineStart {
					conditionMatches[currentConditionMatch].lineno = lineCount
					currentConditionMatch++
				}
				lineCount++
			}
		}
		// check for matches on last line without newline
		for currentMatch < len(matches) && offset+int64(validMatchRange) >= matches[currentMatch].lineStart {
			matches[currentMatch].lineno = lineCount
			currentMatch++
		}
		for currentConditionMatch < len(conditionMatches) && offset+int64(validMatchRange) >= conditionMatches[currentConditionMatch].lineStart {
			conditionMatches[currentConditionMatch].lineno = lineCount
			currentConditionMatch++
		}
	} else {
		lineCount += int64(countNewlines(data, validMatchRange))
	}
	return lineCount
}

// applyConditions removes matches from a result that do not fulfill all conditions
func (result *Result) applyConditions() {
	if len(result.matches) == 0 || len(global.conditions) == 0 {
		return
	}

	// check conditions that are independent of found matches
	conditionStatus := make([]bool, len(global.conditions))
	var conditionFulfilled bool
	for _, conditionMatch := range result.conditionMatches {
		conditionFulfilled = false
		switch global.conditions[conditionMatch.conditionID].conditionType {
		case ConditionFileMatches:
			conditionFulfilled = true
		case ConditionLineMatches:
			if conditionMatch.lineno == global.conditions[conditionMatch.conditionID].lineRangeStart {
				conditionFulfilled = true
			}
		case ConditionRangeMatches:
			if conditionMatch.lineno >= global.conditions[conditionMatch.conditionID].lineRangeStart &&
				conditionMatch.lineno <= global.conditions[conditionMatch.conditionID].lineRangeEnd {
				conditionFulfilled = true
			}
		default:
			// ingore other condition types
			conditionFulfilled = !global.conditions[conditionMatch.conditionID].negated
		}
		if conditionFulfilled {
			if global.conditions[conditionMatch.conditionID].negated {
				result.matches = Matches{}
				return
			}
			conditionStatus[conditionMatch.conditionID] = true
		}
	}
	for i := range conditionStatus {
		if conditionStatus[i] != true && !global.conditions[i].negated {
			result.matches = Matches{}
			return
		}
	}

MatchLoop:
	// check for each match whether preceded/followed/surrounded conditions are fulfilled
	for matchIndex := 0; matchIndex < len(result.matches); {
		match := result.matches[matchIndex]
		lineno := match.lineno
		conditionStatus := make([]bool, len(global.conditions))
		for _, conditionMatch := range result.conditionMatches {
			conditionFulfilled := false
			maxAllowedDistance := global.conditions[conditionMatch.conditionID].within
			var actualDistance int64 = -1
			switch global.conditions[conditionMatch.conditionID].conditionType {
			case ConditionPreceded:
				actualDistance = lineno - conditionMatch.lineno
				if actualDistance == 0 {
					conditionFulfilled = conditionMatch.start < match.start
				} else {
					conditionFulfilled = (actualDistance >= 0) && (maxAllowedDistance == -1 || actualDistance <= maxAllowedDistance)
				}
			case ConditionFollowed:
				actualDistance = conditionMatch.lineno - lineno
				if actualDistance == 0 {
					conditionFulfilled = conditionMatch.start > match.start
				} else {
					conditionFulfilled = (actualDistance >= 0) && (maxAllowedDistance == -1 || actualDistance <= maxAllowedDistance)
				}
			case ConditionSurrounded:
				if lineno > conditionMatch.lineno {
					actualDistance = lineno - conditionMatch.lineno
				} else {
					actualDistance = conditionMatch.lineno - lineno
				}
				if actualDistance == 0 {
					conditionFulfilled = true
				} else {
					conditionFulfilled = (actualDistance >= 0) && (maxAllowedDistance == -1 || actualDistance <= maxAllowedDistance)
				}
			default:
				// ingore other condition types
				conditionFulfilled = !global.conditions[conditionMatch.conditionID].negated
			}
			if conditionFulfilled {
				if global.conditions[conditionMatch.conditionID].negated {
					goto ConditionFailed
				} else {
					conditionStatus[conditionMatch.conditionID] = true
				}
			}
		}
		for i := range conditionStatus {
			if conditionStatus[i] != true && !global.conditions[i].negated {
				goto ConditionFailed
			}
		}
		matchIndex++
		continue MatchLoop

	ConditionFailed:
		copy(result.matches[matchIndex:], result.matches[matchIndex+1:])
		result.matches = result.matches[0 : len(result.matches)-1]
	}
}

// getBeforeContextFromFile gets the context lines directly from the file.
// It is used when the context lines exceed the currently buffered data from the file.
func getBeforeContextFromFile(target string, offset int64, start int) *string {
	var contextBeforeStart int
	infile, _ := os.Open(target)
	seekPosition := offset + int64(start) - int64(InputBlockSize)
	if seekPosition < 0 {
		seekPosition = 0
	}
	count := InputBlockSize
	if offset == 0 && start < InputBlockSize {
		count = start
	}
	infile.Seek(seekPosition, 0)
	reader := bufio.NewReader(infile)
	buffer := make([]byte, count)
	reader.Read(buffer)

	lineStart := len(buffer)
	for lineStart > 0 && buffer[lineStart-1] != 0x0a {
		lineStart--
	}
	if lineStart > 0 {
		contextBeforeStart = lineStart - 1
		precedingLinesFound := 0
		for contextBeforeStart > 0 {
			if buffer[contextBeforeStart-1] == 0x0a {
				precedingLinesFound++
				if precedingLinesFound == options.ContextBefore {
					break
				}
			}
			contextBeforeStart--
		}
		tmp := string(buffer[contextBeforeStart : lineStart-1])
		return &tmp
	}
	return nil
}

// getAfterContextFromFile gets the context lines directly from the file.
// It is used when the context lines exceed the currently buffered data from the file.
func getAfterContextFromFile(target string, offset int64, end int) *string {
	var contextAfterEnd int
	infile, _ := os.Open(target)
	seekPosition := offset + int64(end)
	infile.Seek(seekPosition, 0)
	reader := bufio.NewReader(infile)
	buffer := make([]byte, InputBlockSize)
	length, _ := reader.Read(buffer)

	lineEnd := 0
	for lineEnd < length && buffer[lineEnd] != 0x0a {
		lineEnd++
	}
	if lineEnd < length-1 {
		contextAfterEnd = lineEnd
		followingLinesFound := 0
		for contextAfterEnd < length-1 {
			if buffer[contextAfterEnd+1] == 0x0a {
				followingLinesFound++
				if followingLinesFound == options.ContextAfter {
					contextAfterEnd++
					break
				}
			}
			contextAfterEnd++
		}
		if followingLinesFound < options.ContextAfter && contextAfterEnd == length-1 && buffer[length-1] != 0x0a {
			contextAfterEnd++
		}
		tmp := string(buffer[lineEnd+1 : contextAfterEnd])
		return &tmp
	}
	return nil
}

// processInvertMatchesReader is used to handle the '--invert' option.
// This function works line based and provides very limited support for options.
func processReaderInvertMatch(reader io.Reader, matchRegexes []*regexp.Regexp, target string) error {
	matches := make([]Match, 0, 16)
	var linecount int64
	var matchFound bool
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		linecount++
		matchFound = false
		for _, re := range global.matchRegexes {
			if re.MatchString(line) {
				matchFound = true
			}
		}
		if !matchFound {
			if options.FilesWithMatches || options.FilesWithoutMatch {
				global.resultsChan <- &Result{matches: []Match{Match{}}, target: target}
				return nil
			}
			m := Match{
				lineno: linecount,
				line:   line}
			matches = append(matches, m)

		}
	}
	result := &Result{matches: matches, target: target}
	global.resultsChan <- result
	return nil
}
