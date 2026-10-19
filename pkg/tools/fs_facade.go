package tools

import (
	"github.com/sipeed/picoclaw/pkg/media"
	fstools "github.com/sipeed/picoclaw/pkg/tools/fs"
)

type (
	ReadFileTool      = fstools.ReadFileTool
	ReadFileLinesTool = fstools.ReadFileLinesTool
	WriteFileTool     = fstools.WriteFileTool
	ListDirTool       = fstools.ListDirTool
	EditFileTool      = fstools.EditFileTool
	AppendFileTool    = fstools.AppendFileTool
	LoadImageTool     = fstools.LoadImageTool
	SendFileTool      = fstools.SendFileTool
)

const MaxReadFileSize = fstools.MaxReadFileSize

func NewReadFileTool(workspace string, maxReadFileSize int) *ReadFileTool {
	return fstools.NewReadFileTool(workspace, maxReadFileSize)
}

func NewReadFileBytesTool(workspace string, maxReadFileSize int) *ReadFileTool {
	return fstools.NewReadFileBytesTool(workspace, maxReadFileSize)
}

func NewReadFileLinesTool(workspace string, maxReadFileSize int) *ReadFileLinesTool {
	return fstools.NewReadFileLinesTool(workspace, maxReadFileSize)
}

func NewWriteFileTool(workspace string) *WriteFileTool {
	return fstools.NewWriteFileTool(workspace)
}

func NewListDirTool(workspace string) *ListDirTool {
	return fstools.NewListDirTool(workspace)
}

func NewEditFileTool(workspace string) *EditFileTool {
	return fstools.NewEditFileTool(workspace)
}

func NewAppendFileTool(workspace string) *AppendFileTool {
	return fstools.NewAppendFileTool(workspace)
}

func NewLoadImageTool(workspace string, maxFileSize int, store media.MediaStore) *LoadImageTool {
	return fstools.NewLoadImageTool(workspace, maxFileSize, store)
}

func NewSendFileTool(workspace string, maxFileSize int, store media.MediaStore) *SendFileTool {
	return fstools.NewSendFileTool(workspace, maxFileSize, store)
}
