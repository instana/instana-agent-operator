/*
 * (c) Copyright IBM Corp. 2025
 */

package testmocks

import (
	"github.com/stretchr/testify/mock"
)

// AnyContext is a matcher for any context
var AnyContext = mock.MatchedBy(func(ctx interface{}) bool {
	return true
})

// AnyObject is a matcher for any client.Object
var AnyObject = mock.MatchedBy(func(obj interface{}) bool {
	return true
})

// AnyPatchOptions is a matcher for any slice of client.PatchOption
var AnyPatchOptions = mock.MatchedBy(func(opts interface{}) bool {
	return true
})

// AnyGetOptions is a matcher for any slice of client.GetOption
var AnyGetOptions = mock.MatchedBy(func(opts interface{}) bool {
	return true
})

// AnyDeleteOptions is a matcher for any slice of client.DeleteOption
var AnyDeleteOptions = mock.MatchedBy(func(opts interface{}) bool {
	return true
})

// Made with Bob
