package util

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

const alphabet = "abcdefghijklmnopqrstuvwxyz"

// will be called automatically when the package is first used
func init() {
	rand.Seed(time.Now().UnixNano()) // set the seed value for the random generator - often uses the current time which we are converting with UnixNano to make it an int64
}

// RandomInt will generate a random integer between min and max
func RandomInt(min, max int64) int64 {
	return min + rand.Int63n(max-min+1) // Int63n returns a pseudo random number between 0 inclusive and n not inclusive
}

// RandomString will generate a random string of n characters
func RandomString(n int) string {
	var sb strings.Builder
	k := len(alphabet) // alphabet is the const declared above

	for i := 0; i < n; i++ { // randomly generating a string of length n
		c := alphabet[rand.Intn(k)] // Intn returns a pseudo random number from 0 inclusive to k not inclusive
		sb.WriteByte(c)             // WriteByte writes the character to the string builder
	}

	return sb.String() // returns the accumulated string from the strings builder

}

// RandomOwner will generate a random owner name
func RandomOwner() string {
	return RandomString(6) // we use six characters to reasonably avoid duplicates
}

// RandomMoney will generate a random amount of money
func RandomMoney() int64 {
	return RandomInt(0, 1000) // arbitrary min and max
}

// RandomCurrency will generate a random currency code
func RandomCurrency() string {
	currencies := []string{USD, EUR, CAD} // arbitrary currency codes - constants declared in currency.go
	n := len(currencies)
	return currencies[rand.Intn(n)] // will return a pseudo random index from 0 inclusive to n not inclusive
}

// RandomEmail will generate a random email
func RandomEmail() string {
	return fmt.Sprintf("%s@email.com", RandomString(6))
}
