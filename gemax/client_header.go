package gemax

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"

	"github.com/ninedraft/gemax/gemax/internal/multierr"
	"github.com/ninedraft/gemax/gemax/status"
)

// MaxHeaderMetaSize is used while parsing server responses.
const MaxHeaderMetaSize = 1024

const MaxHeaderSize = 3 + MaxHeaderMetaSize

// ErrInvalidResponse means that server response is badly formed.
var ErrInvalidResponse = errors.New("malformed server response header")

// ErrHeaderTooLarge means that server response exceeds the MaxHeaderSize limit.
var ErrHeaderTooLarge = errors.New("header is too large: max header size is " + strconv.Itoa(MaxHeaderMetaSize))

// ParseResponseHeader reads gemini header in form of "<code><SP><meta><CRLF>".
// If provided header is longer than MaxHeaderSize, than returns ErrHeaderTooLarge.
// Returns ErrInvalidResponse for badly formed server responses.
func ParseResponseHeader(re io.RuneReader) (status.Code, string, error) {
	code, errCode := parseStatusCode(re)
	if errCode != nil {
		return status.Undefined, "", errCode
	}

	if err := expectStreamRune(re, ' '); err != nil {
		return status.Undefined, "", err
	}

	var meta strings.Builder
readLoop:
	for meta.Len() <= MaxHeaderMetaSize {
		var ru, size, errRune = re.ReadRune()
		if errRune != nil {
			return status.Undefined, "", multierr.Combine(ErrInvalidResponse, errRune)
		}
		switch {
		case ru == '\r':
			break readLoop
		case ru != '\r' && ru != '\n' && meta.Len()+size <= MaxHeaderMetaSize:
			_, _ = meta.WriteRune(ru)
		default:
			return status.Undefined, "", fmt.Errorf("%w: unexpected byte %q", ErrInvalidResponse, ru)
		}
	}

	errNL := expectStreamRune(re, '\n')
	switch {
	case errNL != nil && meta.Len() == MaxHeaderMetaSize:
		return status.Undefined, "", ErrHeaderTooLarge
	case errNL != nil:
		return status.Undefined, "", errNL
	}

	return code, meta.String(), nil
}

func expectStreamRune(re io.RuneReader, expect rune) error {
	b, _, err := re.ReadRune()
	if err != nil {
		return err
	}
	if b != expect {
		return fmt.Errorf("an %q is expected, got %q", expect, b)
	}
	return nil
}

func parseStatusCode(re io.RuneReader) (status.Code, error) {
	high, _, errHigh := re.ReadRune()
	if errHigh != nil {
		return status.Undefined, errHigh
	}
	low, _, errLow := re.ReadRune()
	if errLow != nil {
		return status.Undefined, errLow
	}
	if !unicode.IsDigit(high) || !unicode.IsDigit(low) {
		return status.Undefined, fmt.Errorf("%w: unexpected status string: %q", ErrInvalidResponse, []rune{high, low})
	}
	key := statusCodeKey{
		high: byte(high) - '0',
		low:  byte(low) - '0',
	}
	code, ok := statusCodes[key]
	if !ok {
		return status.Undefined, fmt.Errorf("%w: unexpected status %d", ErrInvalidResponse, key.int())
	}
	return code, nil
}

var statusCodes map[statusCodeKey]status.Code

type statusCodeKey struct {
	high, low byte
}

func (key statusCodeKey) int() int {
	return 10*int(key.high+'0') + int(key.low+'0')
}

func init() {
	allCodes := status.AllCodes()
	statusCodes = make(map[statusCodeKey]status.Code, len(allCodes))
	for _, code := range allCodes {
		c := code.Int()
		key := statusCodeKey{
			high: byte(c / 10),
			low:  byte(c % 10),
		}
		statusCodes[key] = code
	}
}
