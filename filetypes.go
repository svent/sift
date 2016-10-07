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
	"regexp"
)

func init() {
	global.fileTypesMap = map[string]FileType{
		"go": FileType{
			Patterns: []string{"*.go"},
		},
		"cc": FileType{
			Patterns: []string{"*.c", "*.h", "*.xs"},
		},
		"cpp": FileType{
			Patterns: []string{"*.cpp", "*.cc", "*.cxx", "*.m", "*.hpp", "*.hh", "*.h", "*.hxx"},
		},
		"html": FileType{
			Patterns: []string{"*.htm", "*.html", "*.shtml", "*.xhtml"},
		},
		"groovy": FileType{
			Patterns: []string{"*.groovy", "*.gtmpl", "*.gpp", "*.grunit", "*.gradle"},
		},
		"java": FileType{
			Patterns: []string{"*.java", "*.properties"},
		},
		"jsp": FileType{
			Patterns: []string{"*.jsp", "*.jspx", "*.jhtm", "*.jhtml"},
		},
		"perl": FileType{
			Patterns:     []string{"*.pl", "*.pm", "*.pod", "*.t"},
			ShebangRegex: regexp.MustCompile(`^#!.*\bperl\b`),
		},
		"php": FileType{
			Patterns:     []string{"*.php", "*.phpt", "*.php3", "*.php4", "*.php5", "*.phtml"},
			ShebangRegex: regexp.MustCompile(`^#!.*\bphp\b`),
		},
		"ruby": FileType{
			Patterns:     []string{"*.rb", "*.rhtml", "*.rjs", "*.rxml", "*.erb", "*.rake", "*.spec", "Rakefile"},
			ShebangRegex: regexp.MustCompile(`^#!.*\bruby\b`),
		},
		"python": FileType{
			Patterns:     []string{"*.py", "*.pyw", "*.pyx", "SConstruct"},
			ShebangRegex: regexp.MustCompile(`^#!.*\bpython[0-9.]*\b`),
		},
		"shell": FileType{
			Patterns:     []string{"*.sh", "*.bash", "*.csh", "*.tcsh", "*.ksh", "*.zsh"},
			ShebangRegex: regexp.MustCompile(`^#!.*\b(?:ba|t?c|k|z)?sh\b`),
		},
		"xml": FileType{
			Patterns:     []string{"*.xml", "*.dtd", "*.xsl", "*.xslt", "*.ent"},
			ShebangRegex: regexp.MustCompile(`<\?xml`),
		},
	}
}
