package gemax_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/ninedraft/gemax/gemax"
	"github.com/ninedraft/gemax/gemax/status"
)

func FuzzParseHeader(f *testing.F) {
	testcases := [][]byte{
		[]byte("10 " + gemax.MIMEGemtext + "\r\n"),
		[]byte("10 test\r\n"),
		[]byte("00000001212"),
		[]byte("99 esadas\r\n"),
	}
	for _, tc := range testcases {
		f.Add(tc) // Use f.Add to provide a seed corpus
	}
	f.Fuzz(func(t *testing.T, header []byte) {
		re := bytes.NewReader(header)
		code, meta, err := gemax.ParseResponseHeader(re)
		if err != nil {
			return
		}
		if n := len(header) - re.Len(); n > gemax.MaxHeaderSize+2 {
			t.Fatalf("parser must read at most %d bytes, byte %d bytes are read", gemax.MaxHeaderMetaSize, n)
		}
		if len(meta) > gemax.MaxHeaderMetaSize {
			t.Fatalf("meta size is too big: %d bytes", len(meta))
		}
		if code == status.Undefined {
			t.Fatalf("unexpected status code %s", status.Undefined)
		}

		reconstructedHeader := fmt.Sprintf("%d %s\r\n", code.Int(), meta)
		reparsedCode, reparsedMeta, errReparse := gemax.ParseResponseHeader(strings.NewReader(reconstructedHeader))
		if errReparse != nil {
			t.Fatalf("reparsing header %q: unexpected error: %v", reconstructedHeader, errReparse)
		}
		if reparsedCode != code || meta != reparsedMeta {
			t.Fatalf("expected(%s, %q)!=got(%s, %q)", code, meta, reparsedCode, reparsedMeta)
		}
	})
}
