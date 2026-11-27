package hushscript

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/Hushscript/hushscript-sdk-go/internal/probe"
)

const (
	maxFileBytes       = 5_000_000_000
	minDurationSeconds = 15
	maxDurationSeconds = 36_000
	maxKeyterms        = 1000
	maxKeytermWords    = 6
	maxKeytermChars    = 128
	maxKeytermTotal    = 1000
	maxPromptChars     = 2000
	maxTranslateTo     = 15
	maxMatterCodeChars = 64
	minKeyLength       = 16
	maxKeyLength       = 128
)

func formatInt(v int64) string { return strconv.FormatInt(v, 10) }

func invalidOption(red redactor, message string, cause error) *Error {
	return newSDKError(KindPreflight, "invalid_option", message, cause, red, hintInputs{})
}

// safeSegment reports whether id can stand as one path segment of a request: "", "." and ".." would
// collapse into another route (GET /v1/jobs/.. is GET /v1), and bytes that are not UTF-8 have no
// agreed percent-encoding. Everything else is percent-encoded by the generated client.
func safeSegment(id string) bool {
	return id != "" && id != "." && id != ".." && utf8.ValidString(id)
}

// checkID is the invalid_option error for an id that cannot be a path segment, raised before any
// request. name is the option as the caller knows it.
func checkID(red redactor, name, id string) error {
	if safeSegment(id) {
		return nil
	}
	return invalidOption(red, name+` must be a non-empty id that is not "." or "..", in valid UTF-8.`, nil)
}

// newIdempotencyKey is a UUID v4 string: 36 characters, valid for uploads (16 to 128 bytes of 0x21 to 0x7E)
// and purchases alike (spec 7.3).
func newIdempotencyKey() string {
	var b [16]byte
	_, _ = rand.Read(b[:]) // never fails: crypto/rand crashes the program instead (Go 1.24+)
	b[6] &= 0x0f
	b[6] |= 0x40 // version 4
	b[8] &= 0x3f
	b[8] |= 0x80 // RFC 9562 variant
	h := hex.EncodeToString(b[:])
	return h[:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:]
}

// runtimeOptions validates what every call takes: concurrency and wait timeout. Zero means "unset" for
// each; anything else outside its range is invalid_option, never clamped. On an error the values are
// zero: the bare returns leave them unset.
func runtimeOptions(red redactor, requested int, waitTimeout time.Duration) (concurrency int, timeout time.Duration, err error) {
	if requested < 0 || requested > maxConcurrency {
		err = invalidOption(red, "Concurrency must be a whole number from 1 to 8.", nil)
		return
	}
	if timeout, err = waitTimeoutOf(red, waitTimeout); err != nil {
		return
	}
	concurrency = requested
	if concurrency == 0 {
		concurrency = defaultConcurrency
	}
	return
}

// checkDuration validates a duration override; zero means "unset".
func checkDuration(red redactor, duration float64) error {
	if duration < 0 || duration > math.MaxFloat64 || math.IsNaN(duration) {
		return invalidOption(red, "DurationSeconds must be a finite number of seconds, above 0.", nil)
	}
	return nil
}

// waitTimeoutOf is the wait timeout to use: zero means the default. On an error it is zero.
func waitTimeoutOf(red redactor, waitTimeout time.Duration) (timeout time.Duration, err error) {
	if waitTimeout < 0 {
		err = invalidOption(red, "WaitTimeout must be a positive duration.", nil)
		return
	}
	timeout = waitTimeout
	if timeout == 0 {
		timeout = defaultWaitTimeout
	}
	return
}

// checkKey is spec 7.3: an upload key is 16 to 128 bytes of 0x21 to 0x7E (printable ASCII, no space).
func checkKey(red redactor, key string) error {
	ok := len(key) >= minKeyLength && len(key) <= maxKeyLength
	for i := 0; ok && i < len(key); i++ {
		ok = key[i] >= 0x21 && key[i] <= 0x7e
	}
	if !ok {
		return invalidOption(red, "IdempotencyKey must be 16 to 128 printable ASCII characters, without spaces.", nil)
	}
	return nil
}

// medicalLanguages mirrors the backend's MEDICAL_LANGUAGES, compared after the backend's own normalization.
var medicalLanguages = []string{"en", "en_au", "en_uk", "en_us", "es", "fr", "de", "de_ch"}

// vetLanguages mirrors the backend's VET_LANGUAGES, compared after the backend's own normalization.
var vetLanguages = []string{"en", "en_au", "en_uk", "en_us", "es", "fr", "de", "de_ch"}

// stt37Languages mirrors the API's STT37_LANGUAGES: the languages that apply key terms and a prompt.
var stt37Languages = []string{
	"af", "ar", "ca", "da", "de", "de_ch", "en", "en_au", "en_uk", "en_us",
	"es", "et", "fa", "fi", "fr", "gl", "he", "hi", "it", "ja",
	"ko", "mr", "nl", "nn", "no", "pt", "ro", "ru", "sv", "tr",
	"ur", "vi", "zh",
}

// universal2Languages is the rest of the API's source languages; the server names a code in neither list itself.
var universal2Languages = []string{
	"am", "as", "az", "ba", "be", "bg", "bn", "bo", "br", "bs",
	"cs", "cy", "el", "eu", "fo", "gu", "ha", "haw", "hr", "ht",
	"hu", "hy", "id", "is", "jw", "ka", "kk", "km", "kn", "la",
	"lb", "ln", "lo", "lt", "lv", "mg", "mi", "mk", "ml", "mn",
	"ms", "mt", "my", "ne", "oc", "pa", "pl", "ps", "sa", "sd",
	"si", "sk", "sl", "sn", "so", "sq", "sr", "su", "sw", "ta",
	"te", "tg", "th", "tk", "tl", "tt", "uk", "uz", "yi", "yo",
}

func checkOptions(red redactor, opts *TranscribeOptions) error {
	if opts.Medical && opts.Vet {
		return invalidOption(red, "Medical and Vet cannot be combined.", nil)
	}
	if opts.Medical && !slices.Contains(medicalLanguages, sourceLanguage(opts)) {
		return invalidOption(red, "Medical needs Language to be one of en, en_au, en_uk, en_us, es, fr, de or de_ch.", nil)
	}
	if opts.Vet && !slices.Contains(vetLanguages, sourceLanguage(opts)) {
		return invalidOption(red, "Vet needs Language to be one of en, en_au, en_uk, en_us, es, fr, de or de_ch.", nil)
	}
	if err := checkContext(red, opts); err != nil {
		return err
	}
	if distinctTargets(opts.TranslateTo) > maxTranslateTo {
		return invalidOption(red, "TranslateTo takes at most 15 distinct languages.", nil)
	}
	if len([]rune(opts.MatterCode)) > maxMatterCodeChars {
		return invalidOption(red, "MatterCode is at most 64 characters.", nil)
	}
	return nil
}

// distinctTargets is how many translation targets the backend keeps: each trimmed (Unicode white space)
// and lowercased (ASCII), blank ones dropped, repeated ones counted once.
func distinctTargets(targets []string) int {
	var seen map[string]struct{}
	for _, raw := range targets {
		target := strings.TrimFunc(raw, isContextSpace)
		if target == "" {
			continue
		}
		if seen == nil {
			seen = make(map[string]struct{}, len(targets))
		}
		seen[asciiLower(target)] = struct{}{}
	}
	return len(seen)
}

// asciiLower lowercases A to Z only, as the backend's to_ascii_lowercase does.
func asciiLower(s string) string {
	return strings.Map(func(r rune) rune {
		if r >= 'A' && r <= 'Z' {
			return r + 'a' - 'A'
		}
		return r
	}, s)
}

// sourceLanguage is Language as the backend reads it: trimmed (Unicode white space) and lowercased (ASCII).
func sourceLanguage(opts *TranscribeOptions) string {
	return asciiLower(strings.TrimFunc(opts.Language, isContextSpace))
}

// isContextSpace is Unicode White_Space, the set Rust's str::trim and split_whitespace use on the
// backend: U+0009 to U+000D, U+0020, U+0085, U+00A0, U+1680, U+2000 to U+200A, U+2028, U+2029,
// U+202F, U+205F and U+3000. unicode.IsSpace is that same set; U+FEFF and U+180E are in neither.
// The predicate is named so a test can pin the equivalence rune by rune.
func isContextSpace(r rune) bool { return unicode.IsSpace(r) }

// contextOptions is the key terms and prompt as the backend sees them after its own normalization:
// every term and the prompt trimmed, blank terms dropped, repeated terms kept once. origin[i] is the
// index in the caller's Keyterms of terms[i].
type contextOptions struct {
	terms  []string
	origin []int
	prompt string
}

func normalizeContext(opts *TranscribeOptions) contextOptions {
	var out contextOptions
	var seen map[string]struct{}
	for i, raw := range opts.Keyterms {
		term := strings.TrimFunc(raw, isContextSpace)
		if term == "" {
			continue
		}
		if _, dup := seen[term]; dup {
			continue
		}
		if seen == nil {
			seen = make(map[string]struct{}, len(opts.Keyterms))
		}
		seen[term] = struct{}{}
		out.terms = append(out.terms, term)
		out.origin = append(out.origin, i)
	}
	out.prompt = strings.TrimFunc(opts.Prompt, isContextSpace)
	return out
}

// checkContextLanguage refuses key terms or a prompt in a language the API does not apply them in. The
// language is trimmed and lowercased first, as the backend does; auto-detect and a code in neither list
// pass.
func checkContextLanguage(red redactor, opts *TranscribeOptions, normalized contextOptions) error {
	language := sourceLanguage(opts)
	if !slices.Contains(universal2Languages, language) {
		return nil
	}
	var option string
	switch {
	case len(normalized.terms) > 0:
		option = "keyterms"
	case normalized.prompt != "":
		option = "prompt"
	default:
		return nil
	}
	return invalidOption(red, option+` does not work for language "`+language+`"; it needs language to be one of `+strings.Join(stt37Languages, ", ")+`.`, nil)
}

// checkContext applies the backend's key term and prompt limits to what normalizeContext leaves, so a
// violation fails before any request. Saved dictionaries and prompt presets are merged server-side
// and cannot be seen here; the server stays the authority on them.
func checkContext(red redactor, opts *TranscribeOptions) error {
	normalized := normalizeContext(opts)
	if err := checkContextLanguage(red, opts, normalized); err != nil {
		return err
	}
	for i, term := range normalized.terms {
		index := normalized.origin[i]
		if len(strings.FieldsFunc(term, isContextSpace)) > maxKeytermWords {
			return invalidOption(red, "Keyterms["+strconv.Itoa(index)+"] has more than 6 words.", nil)
		}
		if utf8.RuneCountInString(term) > maxKeytermChars {
			return invalidOption(red, "Keyterms["+strconv.Itoa(index)+"] is longer than 128 characters.", nil)
		}
	}
	if len(normalized.terms) > maxKeyterms {
		return invalidOption(red, "Keyterms takes at most 1000 unique terms.", nil)
	}
	words := 0
	for _, term := range normalized.terms {
		words += len(strings.FieldsFunc(term, isContextSpace))
	}
	if words > maxKeytermTotal {
		return invalidOption(red, "Keyterms take at most 1000 words in total ("+strconv.Itoa(words)+").", nil)
	}
	if utf8.RuneCountInString(normalized.prompt) > maxPromptChars {
		return invalidOption(red, "Prompt is longer than 2000 characters.", nil)
	}
	return nil
}

// annotator adds what a caller needs to the errors of one call: the idempotency key once the create
// was sent, the resume token once the job id is known (spec 7.3), the SDK's own abort error for a
// bare context error, and, once a request that carries a key or token was sent, unexpected_error for
// any other failure that is not an SDK error (spec 9.2).
type annotator struct {
	core  *core
	key   string
	token string
}

func (a *annotator) annotate(ctx context.Context, err error) error {
	failure, ok := err.(*Error) //nolint:errorlint // the pipeline returns its own errors unwrapped
	if !ok {
		switch {
		case ctx.Err() != nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded):
			failure = a.core.abortError(err)
		case a.key != "" || a.token != "":
			// With a resume token the create was answered, so its request was sent; with only the key
			// the create may or may not have reached the server.
			failure = a.core.unexpectedError(err, a.token != "")
		default:
			failure = a.core.requestNotBuilt(err)
		}
	}
	if failure.IdempotencyKey == "" {
		failure.IdempotencyKey = a.key
	}
	if failure.ResumeToken == "" {
		failure.ResumeToken = a.token
	}
	return failure
}

// requestNotBuilt is the error for a request the SDK could not even build, which only a base URL or a
// path that cannot form a URL causes.
func (c *core) requestNotBuilt(cause error) *Error {
	return newSDKError(KindConfiguration, "config_invalid", "The request could not be built. "+underlyingError(cause), cause, c.cfg.redactor(), hintInputs{})
}

// unexpectedError is the error for a failure of the SDK's own after a request was sent that fits no
// other class (spec 9.2). Its kind is KindUnexpected, the base class, with no status. The message
// says whether the request was sent only when that is known, and names the original's type; the
// original rides along as a redacted cause.
func (c *core) unexpectedError(cause error, requestSent bool) *Error {
	message := "The SDK failed unexpectedly, and the request may or may not have been sent."
	if requestSent {
		message = "The SDK failed unexpectedly after the request was sent."
	}
	return newSDKError(KindUnexpected, "unexpected_error", message+" "+underlyingError(cause), cause, c.cfg.redactor(), hintInputs{})
}

// Transcribe uploads the file at path, waits for it to be transcribed and returns the transcript. The
// format is decided by the file's bytes, not its name. A nil opts uses every default.
//
// Every error raised after the upload was created carries the idempotency key (Error.IdempotencyKey)
// and, once the job exists, a resume token (Error.ResumeToken); a cancelled ctx is a KindAbort error.
func (c Client) Transcribe(ctx context.Context, path string, opts *TranscribeOptions) (*Transcript, error) {
	p, err := c.pipeline()
	if err != nil {
		return nil, err
	}
	src, err := openPath(path, p.cfg.redactor())
	if err != nil {
		return nil, err
	}
	defer src.close()
	return p.transcribe(ctx, src, opts)
}

// TranscribeReader is Transcribe for a random-access reader of exactly size bytes. It reads each part
// at its own offset, never the whole file at once; a reader that holds more or fewer bytes than size
// is an UploadError with code stream_size_mismatch, before the upload is completed.
func (c Client) TranscribeReader(ctx context.Context, r io.ReaderAt, size int64, opts *TranscribeOptions) (*Transcript, error) {
	p, err := c.pipeline()
	if err != nil {
		return nil, err
	}
	red := p.cfg.redactor()
	if r == nil {
		return nil, invalidOption(red, "The reader cannot be nil.", nil)
	}
	if size <= 0 {
		return nil, invalidOption(red, "The size must be the exact number of bytes in the reader, above 0.", nil)
	}
	return p.transcribe(ctx, &fileSource{r: r, size: size}, opts)
}

// TranscribeStream is Transcribe for a forward-only reader of exactly size bytes, such as a pipe or a
// network body. The SDK cannot seek it or probe it, so opts.DurationSeconds is required, and the size
// must be exact: a reader that holds more or fewer bytes is an UploadError with code
// stream_size_mismatch, before the upload is completed. Parts are read one at a time, in order, with
// at most one part in memory. Canceling ctx is noticed between parts; close the reader to interrupt a
// read that is blocked. The SDK never closes r.
func (c Client) TranscribeStream(ctx context.Context, r io.Reader, size int64, opts *TranscribeOptions) (*Transcript, error) {
	p, err := c.pipeline()
	if err != nil {
		return nil, err
	}
	red := p.cfg.redactor()
	if r == nil {
		return nil, invalidOption(red, "The reader cannot be nil.", nil)
	}
	if size <= 0 {
		return nil, invalidOption(red, "The size must be the exact number of bytes in the reader, above 0.", nil)
	}
	return p.transcribe(ctx, newStreamSource(r, size), opts)
}

// run is the state of one transcribe or resume call that its stages share.
type run struct {
	core        *core
	state       resumeState
	src         *fileSource
	concurrency int
	waitTimeout time.Duration
	onProgress  func(Progress)
}

type createBody struct {
	SizeBytes            int64                 `json:"size_bytes"`
	DurationSeconds      int                   `json:"duration_seconds"`
	IdempotencyKey       string                `json:"idempotency_key"`
	Title                string                `json:"title,omitempty"`
	MatterCode           string                `json:"matter_code,omitempty"`
	TranscriptionOptions *transcriptionOptions `json:"transcription_options,omitempty"`
}

type transcriptionOptions struct {
	LanguageCode         string   `json:"language_code,omitempty"`
	SpeakerDetection     *bool    `json:"speaker_detection,omitempty"`
	TranslationTargets   []string `json:"translation_targets,omitempty"`
	KeytermsPrompt       []string `json:"keyterms_prompt,omitempty"`
	Prompt               string   `json:"prompt,omitempty"`
	MedicalMode          bool     `json:"medical_mode,omitempty"`
	VetMode              bool     `json:"vet_mode,omitempty"`
	KeepDisfluencies     bool     `json:"keep_disfluencies,omitempty"`
	Multichannel         bool     `json:"multichannel,omitempty"`
	AutoGenerateInsights *bool    `json:"auto_generate_insights,omitempty"`
}

func bodyFrom(opts *TranscribeOptions, size int64, declared int, key string) createBody {
	body := createBody{SizeBytes: size, DurationSeconds: declared, IdempotencyKey: key, Title: opts.Title, MatterCode: opts.MatterCode}
	normalized := normalizeContext(opts)
	options := transcriptionOptions{
		LanguageCode: opts.Language, SpeakerDetection: opts.Speakers, TranslationTargets: opts.TranslateTo,
		KeytermsPrompt: normalized.terms, Prompt: normalized.prompt, MedicalMode: opts.Medical, VetMode: opts.Vet, KeepDisfluencies: opts.KeepDisfluencies, Multichannel: opts.Multichannel,
		AutoGenerateInsights: opts.AutoInsights,
	}
	// Every field is omitempty, so the options encode to {} exactly when the caller set none of them.
	if encoded, _ := json.Marshal(options); string(encoded) != "{}" {
		body.TranscriptionOptions = &options
	}
	return body
}

// createUpload sends POST /v1/uploads. The doer paces it and replays it on a lost answer, because the
// body carries the idempotency key. On an error created and status are unset: the bare returns leave them so.
func (c *core) createUpload(ctx context.Context, body createBody) (created *createdUpload, status int, err error) {
	const route = "POST /v1/uploads"
	// createBody holds only integers, strings, string slices and bool pointers: json.Marshal cannot fail on it.
	encoded, _ := json.Marshal(body)
	response, err := c.api.CreateUploadWithBodyWithResponse(ctx, "application/json", bytes.NewReader(encoded))
	if err != nil {
		err = c.readFailed(err, route)
		return
	}
	parsed := decodedBody(response.JSON200, response.HTTPResponse, response.Body)
	if failure := c.unusable(route, response.HTTPResponse, parsed != nil, response.Body, "job_id", "state"); failure != nil {
		err = failure
		return
	}
	if !safeSegment(parsed.JobId) {
		err = c.badResponse(route, response.StatusCode())
		return
	}
	if upload := parsed.Upload; upload != nil && !partRowsUsable(upload.PartUrls) {
		err = c.badResponse(route, response.StatusCode())
		return
	}
	return &createdUpload{UploadCreateResponse: parsed, noDirect: createdWithoutPartURLs(response.Body)}, response.StatusCode(), nil
}

func (c *core) transcribe(ctx context.Context, src *fileSource, opts *TranscribeOptions) (result *Transcript, err error) {
	if opts == nil {
		opts = &TranscribeOptions{}
	}
	red := c.cfg.redactor()
	ann := &annotator{core: c}
	defer func() {
		if err != nil {
			result, err = nil, ann.annotate(ctx, err)
		}
	}()
	concurrency, waitTimeout, err := runtimeOptions(red, opts.Concurrency, opts.WaitTimeout)
	if err != nil {
		return nil, err
	}
	err = checkDuration(red, opts.DurationSeconds)
	if err != nil {
		return nil, err
	}
	if opts.IdempotencyKey != "" {
		err = checkKey(red, opts.IdempotencyKey)
		if err != nil {
			return nil, err
		}
	}
	err = checkOptions(red, opts)
	if err != nil {
		return nil, err
	}
	if src.stream != nil && opts.DurationSeconds == 0 {
		return nil, invalidOption(red, "A stream input needs DurationSeconds: it cannot be probed.", nil)
	}
	declared, err := c.preflight(ctx, src, opts)
	if err != nil {
		return nil, err
	}
	key := opts.IdempotencyKey
	if key == "" {
		key = newIdempotencyKey()
	}
	if opts.OnProgress != nil {
		opts.OnProgress(Progress{Stage: StageUploading, BytesTotal: src.size})
	}
	ann.key = key // from here the create may have reached the server
	created, status, err := c.createUpload(ctx, bodyFrom(opts, src.size, declared, key))
	if err != nil {
		return nil, err
	}
	state := resumeState{JobID: created.JobId, IdempotencyKey: key, Size: src.size, DurationSeconds: float64(declared)}
	token := encodeResumeToken(state)
	ann.token = token
	if opts.OnResumeToken != nil {
		opts.OnResumeToken(token)
	}
	plan, failure := c.planOf(created, status)
	if failure != nil {
		return nil, failure // after the token is known: the job exists, so the error carries the way back to it
	}
	r := &run{core: c, state: state, src: src, concurrency: concurrency, waitTimeout: waitTimeout, onProgress: opts.OnProgress}
	return r.finish(ctx, created.State, plan, nil)
}

// preflight is everything the SDK can refuse before a hold is placed (spec 8.1), in the spec's order.
// It returns the duration to declare: the probed or supplied seconds, rounded up. On an error the
// duration is unset: the bare returns leave it zero.
func (c *core) preflight(ctx context.Context, src *fileSource, opts *TranscribeOptions) (declared int, err error) {
	red := c.cfg.redactor()
	var format probe.Format
	if src.stream != nil {
		var head []byte
		if head, err = src.stream.takeHead(src.size); err != nil {
			err = fileUnreadable(red, "The stream could not be read.", err)
			return
		}
		format = probe.SniffAt(bytes.NewReader(head), int64(len(head)))
	} else {
		format = probe.SniffAt(src.r, src.size)
	}
	if format == probe.FormatUnknown {
		err = newSDKError(KindUnsupportedMedia, "unsupported_media", "The file is not in a supported audio format.", nil, red, hintInputs{})
		return
	}
	// No format is known in fewer than 2 bytes, so a file that got here holds at least 1.
	if src.size > maxFileBytes {
		err = newSDKError(KindFileLimit, "size_out_of_range",
			"The file is "+formatInt(src.size)+" bytes; the limit is 1 byte to 5000000000 bytes.", nil, red, hintInputs{})
		return
	}
	seconds := opts.DurationSeconds
	if seconds == 0 {
		if seconds, err = probe.Duration(src.r, src.size, format); err != nil {
			err = newSDKError(KindUnsupportedMedia, "duration_unreadable", "The file's duration could not be read.", err, red, hintInputs{})
			return
		}
	}
	if seconds < minDurationSeconds || math.Ceil(seconds) > maxDurationSeconds {
		err = newSDKError(KindFileLimit, "duration_out_of_range",
			"The duration is "+strconv.FormatFloat(seconds, 'f', -1, 64)+" s; it must be 15 s to 10 hours.", nil, red, hintInputs{})
		return
	}
	rounded := int(math.Ceil(seconds))

	const accountRoute = "GET /v1/account"
	account, err := c.api.GetAccountWithResponse(ctx)
	if err != nil {
		err = c.readFailed(err, accountRoute)
		return
	}
	parsed := decodedBody(account.JSON200, account.HTTPResponse, account.Body)
	if failure := c.unusable(accountRoute, account.HTTPResponse, parsed != nil, account.Body,
		"limits.used_today_seconds", "limits.max_daily_seconds"); failure != nil {
		err = failure
		return
	}
	limits := parsed.Limits
	if limits.UsedTodaySeconds+rounded > limits.MaxDailySeconds {
		failure := newSDKError(KindDailyLimit, "daily_limit",
			"This upload would pass the daily limit ("+strconv.Itoa(limits.UsedTodaySeconds)+" of "+strconv.Itoa(limits.MaxDailySeconds)+" s used).", nil, red,
			hintInputs{usedSeconds: some(float64(limits.UsedTodaySeconds)), maxSeconds: some(float64(limits.MaxDailySeconds))})
		failure.Details = map[string]any{"used_seconds": float64(limits.UsedTodaySeconds), "max_seconds": float64(limits.MaxDailySeconds)}
		err = failure
		return
	}
	declared = rounded
	return
}

// finish takes a job from the state its create or its read reported to a transcript. known is the job
// as Resume just read it, for a job that is not uploading.
func (r *run) finish(ctx context.Context, jobState string, plan *uploadPlan, known *Job) (*Transcript, error) {
	c := r.core
	finished := known
	if jobState == "uploading" {
		if r.src == nil {
			return nil, newSDKError(KindResumeMismatch, "resume_file_required",
				"The upload is not finished, so resuming needs the original file.", nil, c.cfg.redactor(), hintInputs{})
		}
		next, err := r.upload(ctx, plan)
		if err != nil {
			return nil, err
		}
		jobState, finished = next, nil
	}
	if jobState == "done" || jobState == "failed" {
		// The complete response has no transcript id: one job read gives it, with no poll loop.
		if finished == nil {
			job, err := c.readJob(ctx, r.state.JobID)
			if err != nil {
				return nil, err
			}
			finished = job
		}
		return c.fetchTranscript(ctx, finished)
	}
	if r.onProgress != nil && (jobState == "queued" || jobState == "processing") {
		r.onProgress(Progress{Stage: StageProcessing, State: jobState})
	}
	job, err := c.waitForJob(ctx, r.state.JobID, r.waitTimeout, r.onProgress)
	if err != nil {
		return nil, err
	}
	return c.fetchTranscript(ctx, job)
}
