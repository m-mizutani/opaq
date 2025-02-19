package opaq_test

import (
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/opaq"
)

func TestSource(t *testing.T) {
	t.Run("Files", func(t *testing.T) {
		src := opaq.Files("testdata/local")
		data, err := src()
		gt.NoError(t, err)
		gt.Map(t, data).
			Length(2).
			HaveKey("testdata/local/f1.rego").
			HaveKey("testdata/local/f2.rego")
	})

	t.Run("Files with invalid args", func(t *testing.T) {
		src := opaq.Files("not_exists")
		_, err := src()
		gt.Error(t, err)
	})

	t.Run("Data", func(t *testing.T) {
		src := opaq.Data("key1", "value1", "key2", "value2")
		data, err := src()
		gt.NoError(t, err)
		gt.Map(t, data).
			Length(2).
			HaveKey("key1").
			HaveKey("key2")
	})

	t.Run("Data with invalid args", func(t *testing.T) {
		src := opaq.Data("key1")
		_, err := src()
		gt.Error(t, err)
	})

	t.Run("DataMap", func(t *testing.T) {
		input := map[string]string{
			"key1": "value1",
			"key2": "value2",
		}
		src := opaq.DataMap(input)
		data, err := src()
		gt.NoError(t, err)
		gt.Map(t, data).
			Length(2).
			HaveKey("key1").
			HaveKey("key2")
	})
}
