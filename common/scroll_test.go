package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"infini.sh/framework/core/elastic"
)

func TestEnsureExactScrollTotalHits(t *testing.T) {
	query := EnsureExactScrollTotalHits(elastic.Version{Distribution: elastic.Elasticsearch, Major: 7}, nil)
	assert.NotNil(t, query)
	assert.Contains(t, query.ToJSONString(), "\"track_total_hits\":true")
}

func TestEnsureExactScrollTotalHitsSkipLegacyElasticsearch(t *testing.T) {
	query := EnsureExactScrollTotalHits(elastic.Version{Distribution: elastic.Elasticsearch, Major: 6}, nil)
	assert.Nil(t, query)
}
