package fstools

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/sipeed/picoclaw/pkg/fileutil"
	"github.com/sipeed/picoclaw/pkg/logger"
)

const MaxReadFileSize = 64 * 1024 // 64KB limit to avoid context overflow

// ResolvePath turns a tool path argument into an absolute path.
// Relative paths resolve against the workspace; absolute paths are used as given.
func ResolvePath(path, workspace string) (string, error) {
	if filepath.IsAbs(path) {
		return filepath.Clean(path), nil
	}
	if workspace == "" {
		return path, nil
	}

	absWorkspace, err := filepath.Abs(workspace)
	if err != nil {
		return "", fmt.Errorf("failed to resolve workspace path: %w", err)
	}

	absPath, err := filepath.Abs(filepath.Join(absWorkspace, path))
	if err != nil {
		return "", fmt.Errorf("failed to resolve file path: %w", err)
	}

	return absPath, nil
}




type ReadFileTool struct {
	fs      fileSystem
	maxSize int64
}

type ReadFileLinesTool struct {
	fs      fileSystem
	maxSize int64
}

func NewReadFileTool(workspace string, maxReadFileSize int) *ReadFileTool {
	maxSize := int64(maxReadFileSize)
	if maxSize <= 0 {
		maxSize = MaxReadFileSize
	}

	return &ReadFileTool{
		fs:      &hostFs{workspace: workspace},
		maxSize: maxSize,
	}
}

func NewReadFileBytesTool(workspace string, maxReadFileSize int) *ReadFileTool {
	return NewReadFileTool(workspace, maxReadFileSize)
}

func NewReadFileLinesTool(workspace string, maxReadFileSize int) *ReadFileLinesTool {
	maxSize := int64(maxReadFileSize)
	if maxSize <= 0 {
		maxSize = MaxReadFileSize
	}

	return &ReadFileLinesTool{
		fs:      &hostFs{workspace: workspace},
		maxSize: maxSize,
	}
}

func (t *ReadFileTool) Name() string {
	return "read_file"
}

func (t *ReadFileLinesTool) Name() string {
	return "read_file"
}

func (t *ReadFileTool) Description() string {
	return "Read the contents of a file. Supports pagination via `offset` and `length`."
}

func (t *ReadFileLinesTool) Description() string {
	return "Read a UTF-8 text file from the filesystem. Output always includes line numbers in the format `LINE_NUMBER|LINE_CONTENT` (1-indexed). Supports partial reads via `start_line` and `max_lines` for large text files."
}

func (t *ReadFileTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Path to the file to read.",
			},
			"offset": map[string]any{
				"type":        "integer",
				"description": "Byte offset to start reading from.",
				"default":     0,
			},
			"length": map[string]any{
				"type":        "integer",
				"description": "Maximum number of bytes to read.",
				"default":     t.maxSize,
			},
		},
		"required": []string{"path"},
	}
}

func (t *ReadFileLinesTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Path to the file to read.",
			},
			"start_line": map[string]any{
				"type":        "integer",
				"description": "Line number to start reading from (1-indexed, inclusive).",
				"default":     1,
			},
			"max_lines": map[string]any{
				"type":        "integer",
				"description": "Maximum number of lines to read.",
			},
		},
		"required": []string{"path"},
	}
}

func (t *ReadFileTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	path, ok := args["path"].(string)
	if !ok {
		return ErrorResult("path is required")
	}

	// offset (optional, default 0)
	offset, err := getInt64Arg(args, "offset", 0)
	if err != nil {
		return ErrorResult(err.Error())
	}
	if offset < 0 {
		return ErrorResult("offset must be >= 0")
	}

	// length (optional, capped at MaxReadFileSize)
	length, err := getInt64Arg(args, "length", t.maxSize)
	if err != nil {
		return ErrorResult(err.Error())
	}
	if length <= 0 {
		return ErrorResult("length must be > 0")
	}
	if length > t.maxSize {
		length = t.maxSize
	}

	file, err := t.fs.Open(path)
	if err != nil {
		return ErrorResult(err.Error())
	}
	defer file.Close()

	// measure total size
	totalSize := int64(-1) // -1 means unknown
	if info, statErr := file.Stat(); statErr == nil {
		totalSize = info.Size()
	}

	// sniff the first 512 bytes to detect binary content before loading
	// it into the LLM context. Seeking back to 0 afterwards restores state.
	sniff := make([]byte, 512)
	sniffN, _ := file.Read(sniff)

	// Reset read position to beginning before applying the caller's offset.
	if seeker, ok := file.(io.Seeker); ok {
		_, err = seeker.Seek(0, io.SeekStart)
		if err != nil {
			return ErrorResult(fmt.Sprintf("failed to reset file position after sniff: %v", err))
		}
	} else {
		// Non-seekable: we consumed sniffN bytes above; account for them when
		// discarding to reach the requested offset below.
		// If offset < sniffN the data we already read covers it, which we
		// cannot replay on a non-seekable stream — return a clear error.
		if offset < int64(sniffN) && offset > 0 {
			return ErrorResult(
				"non-seekable file: cannot seek to an offset within the first 512 bytes after binary detection",
			)
		}
	}

	// Seek to the requested offset.
	if seeker, ok := file.(io.Seeker); ok {
		_, err = seeker.Seek(offset, io.SeekStart)
		if err != nil {
			return ErrorResult(fmt.Sprintf("failed to seek to offset %d: %v", offset, err))
		}
	} else if offset > 0 {
		// Fallback for non-seekable streams: discard leading bytes.
		// sniffN bytes were already consumed above, so subtract them.
		remaining := offset - int64(sniffN)
		if remaining > 0 {
			_, err = io.CopyN(io.Discard, file, remaining)
			if err != nil {
				return ErrorResult(fmt.Sprintf("failed to advance to offset %d: %v", offset, err))
			}
		}
	}

	// read length+1 bytes to reliably detect whether more content exists
	// without relying on totalSize (which may be -1 for non-seekable streams).
	// This avoids the false-positive TRUNCATED message on the last page.
	probe := make([]byte, length+1)
	n, err := io.ReadFull(file, probe)
	// FIX: io.ReadFull returns io.ErrUnexpectedEOF for partial reads (0 < n < len),
	// and io.EOF only when n == 0. Both are normal terminal conditions — only
	// other errors are genuine failures.
	if err != nil && err != io.EOF && !errors.Is(err, io.ErrUnexpectedEOF) {
		return ErrorResult(fmt.Sprintf("failed to read file content: %v", err))
	}

	// hasMore is true only when we actually got the extra probe byte.
	hasMore := int64(n) > length
	data := probe[:min(int64(n), length)]

	if len(data) == 0 {
		return NewToolResult("[END OF FILE - no content at this offset]")
	}

	// Build metadata header.
	// use filepath.Base(path) instead of the raw path to avoid leaking
	// internal filesystem structure into the LLM context.
	readEnd := offset + int64(len(data))
	// use ASCII hyphen-minus instead of en-dash (U+2013) to keep the
	// header parseable by downstream tools and log processors.
	readRange := fmt.Sprintf("bytes %d-%d", offset, readEnd-1)

	displayPath := filepath.Base(path)
	var header string
	if totalSize >= 0 {
		header = fmt.Sprintf(
			"[file: %s | total: %d bytes | read: %s]",
			displayPath, totalSize, readRange,
		)
	} else {
		header = fmt.Sprintf(
			"[file: %s | read: %s | total size unknown]",
			displayPath, readRange,
		)
	}

	if hasMore {
		header += fmt.Sprintf(
			"\n[TRUNCATED - file has more content. Call read_file again with offset=%d to continue.]",
			readEnd,
		)
	} else {
		header += "\n[END OF FILE - no further content.]"
	}

	logger.DebugCF("tool", "ReadFileTool execution completed successfully",
		map[string]any{
			"path":       path,
			"bytes_read": len(data),
			"has_more":   hasMore,
		})

	return NewToolResult(header + "\n\n" + string(data))
}

func (t *ReadFileLinesTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	path, ok := args["path"].(string)
	if !ok {
		return ErrorResult("path is required")
	}

	startLine, err := getInt64Arg(args, "start_line", 1)
	if err != nil {
		return ErrorResult(err.Error())
	}
	if startLine < 1 {
		return ErrorResult("start_line must be >= 1")
	}
	if _, exists := args["offset"]; exists {
		return ErrorResult("offset is not supported in line mode; use start_line")
	}
	if _, exists := args["length"]; exists {
		return ErrorResult("length is not supported in line mode; use max_lines")
	}
	if _, exists := args["limit"]; exists {
		return ErrorResult("limit is not supported in line mode; use max_lines")
	}

	limit := int64(-1)
	if raw, exists := args["max_lines"]; exists && raw != nil {
		limit, err = getInt64Arg(args, "max_lines", -1)
		if err != nil {
			return ErrorResult(err.Error())
		}
		if limit <= 0 {
			return ErrorResult("max_lines, if provided, must be > 0")
		}
	}

	file, err := t.fs.Open(path)
	if err != nil {
		return ErrorResult(err.Error())
	}
	defer file.Close()

	if info, statErr := file.Stat(); statErr == nil && info.IsDir() {
		return ErrorResult(fmt.Sprintf("failed to open file: path is a directory: %s", path))
	}

	sample := make([]byte, 512)
	sampleN, readErr := file.Read(sample)
	if readErr != nil && readErr != io.EOF {
		return ErrorResult(fmt.Sprintf("failed to read file: %v", readErr))
	}
	sample = sample[:sampleN]
	if isBinaryReadFileData(sample) {
		return ErrorResult("file appears to be binary; switch read_file mode to 'bytes' for byte-based inspection")
	}

	reader := bufio.NewReaderSize(io.MultiReader(bytes.NewReader(sample), file), 32*1024)

	var content strings.Builder
	lineIndex := int64(1)
	var linesRead int64
	var fileBytesRead int64
	var outputBytesRead int64
	var reachedEOF bool
	var byteBudgetTruncated bool
	var lineTruncated bool

	for lineIndex < startLine {
		hasLine, consumeErr := consumeNextLine(reader)
		if consumeErr != nil {
			return ErrorResult(fmt.Sprintf("failed to read file content: %v", consumeErr))
		}
		if !hasLine {
			reachedEOF = true
			break
		}
		lineIndex++
	}

	for !reachedEOF && (limit < 0 || linesRead < limit) {
		prefix := formatReadFileLinePrefix(lineIndex)
		remaining := t.maxSize - outputBytesRead - int64(len(prefix))
		if remaining <= 0 {
			byteBudgetTruncated = true
			break
		}

		line, complete, hasLine, readLineErr := readNextLinePrefix(reader, remaining)
		if readLineErr != nil {
			return ErrorResult(fmt.Sprintf("failed to read file content: %v", readLineErr))
		}
		if !hasLine {
			reachedEOF = true
			break
		}

		content.WriteString(prefix)
		content.Write(line)
		fileBytesRead += int64(len(line))
		outputBytesRead += int64(len(prefix) + len(line))
		linesRead++
		lineIndex++

		if !complete {
			byteBudgetTruncated = true
			lineTruncated = true
			break
		}
	}

	if !reachedEOF && !lineTruncated {
		hasMoreContent, peekErr := readerHasMoreContent(reader)
		if peekErr != nil {
			return ErrorResult(fmt.Sprintf("failed to inspect remaining file content: %v", peekErr))
		}
		if !hasMoreContent {
			reachedEOF = true
			byteBudgetTruncated = false
		}
	}

	if linesRead == 0 && content.Len() == 0 {
		return NewToolResult(fmt.Sprintf("[END OF FILE - no content at or after start_line=%d]", startLine))
	}

	start := startLine
	endLine := startLine + linesRead - 1
	displayPath := filepath.Base(path)
	header := fmt.Sprintf(
		"[file: %s | read: lines %d-%d (1-indexed) | file_bytes: %d | output_bytes: %d]",
		displayPath, start, endLine, fileBytesRead, outputBytesRead,
	)

	switch {
	case lineTruncated:
		header += fmt.Sprintf(
			"\n[TRUNCATED - line %d exceeded the %d byte read budget and was cut mid-line.]",
			endLine,
			t.maxSize,
		)
	case byteBudgetTruncated:
		if limit > 0 {
			header += fmt.Sprintf(
				"\n[TRUNCATED - byte budget reached. Call read_file again with start_line=%d and max_lines=%d to continue at the next line.]",
				startLine+linesRead,
				limit,
			)
		} else {
			header += fmt.Sprintf(
				"\n[TRUNCATED - byte budget reached. Call read_file again with start_line=%d to continue at the next line.]",
				startLine+linesRead,
			)
		}
	case !reachedEOF && limit > 0 && linesRead >= limit:
		header += fmt.Sprintf(
			"\n[PARTIAL - more content remains. Call read_file again with start_line=%d and max_lines=%d to continue.]",
			startLine+linesRead,
			limit,
		)
	default:
		header += "\n[END OF FILE - no further content.]"
	}

	logger.DebugCF("tool", "ReadFileTool execution completed successfully",
		map[string]any{
			"path":              path,
			"lines_read":        linesRead,
			"file_bytes_read":   fileBytesRead,
			"output_bytes_read": outputBytesRead,
			"truncated":         byteBudgetTruncated,
			"tool":              t.Name(),
		})

	return NewToolResult(header + "\n\n" + content.String())
}

func formatReadFileLinePrefix(lineNumber int64) string {
	return strconv.FormatInt(lineNumber, 10) + "|"
}

func isBinaryReadFileData(data []byte) bool {
	if len(data) == 0 {
		return false
	}

	sample := data
	if len(sample) > 512 {
		sample = sample[:512]
	}

	if bytes.IndexByte(sample, 0) >= 0 {
		return true
	}

	contentType := http.DetectContentType(sample)
	if strings.HasPrefix(contentType, "text/") {
		return false
	}
	if strings.HasSuffix(contentType, "/json") ||
		strings.HasSuffix(contentType, "+json") ||
		strings.HasSuffix(contentType, "/xml") ||
		strings.HasSuffix(contentType, "+xml") ||
		strings.Contains(contentType, "javascript") {
		return false
	}

	if !utf8.Valid(sample) {
		return true
	}

	controlChars := 0
	for _, b := range sample {
		if b < 0x20 && b != '\n' && b != '\r' && b != '\t' && b != '\f' && b != '\b' {
			controlChars++
		}
	}

	return float64(controlChars)/float64(len(sample)) > 0.1
}

func consumeNextLine(reader *bufio.Reader) (bool, error) {
	sawData := false

	for {
		fragment, err := reader.ReadSlice('\n')
		if len(fragment) > 0 {
			sawData = true
		}

		switch {
		case err == nil:
			return true, nil
		case errors.Is(err, bufio.ErrBufferFull):
			continue
		case errors.Is(err, io.EOF):
			return sawData, nil
		default:
			return false, err
		}
	}
}

func readNextLinePrefix(reader *bufio.Reader, maxBytes int64) ([]byte, bool, bool, error) {
	if maxBytes <= 0 {
		return nil, false, false, nil
	}

	var out bytes.Buffer
	sawData := false
	complete := true

	for {
		fragment, err := reader.ReadSlice('\n')
		if len(fragment) > 0 {
			sawData = true
			if remaining := maxBytes - int64(out.Len()); remaining > 0 {
				take := len(fragment)
				if int64(take) > remaining {
					take = int(remaining)
					complete = false
				}
				out.Write(fragment[:take])
			} else {
				complete = false
			}
		}

		switch {
		case err == nil:
			return out.Bytes(), complete, sawData, nil
		case errors.Is(err, bufio.ErrBufferFull):
			if !complete {
				return out.Bytes(), false, true, nil
			}
			continue
		case errors.Is(err, io.EOF):
			if !sawData {
				return nil, true, false, nil
			}
			return out.Bytes(), complete, true, nil
		default:
			return nil, false, false, err
		}
	}
}

func readerHasMoreContent(reader *bufio.Reader) (bool, error) {
	_, err := reader.Peek(1)
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, io.EOF):
		return false, nil
	default:
		return false, err
	}
}

// getInt64Arg extracts an integer argument from the args map, returning the
// provided default if the key is absent.
func getInt64Arg(args map[string]any, key string, defaultVal int64) (int64, error) {
	raw, exists := args[key]
	if !exists {
		return defaultVal, nil
	}

	switch v := raw.(type) {
	case float64:
		if v != math.Trunc(v) {
			return 0, fmt.Errorf("%s must be an integer, got float %v", key, v)
		}
		if v > math.MaxInt64 || v < math.MinInt64 {
			return 0, fmt.Errorf("%s value %v overflows int64", key, v)
		}
		return int64(v), nil
	case int:
		return int64(v), nil
	case int64:
		return v, nil
	case string:
		parsed, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid integer format for %s parameter: %w", key, err)
		}
		return parsed, nil
	default:
		return 0, fmt.Errorf("unsupported type %T for %s parameter", raw, key)
	}
}

type WriteFileTool struct {
	fs       fileSystem
	altTools []string
}

func NewWriteFileTool(workspace string) *WriteFileTool {
	// Default to both alternatives so standalone callers keep the full guidance;
	// the agent wiring narrows this to the tools actually registered.
	return &WriteFileTool{
		fs:       &hostFs{workspace: workspace},
		altTools: []string{"append_file", "edit_file"},
	}
}

// SetAlternativeTools limits which alternatives the copy names, so it never
// directs the model to tools that are not available.
func (t *WriteFileTool) SetAlternativeTools(names []string) {
	present := make(map[string]bool, len(names))
	for _, name := range names {
		present[name] = true
	}
	ordered := make([]string, 0, 2)
	for _, name := range []string{"append_file", "edit_file"} {
		if present[name] {
			ordered = append(ordered, name)
		}
	}
	t.altTools = ordered
}

func (t *WriteFileTool) altToolsPhrase() string {
	return strings.Join(t.altTools, " or ")
}

func (t *WriteFileTool) Name() string {
	return "write_file"
}

func (t *WriteFileTool) Description() string {
	desc := "Write content to a file, replacing any existing content. Content is written byte-for-byte after argument decoding. Standard JSON escaping applies: \\n for newline and \\\\n for a literal backslash-n sequence. If the file already exists you must set overwrite=true, which replaces the ENTIRE file."
	if phrase := t.altToolsPhrase(); phrase != "" {
		desc += fmt.Sprintf(
			" To add to or change part of an existing file without losing its current contents, use %s instead.",
			phrase,
		)
	}
	return desc
}

func (t *WriteFileTool) Parameters() map[string]any {
	overwriteDesc := "Set to true to replace an existing file in full. This discards the file's current contents."
	if phrase := t.altToolsPhrase(); phrase != "" {
		overwriteDesc = fmt.Sprintf(
			"Set to true to replace an existing file in full. This discards the file's current contents — to preserve them, use %s instead of write_file.",
			phrase,
		)
	}
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Path to the file to write",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "Content to write to the file. Standard JSON escaping applies: \\n for newline and \\\\n for literal backslash-n.",
			},
			"overwrite": map[string]any{
				"type":        "boolean",
				"description": overwriteDesc,
				"default":     false,
			},
		},
		"required": []string{"path", "content"},
	}
}

func (t *WriteFileTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	path, ok := args["path"].(string)
	if !ok {
		return ErrorResult("path is required")
	}

	content, ok := args["content"].(string)
	if !ok {
		return ErrorResult("content is required")
	}

	overwrite, _ := args["overwrite"].(bool)

	if !overwrite {
		if _, err := t.fs.Open(path); err == nil {
			if phrase := t.altToolsPhrase(); phrase != "" {
				return ErrorResult(
					fmt.Sprintf(
						"file: %s already exists. To add to it or change part of it without losing the current contents, use %s. Only set overwrite=true if you intend to replace the entire file.",
						path,
						phrase,
					),
				)
			}
			return ErrorResult(
				fmt.Sprintf(
					"file: %s already exists. Set overwrite=true only if you intend to replace the entire file, which discards its current contents.",
					path,
				),
			)
		}
	}

	if err := t.fs.WriteFile(path, []byte(content)); err != nil {
		return ErrorResult(err.Error())
	}

	return SilentResult(fmt.Sprintf("File written: %s", path))
}

type ListDirTool struct {
	fs fileSystem
}

func NewListDirTool(workspace string) *ListDirTool {
	return &ListDirTool{fs: &hostFs{workspace: workspace}}
}

func (t *ListDirTool) Name() string {
	return "list_dir"
}

func (t *ListDirTool) Description() string {
	return "List files and directories in a path"
}

func (t *ListDirTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Path to list",
			},
		},
		"required": []string{"path"},
	}
}

func (t *ListDirTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	path, ok := args["path"].(string)
	if !ok {
		path = "."
	}

	entries, err := t.fs.ReadDir(path)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to read directory: %v", err))
	}
	return formatDirEntries(entries)
}

func formatDirEntries(entries []os.DirEntry) *ToolResult {
	var result strings.Builder
	for _, entry := range entries {
		if entry.IsDir() {
			result.WriteString("DIR:  " + entry.Name() + "\n")
		} else {
			result.WriteString("FILE: " + entry.Name() + "\n")
		}
	}
	return NewToolResult(result.String())
}

// fileSystem abstracts reading, writing, and listing files, allowing both
// unrestricted (host filesystem) and sandbox (os.Root) implementations to share the same polymorphic interface.
type fileSystem interface {
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte) error
	ReadDir(path string) ([]os.DirEntry, error)
	Open(path string) (fs.File, error)
}

// hostFs reads and writes files directly on the host filesystem. Relative paths
// resolve against the workspace, which stays the default base for path arguments.
type hostFs struct {
	workspace string
}

func (h *hostFs) resolve(path string) (string, error) {
	return ResolvePath(path, h.workspace)
}

func (h *hostFs) ReadFile(path string) ([]byte, error) {
	resolved, err := h.resolve(path)
	if err != nil {
		return nil, err
	}
	content, err := os.ReadFile(resolved)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to read file: file not found: %w", err)
		}
		if os.IsPermission(err) {
			return nil, fmt.Errorf("failed to read file: access denied: %w", err)
		}
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	return content, nil
}

func (h *hostFs) ReadDir(path string) ([]os.DirEntry, error) {
	resolved, err := h.resolve(path)
	if err != nil {
		return nil, err
	}
	return os.ReadDir(resolved)
}

func (h *hostFs) WriteFile(path string, data []byte) error {
	resolved, err := h.resolve(path)
	if err != nil {
		return err
	}
	// Use unified atomic write utility with explicit sync for flash storage reliability.
	// Using 0o600 (owner read/write only) for secure default permissions.
	return fileutil.WriteFileAtomic(resolved, data, 0o600)
}

func (h *hostFs) Open(path string) (fs.File, error) {
	resolved, err := h.resolve(path)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(resolved)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to open file: file not found: %w", err)
		}
		if os.IsPermission(err) {
			return nil, fmt.Errorf("failed to open file: access denied: %w", err)
		}
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	return f, nil
}




