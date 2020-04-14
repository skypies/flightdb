// Hacky approach to config, to keep secrets out of github; population of this
// config happens from local files synlinked into this dir of the buildtree.
package config

// Global constants ? Yes, global variables.
var theConfig = make(map[string]string)
func Set(key,val string) { theConfig[key] = val }
func Get(key string) string { return theConfig[key] }
