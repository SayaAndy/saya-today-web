package router

import "github.com/dgraph-io/ristretto/v2"

var PCache *ristretto.Cache[string, []byte]
