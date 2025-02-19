package opaq

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Files returns a Source that reads policy files. The method gets the policy files that have `.rego` extension, and gets the policy files recursively from the given paths.
func Files(files ...string) Source {
	return func() (map[string]string, error) {
		policy := make(map[string]string)

		for _, file := range files {
			err := filepath.WalkDir(file, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.IsDir() {
					return nil
				}
				if filepath.Ext(path) != ".rego" {
					return nil
				}

				fpath := filepath.Clean(path)
				raw, err := os.ReadFile(fpath)
				if err != nil {
					return fmt.Errorf("failed to read policy file: %w", err)
				}

				policy[fpath] = string(raw)

				return nil
			})
			if err != nil {
				return nil, fmt.Errorf("failed to walk directory: %w", err)
			}
		}

		return policy, nil
	}
}

// Data returns a Source that have given data. The arguments are pairs of key and value.
// Example:
//
//	opaq.Data("pkg1", "package pkg1\n\nallow := true\n")
func Data(args ...string) Source {
	return func() (map[string]string, error) {
		data := make(map[string]string)
		for i := 0; i < len(args); i += 2 {
			if i+1 >= len(args) {
				return nil, fmt.Errorf("invalid number of arguments")
			}
			data[args[i]] = args[i+1]
		}
		return data, nil
	}
}

// DataMap returns a Source that have given data. The argument is a map of key and value.
// Example:
//
//	opaq.DataMap(map[string]string{
//		"pkg1": "package pkg1\n\nallow := true\n",
//	})
func DataMap(data map[string]string) Source {
	policy := make(map[string]string, len(data))
	for k, v := range data {
		policy[k] = v
	}
	return func() (map[string]string, error) {
		return policy, nil
	}
}
