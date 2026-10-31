package models

import (
	"errors"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Limits on a pseud, from AO3's app/models/pseud.rb and config/config.yml.
const (
	PseudNameMin         = 1   // NAME_LENGTH_MIN
	PseudNameMax         = 40  // NAME_LENGTH_MAX
	PseudDescriptionMax  = 500 // DESCRIPTION_MAX
	PseudIconAltMax      = 250 // ICON_ALT_MAX
	PseudIconCommentMax  = 50  // ICON_COMMENT_MAX
	pseudIconURLMaxBytes = 500 // the column is VARCHAR(500)
)

// ReservedPseudName is the label shown for a creator of a work that hides its
// creators. No pseud may have it, or its owner could pass as a hidden creator.
const ReservedPseudName = "Anonymous Creator"

// IsReservedPseudName is true for ReservedPseudName ignoring case and spaces.
func IsReservedPseudName(name string) bool {
	squashed := strings.Join(strings.Fields(name), "")
	return strings.EqualFold(squashed, strings.Join(strings.Fields(ReservedPseudName), ""))
}

// ValidatePseudName applies AO3's rules to a pseud name: 1 to 40 characters,
// made of letters, numbers, marks, spaces, underscores and dashes in any
// script (Ruby's \p{Word}, space and dash), with at least one letter or
// number. The caller trims surrounding spaces first if it wants to.
func ValidatePseudName(name string) error {
	if IsReservedPseudName(name) {
		return errors.New("Name is reserved.")
	}
	n := utf8.RuneCountInString(name)
	if n < PseudNameMin {
		return errors.New("Name is too short (minimum is 1 character)")
	}
	if n > PseudNameMax {
		return errors.New("Name is too long (maximum is 40 characters)")
	}
	hasAlnum := false
	for _, r := range name {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			hasAlnum = true
		case unicode.IsMark(r), unicode.Is(unicode.Nl, r), unicode.Is(unicode.Pc, r), r == ' ', r == '-':
		default:
			return errors.New("Name can contain letters, numbers, spaces, underscores, and dashes.")
		}
	}
	if !hasAlnum {
		return errors.New("Name must contain at least one letter or number.")
	}
	return nil
}

// ValidatePseudText checks a pseud's description, icon alt text, icon comment
// text and icon address against AO3's limits. Empty values are fine.
func ValidatePseudText(description, iconAlt, iconComment, iconURL string) error {
	if utf8.RuneCountInString(description) > PseudDescriptionMax {
		return errors.New("Description must be less than 500 characters long.")
	}
	if utf8.RuneCountInString(iconAlt) > PseudIconAltMax {
		return errors.New("Icon alt text must be less than 250 characters long.")
	}
	if utf8.RuneCountInString(iconComment) > PseudIconCommentMax {
		return errors.New("Icon comment text must be less than 50 characters long.")
	}
	if iconURL != "" {
		u, err := url.Parse(iconURL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || len(iconURL) > pseudIconURLMaxBytes {
			return errors.New("Icon must be an http or https address of at most 500 characters.")
		}
	}
	return nil
}

// PseudSummary is how any pseud appears in the public API: its id and name,
// and nothing that leads to the account behind it.
type PseudSummary struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
