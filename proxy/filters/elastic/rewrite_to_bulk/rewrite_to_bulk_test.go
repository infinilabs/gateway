/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package rewrite_to_bulk

import (
	"fmt"
	"github.com/magiconair/properties/assert"
	"testing"
)

func TestParseURLMeta(t *testing.T) {
	url:="/index/_update/id"
	valid, indexPath, typePath, idPath :=ParseURLMeta(url)
	fmt.Println(valid, indexPath, typePath, idPath)
	assert.Equal(t, valid, true)
	assert.Equal(t, indexPath, "index")
	assert.Equal(t, typePath, "_update")
	assert.Equal(t,idPath, "id")
}