// Copyright 2020 Tetrate
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

package registry

import (
	"fmt"
	"io"
	"io/fs"
	"path"
	"path/filepath"
	"strings"

	"github.com/tetratelabs/getenvoy/pkg/extension/workspace/config/extension"
	"github.com/tetratelabs/getenvoy/pkg/extension/workspace/model"
)

// registry represents a registry of example templates.
type registry interface {
	// Get returns a registry entry.
	Get(descriptor *extension.Descriptor, example string) (*Entry, error)
}

// fsRegistry represents a registry of example templates backed by
// an in-memory file system.
type fsRegistry struct {
	fs           fs.FS
	namingScheme func(category extension.Category, example string) string
}

func (r *fsRegistry) Get(descriptor *extension.Descriptor, example string) (*Entry, error) {
	dirName := r.namingScheme(descriptor.Category, example)
	dir, err := r.fs.Open(dirName)
	if err != nil {
		return nil, fmt.Errorf(`failed to open %q: %w`, dirName, err)
	}
	defer dir.Close() //nolint:errcheck
	info, err := dir.Stat()
	if err != nil {
		return nil, fmt.Errorf(`failed to stat %q: %w`, dirName, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf(`%q is not a directory: %w`, dirName, err)
	}

	return &Entry{
		Category: descriptor.Category,
		Name:     example,
		NewExample: func(*extension.Descriptor) (model.Example, error) {
			fileSet := model.NewFileSet()

			// Add language independent files
			fileNames, err := listFiles(r.fs, dirName)
			if err != nil {
				return nil, fmt.Errorf(`failed to list files in a directory %q: %w`, dirName, err)
			}
			for _, fileName := range fileNames {
				err = r.addFile(fileSet, dirName, fileName, descriptor.Language)
				if err != nil {
					return nil, err
				}
			}

			// Add language specific files
			languageDir := path.Join("language-specific", descriptor.Language.String())
			fileNames, err = listFiles(r.fs, languageDir)
			if err != nil {
				return nil, fmt.Errorf(`failed to list files in a directory %q: %w`, dirName, err)
			}
			for _, fileName := range fileNames {
				err = r.addFile(fileSet, languageDir, fileName, descriptor.Language)
				if err != nil {
					return nil, err
				}
			}
			return model.NewExample(fileSet)
		},
	}, nil
}

func (r *fsRegistry) addFile(fileSet model.FileSet, dirName, fileName string, language extension.Language) error {
	file, err := r.fs.Open(fileName)
	if err != nil {
		return fmt.Errorf(`failed to open %q: %w`, dirName, err)
	}
	defer file.Close() //nolint:errcheck
	data, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf(`failed to read %q: %w`, fileName, err)
	}
	relPath, err := filepath.Rel(dirName, fileName)
	if err != nil {
		return err
	}

	// Need to adjust README.md according to the extension config file name.
	// See https://github.com/tetratelabs/getenvoy/issues/124
	if relPath == "README.md" {
		var extensionConfigFileName string
		switch language {
		case extension.LanguageTinyGo:
			extensionConfigFileName = "extension.txt"
		default:
			extensionConfigFileName = "extension.json"
		}
		data = []byte(strings.ReplaceAll(string(data),
			"${EXTENSION_CONFIG_FILE_NAME}", extensionConfigFileName))
	}

	fileSet.Add(relPath, &model.File{Source: fileName, Content: data})
	return nil
}

func listFiles(f fs.FS, root string) ([]string, error) {
	fileNames := make([]string, 0)
	err := fs.WalkDir(f, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		fileNames = append(fileNames, path)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return fileNames, nil
}
