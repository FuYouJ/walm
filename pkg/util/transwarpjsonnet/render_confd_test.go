package transwarpjsonnet

import (
	"WarpCloud/walm/pkg/util/transwarpjsonnet/memkv"
	"fmt"
	"github.com/stretchr/testify/assert"
	"path"
	"testing"
)

func Test_Memkv(t *testing.T) {
	vars := make(map[string]string)
	store := memkv.New()

	yamlMap := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	err := nodeWalk(yamlMap, "", vars)
	assert.Nil(t, err)

	for k, v := range vars {
		store.Set(path.Join("/", k), v)
	}

	fmt.Printf("%v", vars)
}
