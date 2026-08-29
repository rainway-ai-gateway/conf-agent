// Copyright(c) 2026 The Rainway AI Gateway (壬远AI网关) Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package file_store

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/rainway-ai-gateway/conf-agent/xfile"
	"github.com/rainway-ai-gateway/conf-agent/xlog"
)

const versionMarkerFile = ".conf-agent-version"

type FileStore struct {
	// ConfDir is the root dir of conf file
	ConfDir string
	// CoypFiles is list of files and directories copied from default dir to tmp dir
	CopyFiles []string
	// VersionKeepCount is the number of version directories to keep
	VersionKeepCount int
}

// compose path of tempory directory to store files
func (fileStore *FileStore) tmpDir(version string) string {
	return fileStore.ConfDir + "_" + version
}

func NewFileStore(confDir string, copyFiles []string, versionKeepCount int) (*FileStore, error) {
	if versionKeepCount < 1 {
		versionKeepCount = 1
	}

	return &FileStore{
		ConfDir:          confDir,
		CopyFiles:        copyFiles,
		VersionKeepCount: versionKeepCount,
	}, nil
}

// writeVersionMarker writes a marker file to identify a conf-agent managed version directory.
func (fileStore *FileStore) writeVersionMarker(tmpDir, version string) error {
	markerFile := filepath.Join(tmpDir, versionMarkerFile)
	if err := os.WriteFile(markerFile, []byte(version), 0644); err != nil {
		return fmt.Errorf("write version marker fail, file: %s, err: %v", markerFile, err)
	}
	return nil
}

// cleanupOldVersions removes expired version directories, keeping the current
// target plus the most recent VersionKeepCount-1 previous versions.
func (fileStore *FileStore) cleanupOldVersions(ctx context.Context, keep int) error {
	if keep < 1 {
		keep = 1
	}

	parentDir := filepath.Dir(fileStore.ConfDir)
	baseName := filepath.Base(fileStore.ConfDir)

	currentTarget, err := filepath.EvalSymlinks(fileStore.ConfDir)
	if err != nil {
		// If the link does not exist or is broken, there is nothing to protect.
		currentTarget = ""
	}

	entries, err := os.ReadDir(parentDir)
	if err != nil {
		return fmt.Errorf("read parent dir fail, dir: %s, err: %v", parentDir, err)
	}

	type versionDir struct {
		path    string
		modTime time.Time
	}

	var versions []versionDir
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasPrefix(name, baseName+"_") {
			continue
		}
		// Skip backup directories created from regular directory migration.
		if strings.HasSuffix(name, ".backup") {
			continue
		}

		dirPath := filepath.Join(parentDir, name)
		markerFile := filepath.Join(dirPath, versionMarkerFile)
		if _, err := os.Stat(markerFile); err != nil {
			continue
		}

		// Never remove the currently active target.
		absDirPath, _ := filepath.Abs(dirPath)
		absCurrent, _ := filepath.Abs(currentTarget)
		if absDirPath == absCurrent {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			xlog.Default.Error(xlog.ErrLogFormat(ctx, "cleanupOldVersions.Info", err))
			continue
		}

		versions = append(versions, versionDir{path: dirPath, modTime: info.ModTime()})
	}

	// Keep the newest versions first.
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].modTime.After(versions[j].modTime)
	})

	var removeErrs []string
	for i, v := range versions {
		if i < keep-1 {
			continue
		}

		if err := os.RemoveAll(v.path); err != nil {
			removeErrs = append(removeErrs, fmt.Sprintf("%s: %v", v.path, err))
		}
	}

	if len(removeErrs) > 0 {
		return fmt.Errorf("cleanupOldVersions remove fail: %s", strings.Join(removeErrs, "; "))
	}

	return nil
}

// UpdateDefaultConfDir updates default config directory with config files in tempory directory.
func (fileStore *FileStore) UpdateDefaultConfDir(ctx context.Context, version string) error {
	// Inspect ConfDir without following symlinks so we can distinguish a symlink
	// from its target and from a regular directory.
	info, err := os.Lstat(fileStore.ConfDir)

	switch {
	case err == nil:
		if info.Mode()&os.ModeSymlink != 0 {
			// ConfDir is a symlink: remove the link itself. The previous target is kept
			// so cleanupOldVersions can retain up to VersionKeepCount versions.
			if err := os.Remove(fileStore.ConfDir); err != nil {
				err = fmt.Errorf("file: %s, err: %v", fileStore.ConfDir, err)
				xlog.Default.Error(xlog.ErrLogFormat(ctx, "UpdateDefaultConfDir.RemoveLink", err))
				return err
			}
		} else if info.IsDir() {
			// ConfDir is a regular directory: back it up instead of deleting it.
			backupDir := fileStore.ConfDir + "_" + strconv.FormatInt(time.Now().Unix(), 10) + ".backup"
			if err := os.Rename(fileStore.ConfDir, backupDir); err != nil {
				err = fmt.Errorf("backup dir fail, from: %s, to: %s, err: %v", fileStore.ConfDir, backupDir, err)
				xlog.Default.Error(xlog.ErrLogFormat(ctx, "UpdateDefaultConfDir.Backup", err))
				return err
			}
		} else {
			// ConfDir exists as a non-directory, non-symlink entry: remove it.
			if err := os.RemoveAll(fileStore.ConfDir); err != nil {
				err = fmt.Errorf("file: %s, err: %v", fileStore.ConfDir, err)
				xlog.Default.Error(xlog.ErrLogFormat(ctx, "UpdateDefaultConfDir.Remove", err))
				return err
			}
		}

	case os.IsNotExist(err):
		// ConfDir does not exist yet: create the new link directly.

	default:
		// Broken symlink or other Lstat error: try to remove the path.
		if err := os.RemoveAll(fileStore.ConfDir); err != nil && !os.IsNotExist(err) {
			err = fmt.Errorf("file: %s, err: %v", fileStore.ConfDir, err)
			xlog.Default.Error(xlog.ErrLogFormat(ctx, "UpdateDefaultConfDir.Remove", err))
			return err
		}
	}

	// ln -sf ModDemo_{version} ModDemo
	// NOTICE: if link fail, bfe can't restart automatically !!!
	if err := xfile.FileLink(fileStore.tmpDir(version), fileStore.ConfDir); err != nil {
		xlog.Default.Error(xlog.ErrLogFormat(ctx, "UpdateDefaultConfDir.FileLink", err))
		return err
	}

	// Clean up expired version directories after a successful switch.
	if err := fileStore.cleanupOldVersions(ctx, fileStore.VersionKeepCount); err != nil {
		xlog.Default.Error(xlog.ErrLogFormat(ctx, "UpdateDefaultConfDir.cleanupOldVersions", err))
	}

	return nil
}

// StoreFile2TmpDir store all file to tempory directory
// it will create new file or overwrite old file
func (fileStore *FileStore) StoreFile2TmpDir(ctx context.Context, version string, files map[string][]byte) error {
	tmpDir := fileStore.tmpDir(version)

	// delete tmp directory if exist
	if err := os.RemoveAll(tmpDir); err != nil && !xfile.IsFileNotExistError(err) {
		err = fmt.Errorf("RemoveAll fail, dir: %s, err: %v", tmpDir, err)
		xlog.Default.Error(xlog.ErrLogFormat(ctx, "fileStore.RemoveAll", err))

		return err
	}

	// create tmp directory
	if err := os.MkdirAll(tmpDir, os.ModePerm); err != nil {
		err = fmt.Errorf("MkDirAll fail, dir: %s, err: %v", tmpDir, err)
		xlog.Default.Error(xlog.ErrLogFormat(ctx, "fileStore.MkdirAll", err))

		return err
	}

	// copy config files (listed in fileStore.CopyFiles) from default dir to tmp dir
	for _, copyFile := range fileStore.CopyFiles {
		file := filepath.Join(fileStore.ConfDir, copyFile)
		if err := xfile.FileCopyRecursive(file, tmpDir); err != nil {
			if xfile.IsFileNotExistError(err) {
				xlog.Default.Info(xlog.ErrLogFormat(ctx, "fileStore.CopyFiles", err))
				continue
			}

			err = fmt.Errorf("keepFile fail, file: %s, err: %v", file, err)
			xlog.Default.Error(xlog.ErrLogFormat(ctx, "fileStore.CopyFiles", err))

			return err
		}
	}

	// write content to file
	for fileName, fileContent := range files {
		formattedContent := formatJSONWithIndent(fileContent)
		if err := xfile.FileOverwrite(filepath.Join(tmpDir, fileName), formattedContent); err != nil {
			xlog.Default.Error(xlog.ErrLogFormat(ctx, "fileStore.FileOverwrite", err))
			return err
		}

		// xlog.Default.Debug(xlog.InfoLogFormat(ctx, "fileStore.FileOverwrite", "fileName: ", fileName,
		// 	" fileContent: ", string(fileContent)))
	}

	// write version marker so the directory can be identified as conf-agent managed
	if err := fileStore.writeVersionMarker(tmpDir, version); err != nil {
		xlog.Default.Error(xlog.ErrLogFormat(ctx, "fileStore.writeVersionMarker", err))
		return err
	}

	return nil
}

func formatJSONWithIndent(content []byte) []byte {
	var jsonData interface{}
	if err := json.Unmarshal(content, &jsonData); err != nil {
		return content
	}

	formatted, err := json.MarshalIndent(jsonData, "", "    ")
	if err != nil {
		return content
	}

	return formatted
}
