package main

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func writeResponseWithRange(w http.ResponseWriter, r *http.Request, statusCode int, header http.Header, body []byte) {
	w.Header().Set("Accept-Ranges", "bytes")

	if r.Method == http.MethodHead {
		w.WriteHeader(statusCode)
		return
	}

	rangeHeader := r.Header.Get("Range")
	if rangeHeader == "" || statusCode != http.StatusOK || len(body) == 0 {
		w.WriteHeader(statusCode)
		_, _ = w.Write(body)
		return
	}

	start, end, ok := parseSingleByteRange(rangeHeader, int64(len(body)))
	if !ok {
		w.Header().Set("Content-Range", fmt.Sprintf("bytes */%d", len(body)))
		http.Error(w, "invalid range", http.StatusRequestedRangeNotSatisfiable)
		return
	}

	chunk := body[start : end+1]
	w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, len(body)))
	w.Header().Set("Content-Length", strconv.Itoa(len(chunk)))
	w.WriteHeader(http.StatusPartialContent)
	_, _ = w.Write(chunk)
}

func parseSingleByteRange(raw string, size int64) (start, end int64, ok bool) {
	if !strings.HasPrefix(raw, "bytes=") {
		return 0, 0, false
	}
	spec := strings.TrimSpace(strings.TrimPrefix(raw, "bytes="))
	if spec == "" || strings.Contains(spec, ",") {
		return 0, 0, false
	}

	parts := strings.SplitN(spec, "-", 2)
	if len(parts) != 2 {
		return 0, 0, false
	}

	startPart := strings.TrimSpace(parts[0])
	endPart := strings.TrimSpace(parts[1])

	switch {
	case startPart == "":
		// suffix-byte-range-spec: bytes=-N
		n, err := strconv.ParseInt(endPart, 10, 64)
		if err != nil || n <= 0 {
			return 0, 0, false
		}
		if n > size {
			n = size
		}
		return size - n, size - 1, true
	case endPart == "":
		s, err := strconv.ParseInt(startPart, 10, 64)
		if err != nil || s < 0 || s >= size {
			return 0, 0, false
		}
		return s, size - 1, true
	default:
		s, err1 := strconv.ParseInt(startPart, 10, 64)
		e, err2 := strconv.ParseInt(endPart, 10, 64)
		if err1 != nil || err2 != nil || s < 0 || e < s || s >= size {
			return 0, 0, false
		}
		if e >= size {
			e = size - 1
		}
		return s, e, true
	}
}
