package gemax

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/ninedraft/gemax/internal/multierr"
	"github.com/ninedraft/gemax/status"
)

const MaxHeaderSize = 1024

var errInvalidResponse = errors.New("invalid server response")

func ParseResponseHeader(re io.RuneReader) (status.Code, string, error) {
	var buf strings.Builder
	for i := 0; i < MaxHeaderSize; i++ {
		var ru, _, errRune = re.ReadRune()
		if errRune != nil {
			return -1, "", multierr.Combine(errInvalidResponse, errRune)
		}
		if ru == '\n' {
			break
		}
		if i == MaxHeaderSize {
			return -1, "", errInvalidResponse
		}
		buf.WriteRune(ru)
	}
	var line = strings.TrimRight(buf.String(), "\r")
	var code status.Code
	var meta string
	var _, errScan = fmt.Sscanf(line, "%d %s", &code, &meta)
	if errScan != nil {
		return -1, "", multierr.Combine(errInvalidResponse, errScan)
	}
	return code, meta, nil
}
