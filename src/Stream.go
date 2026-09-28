package src

import (
	"errors"
	"math"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

type StreamID struct {
	Milliseconds uint64
	Sequence     uint64
}

func (id StreamID) String() string {
	return strconv.FormatUint(id.Milliseconds, 10) + "-" + strconv.FormatUint(id.Sequence, 10)
}

func (id StreamID) compare(other StreamID) int {
	if id.Milliseconds < other.Milliseconds || (id.Milliseconds == other.Milliseconds && id.Sequence < other.Sequence) {
		return -1
	}
	if id == other {
		return 0
	}
	return 1
}

type Stream struct {
	Name    string
	LastID  StreamID
	Entries []StreamEntry
}

type StreamEntry struct {
	ID     StreamID
	Fields []string
}

type StreamReadResult struct {
	Key     string
	Entries []StreamEntry
}

// streamMu protects the map, entries, IDs, and change signal. CommandMu also
// serializes client commands with AOF recording.
var (
	GlobalStream  = make(map[string]*Stream)
	streamMu      sync.RWMutex
	streamChanged = make(chan struct{})
)

var errStreamID = errors.New("ERR Invalid stream ID specified as stream command argument")
var errStreamSyntax = errors.New("ERR syntax error")
var errStreamInteger = errors.New("ERR value is not an integer or out of range")
var errStreamWrongType = errors.New("WRONGTYPE Operation against a key holding the wrong kind of value")

func parseStreamID(value string) (StreamID, error) {
	parts := strings.Split(value, "-")
	if len(parts) > 2 || parts[0] == "" {
		return StreamID{}, errStreamID
	}
	ms, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return StreamID{}, errStreamID
	}
	var seq uint64
	if len(parts) == 2 {
		if parts[1] == "" {
			return StreamID{}, errStreamID
		}
		seq, err = strconv.ParseUint(parts[1], 10, 64)
		if err != nil {
			return StreamID{}, errStreamID
		}
	}
	return StreamID{Milliseconds: ms, Sequence: seq}, nil
}

type streamBound struct {
	id        StreamID
	exclusive bool
}

func parseStreamBound(value string, upper bool) (streamBound, error) {
	if value == "-" {
		return streamBound{}, nil
	}
	if value == "+" {
		return streamBound{id: StreamID{math.MaxUint64, math.MaxUint64}}, nil
	}
	bound := streamBound{}
	if strings.HasPrefix(value, "(") {
		bound.exclusive = true
		value = value[1:]
	}
	id, err := parseStreamID(value)
	if err != nil {
		return streamBound{}, err
	}
	if upper && !bound.exclusive && !strings.Contains(value, "-") {
		id.Sequence = math.MaxUint64
	}
	bound.id = id
	return bound, nil
}

type streamTrim struct {
	strategy string
	maxLen   uint64
	minID    StreamID
	limit    uint64
	hasLimit bool
}

func parseStreamTrim(msg []string, index int) (streamTrim, int, error) {
	if index >= len(msg) {
		return streamTrim{}, index, errStreamSyntax
	}
	trim := streamTrim{strategy: strings.ToUpper(msg[index])}
	if trim.strategy != "MAXLEN" && trim.strategy != "MINID" {
		return streamTrim{}, index, errStreamSyntax
	}
	index++
	approximate := false
	if index < len(msg) && (msg[index] == "=" || msg[index] == "~") {
		approximate = msg[index] == "~"
		index++
	}
	if index >= len(msg) {
		return streamTrim{}, index, errStreamSyntax
	}
	if trim.strategy == "MAXLEN" {
		value, err := strconv.ParseUint(msg[index], 10, 64)
		if err != nil {
			return streamTrim{}, index, errStreamInteger
		}
		trim.maxLen = value
	} else {
		id, err := parseStreamID(msg[index])
		if err != nil {
			return streamTrim{}, index, err
		}
		trim.minID = id
	}
	index++
	if index < len(msg) && strings.EqualFold(msg[index], "LIMIT") {
		if !approximate || index+1 >= len(msg) {
			return streamTrim{}, index, errStreamSyntax
		}
		value, err := strconv.ParseUint(msg[index+1], 10, 64)
		if err != nil {
			return streamTrim{}, index, errStreamInteger
		}
		trim.limit, trim.hasLimit = value, true
		index += 2
	}
	return trim, index, nil
}

func trimStream(stream *Stream, trim streamTrim) int {
	remove := 0
	if trim.strategy == "MAXLEN" {
		if uint64(len(stream.Entries)) > trim.maxLen {
			remove = len(stream.Entries) - int(trim.maxLen)
		}
	} else {
		for remove < len(stream.Entries) && stream.Entries[remove].ID.compare(trim.minID) < 0 {
			remove++
		}
	}
	if trim.hasLimit && trim.limit != 0 && uint64(remove) > trim.limit {
		remove = int(trim.limit)
	}
	if remove > 0 {
		stream.Entries = slices.Delete(stream.Entries, 0, remove)
	}
	return remove
}

// XADD returns the entry ID and a command with that explicit ID for AOF replay.
// A nil record means NOMKSTREAM skipped a missing stream.
func XADD(msg []string) (string, []string, error) {
	if len(msg) < 5 {
		return "", nil, errStreamSyntax
	}
	key, i := msg[1], 2
	nomkstream := false
	if strings.EqualFold(msg[i], "NOMKSTREAM") {
		nomkstream = true
		i++
	}
	// Reference options have no effect without consumer groups.
	if i < len(msg) && (strings.EqualFold(msg[i], "KEEPREF") || strings.EqualFold(msg[i], "DELREF") || strings.EqualFold(msg[i], "ACKED")) {
		i++
	}
	trimStart := i
	var trim *streamTrim
	if i < len(msg) && (strings.EqualFold(msg[i], "MAXLEN") || strings.EqualFold(msg[i], "MINID")) {
		parsed, next, err := parseStreamTrim(msg, i)
		if err != nil {
			return "", nil, err
		}
		trim, i = &parsed, next
	}
	if i >= len(msg) || len(msg)-i < 3 || (len(msg)-i-1)%2 != 0 {
		return "", nil, errStreamSyntax
	}
	idToken, fields := msg[i], msg[i+1:]
	if idToken != "*" && !strings.HasSuffix(idToken, "-*") {
		if _, err := parseStreamID(idToken); err != nil {
			return "", nil, err
		}
	}
	if kind := KeyType(key); kind != "none" && kind != "stream" {
		return "", nil, errStreamWrongType
	}

	streamMu.Lock()
	defer streamMu.Unlock()
	stream := GlobalStream[key]
	if stream == nil {
		if nomkstream {
			return "", nil, nil
		}
		stream = &Stream{Name: key}
	}
	var id StreamID
	if idToken == "*" || strings.HasSuffix(idToken, "-*") {
		ms := uint64(time.Now().UnixMilli())
		if idToken != "*" {
			parsed, err := strconv.ParseUint(strings.TrimSuffix(idToken, "-*"), 10, 64)
			if err != nil {
				return "", nil, errStreamID
			}
			ms = parsed
		}
		if ms < stream.LastID.Milliseconds {
			if idToken != "*" {
				return "", nil, errors.New("ERR The ID specified in XADD is equal or smaller than the target stream top item")
			}
			ms = stream.LastID.Milliseconds
		}
		id.Milliseconds = ms
		if ms == stream.LastID.Milliseconds {
			if stream.LastID.Sequence == math.MaxUint64 {
				return "", nil, errStreamID
			}
			id.Sequence = stream.LastID.Sequence + 1
		}
		if id == (StreamID{}) {
			id.Sequence = 1
		}
	} else {
		id, _ = parseStreamID(idToken)
	}
	if id == (StreamID{}) {
		return "", nil, errors.New("ERR The ID specified in XADD must be greater than 0-0")
	}
	if id.compare(stream.LastID) <= 0 {
		return "", nil, errors.New("ERR The ID specified in XADD is equal or smaller than the target stream top item")
	}
	stream.Entries = append(stream.Entries, StreamEntry{ID: id, Fields: slices.Clone(fields)})
	stream.LastID = id
	GlobalStream[key] = stream
	if trim != nil {
		trimStream(stream, *trim)
	}
	close(streamChanged)
	streamChanged = make(chan struct{})
	record := []string{"XADD", key}
	if trim != nil {
		record = append(record, msg[trimStart:i]...)
	}
	record = append(record, id.String())
	record = append(record, fields...)
	return id.String(), record, nil
}

func XLEN(msg []string) (int, error) {
	if len(msg) != 2 {
		return 0, errStreamSyntax
	}
	streamMu.RLock()
	defer streamMu.RUnlock()
	if stream := GlobalStream[msg[1]]; stream != nil {
		return len(stream.Entries), nil
	}
	return 0, nil
}

func cloneStreamEntry(entry StreamEntry) StreamEntry {
	return StreamEntry{ID: entry.ID, Fields: slices.Clone(entry.Fields)}
}

func streamRange(msg []string, reverse bool) ([]StreamEntry, error) {
	if len(msg) != 4 && len(msg) != 6 {
		return nil, errStreamSyntax
	}
	count := math.MaxInt
	if len(msg) == 6 {
		if !strings.EqualFold(msg[4], "COUNT") {
			return nil, errStreamSyntax
		}
		value, err := strconv.ParseInt(msg[5], 10, 64)
		if err != nil || value < 0 {
			return nil, errStreamInteger
		}
		if value < int64(count) {
			count = int(value)
		}
	}
	lowerToken, upperToken := msg[2], msg[3]
	if reverse {
		lowerToken, upperToken = msg[3], msg[2]
	}
	lower, err := parseStreamBound(lowerToken, false)
	if err != nil {
		return nil, err
	}
	upper, err := parseStreamBound(upperToken, true)
	if err != nil {
		return nil, err
	}
	result := []StreamEntry{}
	streamMu.RLock()
	defer streamMu.RUnlock()
	stream := GlobalStream[msg[1]]
	if stream == nil || count == 0 {
		return result, nil
	}
	visit := func(entry StreamEntry) bool {
		low, high := entry.ID.compare(lower.id), entry.ID.compare(upper.id)
		if low < 0 || (low == 0 && lower.exclusive) || high > 0 || (high == 0 && upper.exclusive) {
			return false
		}
		result = append(result, cloneStreamEntry(entry))
		return len(result) >= count
	}
	if reverse {
		for i := len(stream.Entries) - 1; i >= 0; i-- {
			if visit(stream.Entries[i]) {
				break
			}
		}
	} else {
		for _, entry := range stream.Entries {
			if visit(entry) {
				break
			}
		}
	}
	return result, nil
}

func XRANGE(msg []string) ([]StreamEntry, error)    { return streamRange(msg, false) }
func XREVRANGE(msg []string) ([]StreamEntry, error) { return streamRange(msg, true) }

type XReadSpec struct {
	Keys     []string
	IDs      []StreamID
	Latest   []bool
	Newest   []bool
	Count    int
	Block    time.Duration
	HasBlock bool
}

func ParseXREAD(msg []string) (XReadSpec, error) {
	if len(msg) < 4 {
		return XReadSpec{}, errStreamSyntax
	}
	spec := XReadSpec{Count: math.MaxInt}
	i := 1
	for i < len(msg) && !strings.EqualFold(msg[i], "STREAMS") {
		if i+1 >= len(msg) {
			return XReadSpec{}, errStreamSyntax
		}
		switch strings.ToUpper(msg[i]) {
		case "COUNT":
			value, err := strconv.ParseInt(msg[i+1], 10, 64)
			if err != nil || value <= 0 {
				return XReadSpec{}, errStreamInteger
			}
			if value < int64(spec.Count) {
				spec.Count = int(value)
			}
		case "BLOCK":
			value, err := strconv.ParseInt(msg[i+1], 10, 64)
			if err != nil || value < 0 || value > math.MaxInt64/int64(time.Millisecond) {
				return XReadSpec{}, errStreamInteger
			}
			spec.Block, spec.HasBlock = time.Duration(value)*time.Millisecond, true
		default:
			return XReadSpec{}, errStreamSyntax
		}
		i += 2
	}
	if i >= len(msg) || i+1 >= len(msg) || (len(msg)-i-1)%2 != 0 {
		return XReadSpec{}, errStreamSyntax
	}
	count := (len(msg) - i - 1) / 2
	spec.Keys = slices.Clone(msg[i+1 : i+1+count])
	spec.IDs = make([]StreamID, count)
	spec.Latest = make([]bool, count)
	spec.Newest = make([]bool, count)
	for j, value := range msg[i+1+count:] {
		if value == "$" {
			spec.Latest[j] = true
			continue
		}
		if value == "+" {
			spec.Newest[j] = true
			continue
		}
		id, err := parseStreamID(value)
		if err != nil {
			return XReadSpec{}, err
		}
		spec.IDs[j] = id
	}
	return spec, nil
}

// Resolve $ only once, including before a blocking wait.
func ResolveXREAD(spec *XReadSpec) {
	streamMu.RLock()
	defer streamMu.RUnlock()
	for i, latest := range spec.Latest {
		if latest {
			if stream := GlobalStream[spec.Keys[i]]; stream != nil {
				spec.IDs[i] = stream.LastID
			}
			spec.Latest[i] = false
		}
	}
}

func XREAD(spec XReadSpec) []StreamReadResult {
	result := []StreamReadResult{}
	streamMu.RLock()
	defer streamMu.RUnlock()
	for i, key := range spec.Keys {
		stream := GlobalStream[key]
		if stream == nil {
			continue
		}
		if spec.Newest[i] {
			if len(stream.Entries) != 0 {
				result = append(result, StreamReadResult{Key: key, Entries: []StreamEntry{cloneStreamEntry(stream.Entries[len(stream.Entries)-1])}})
			}
			continue
		}
		entries := []StreamEntry{}
		for _, entry := range stream.Entries {
			if entry.ID.compare(spec.IDs[i]) > 0 {
				entries = append(entries, cloneStreamEntry(entry))
				if len(entries) >= spec.Count {
					break
				}
			}
		}
		if len(entries) != 0 {
			result = append(result, StreamReadResult{Key: key, Entries: entries})
		}
	}
	return result
}

func StreamSignal() <-chan struct{} {
	streamMu.RLock()
	defer streamMu.RUnlock()
	return streamChanged
}

func StreamLastID(key string) (StreamID, bool) {
	streamMu.RLock()
	defer streamMu.RUnlock()
	stream := GlobalStream[key]
	if stream == nil {
		return StreamID{}, false
	}
	return stream.LastID, true
}

func XDEL(msg []string) (int, error) {
	if len(msg) < 3 {
		return 0, errStreamSyntax
	}
	ids := make(map[StreamID]struct{}, len(msg)-2)
	for _, token := range msg[2:] {
		id, err := parseStreamID(token)
		if err != nil {
			return 0, err
		}
		ids[id] = struct{}{}
	}
	streamMu.Lock()
	defer streamMu.Unlock()
	stream := GlobalStream[msg[1]]
	if stream == nil {
		return 0, nil
	}
	removed := 0
	for i := 0; i < len(stream.Entries); {
		if _, ok := ids[stream.Entries[i].ID]; ok {
			stream.Entries = slices.Delete(stream.Entries, i, i+1)
			removed++
		} else {
			i++
		}
	}
	return removed, nil
}

func XTRIM(msg []string) (int, error) {
	if len(msg) < 4 {
		return 0, errStreamSyntax
	}
	trim, next, err := parseStreamTrim(msg, 2)
	if err != nil {
		return 0, err
	}
	if next != len(msg) {
		return 0, errStreamSyntax
	}
	streamMu.Lock()
	defer streamMu.Unlock()
	if stream := GlobalStream[msg[1]]; stream != nil {
		return trimStream(stream, trim), nil
	}
	return 0, nil
}

// ValidateStreamRecord checks an AOF record before any transaction is applied.
// XADD records must contain the generated, explicit ID rather than *.
func ValidateStreamRecord(msg []string) error {
	if len(msg) == 0 {
		return errStreamSyntax
	}
	switch strings.ToUpper(msg[0]) {
	case "XADD":
		id, err := StreamRecordID(msg)
		if err != nil {
			return err
		}
		if id == (StreamID{}) {
			return errStreamID
		}
	case "XDEL":
		if len(msg) < 3 {
			return errStreamSyntax
		}
		for _, token := range msg[2:] {
			if _, err := parseStreamID(token); err != nil {
				return err
			}
		}
	case "XTRIM":
		if len(msg) < 4 {
			return errStreamSyntax
		}
		_, next, err := parseStreamTrim(msg, 2)
		if err != nil {
			return err
		}
		if next != len(msg) {
			return errStreamSyntax
		}
	default:
		return errStreamSyntax
	}
	return nil
}

// StreamRecordID reads the explicit ID from a validated XADD AOF record.
func StreamRecordID(msg []string) (StreamID, error) {
	if len(msg) < 5 || !strings.EqualFold(msg[0], "XADD") {
		return StreamID{}, errStreamSyntax
	}
	i := 2
	if strings.EqualFold(msg[i], "MAXLEN") || strings.EqualFold(msg[i], "MINID") {
		_, next, err := parseStreamTrim(msg, i)
		if err != nil {
			return StreamID{}, err
		}
		i = next
	}
	if len(msg)-i < 3 || (len(msg)-i-1)%2 != 0 {
		return StreamID{}, errStreamSyntax
	}
	return parseStreamID(msg[i])
}
