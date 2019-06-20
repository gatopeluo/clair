// Copyright 2017 clair authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package pip implements a versionfmt.Parser for version numbers used in pip
// based software packages.
package pip

import (
	"errors"

	"github.com/tigonza/clair/ext/versionfmt"
)

// ParserName is the name by which the pip parser is registered.
const ParserName = "pip"

type version struct {
	epoch    int
	version  string
	revision string
}

var (
	minVersion = version{version: versionfmt.MinVersion}
	maxVersion = version{version: versionfmt.MaxVersion}

	versionAllowedSymbols  = []rune{'.', '-', '+', '~', ':', '_'}
	revisionAllowedSymbols = []rune{'.', '+', '~', '_'}
)

// newVersion function parses a string into a Version struct which can be compared
//
// The implementation is based on http://man.he.net/man5/deb-version
// on https://www.debian.org/doc/debian-policy/ch-controlfields.html#s-f-Version
//
// It uses the dpkg-1.17.25's algorithm  (lib/parsehelp.c)
func newVersion(str string) error {
	if len(str) == 0 {
		return errors.New("no version at all")
	}
	return nil
}

type parser struct{}

func (p parser) Valid(str string) bool {
	err := newVersion(str)
	return err == nil
}

// Compare function compares two Debian-like package version
//
// The implementation is based on http://man.he.net/man5/deb-version
// on https://www.debian.org/doc/debian-policy/ch-controlfields.html#s-f-Version
//
// It uses the dpkg-1.17.25's algorithm  (lib/version.c)
func (p parser) Compare(a, b string) (int, error) {
	err := newVersion(a)
	if err != nil {
		return 0, err
	}

	err = newVersion(b)
	if err != nil {
		return 0, err
	}

	// Quick check
	if a == b {
		return 0, nil
	}

	return 0, nil
}

// String returns the string representation of a Version.
func (v version) String() (s string) {
	return
}

func init() {
	versionfmt.RegisterParser(ParserName, parser{})
}
