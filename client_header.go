package gemax

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/ninedraft/gemax/internal/multierr"
	"github.com/ninedraft/gemax/status"
)

// MaxHeaderSize is used while parsing server responses.
const MaxHeaderSize = 1024

// ErrInvalidResponse means that server response is badly formed.
var ErrInvalidResponse = errors.New("malformed server response header")

// ErrHeaderTooLarge means that server response exceeds the MaxHeaderSize limit.
var ErrHeaderTooLarge = errors.New("header is too large: max header size is " + strconv.Itoa(MaxHeaderSize))

// ParseResponseHeader reads gemini header in form of "<code><SP><meta><CRLF>".
// If provided header is longer than MaxHeaderSize, than returns ErrHeaderTooLarge.
// Returns ErrInvalidResponse for badly formed server responses.
func ParseResponseHeader(re io.ByteReader) (status.Code, string, error) {
	var buf strings.Builder
	var ok bool
	for i := 0; i < MaxHeaderSize; i++ {
		var ru, errRune = re.ReadByte()
		if errRune != nil {
			return -1, "", multierr.Combine(ErrInvalidResponse, errRune)
		}
		if ru == '\n' {
			ok = true
			break
		}
		_ = buf.WriteByte(ru)
	}
	if !ok {
		return -1, "", ErrHeaderTooLarge
	}
	var line = strings.TrimRight(buf.String(), "\r")

	const codePrefixSize = 3
	if len(line) < codePrefixSize {
		return -1, "", fmt.Errorf("%w: header %q is too short: %d bytes", ErrInvalidResponse, line, len(line))
	}
	var code, errCode = strconv.Atoi(line[:codePrefixSize-1])
	if errCode != nil {
		var err = fmt.Errorf("parsing status code: %w", errCode)
		return -1, "", multierr.Combine(ErrInvalidResponse, err)
	}
	var meta = line[codePrefixSize:]
	return status.Code(code), meta, nil
}
