package opaq_test

import (
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/opaq"
)

func TestSource(t *testing.T) {
	t.Run("Files", func(t *testing.T) {
		client, err := opaq.New(opaq.Files("testdata/local"))
		gt.NoError(t, err)
		data := client.Sources()
		gt.Map(t, data).
			Length(2).
			HasKey("testdata/local/f1.rego").
			HasKey("testdata/local/f2.rego")
	})

	t.Run("Files with invalid args", func(t *testing.T) {
		_, err := opaq.New(opaq.Files("not_exists"))
		gt.Error(t, err)
	})

	t.Run("Data", func(t *testing.T) {
		client, err := opaq.New(opaq.Data(
			"key1", "package key1\nallow = true",
			"key2", "package key2\nallow = true",
		))
		gt.NoError(t, err)
		data := client.Sources()
		gt.Map(t, data).
			Length(2).
			HasKey("key1").
			HasKey("key2")
	})

	t.Run("Data with invalid args", func(t *testing.T) {
		_, err := opaq.New(opaq.Data("key1"))
		gt.Error(t, err)
	})

	t.Run("DataMap", func(t *testing.T) {
		input := map[string]string{
			"key1": "package key1\nallow = true",
			"key2": "package key2\nallow = true",
		}
		client, err := opaq.New(opaq.DataMap(input))
		gt.NoError(t, err)
		data := client.Sources()
		gt.Map(t, data).
			Length(2).
			HasKey("key1").
			HasKey("key2")
	})
}
