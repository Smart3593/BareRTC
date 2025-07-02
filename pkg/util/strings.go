package util

import "math/rand"

// RandomString returns a random string of any length.
func RandomString(n int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz"
	var result = make([]byte, n)
	for i := 0; i < n; i++ {
		result[i] = charset[rand.Intn(len(charset))]
	}
	return string(result)
}
