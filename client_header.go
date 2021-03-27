package gemax

import (
	"errors"
	"io"
	"strconv"
	"strings"

	"github.com/ninedraft/gemax/internal/multierr"
	"github.com/ninedraft/gemax/status"
)

const MaxHeaderSize = 1024

var errInvalidResponse = errors.New("invalid server response")

func ParseResponseHeader(re io.ByteReader) (status.Code, string, error) {
	var buf strings.Builder
	var ok bool
	for i := 0; i < MaxHeaderSize; i++ {
		var ru, errRune = re.ReadByte()
		if errRune != nil {
			return -1, "", multierr.Combine(errInvalidResponse, errRune)
		}
		if ru == '\n' {
			ok = true
			break
		}
		_ = buf.WriteByte(ru)
	}
	if !ok {
		return -1, "", errInvalidResponse
	}
	var line = strings.TrimRight(buf.String(), "\r")

	const codePrefixSize = 3
	if len(line) < codePrefixSize {
		return -1, "", errInvalidResponse
	}
	var code, errCode = strconv.Atoi(line[:codePrefixSize])
	if errCode != nil {
		return -1, "", errInvalidResponse
	}
	var meta = line[codePrefixSize:]
	return status.Code(code), meta, nil
}
