package lazysupport

import (
	"fmt"
)

func ExamplePluralize() {
	fmt.Println(Pluralize("post"))
	fmt.Println(Pluralize("octopus"))
	fmt.Println(Pluralize("sheep"))
	fmt.Println(Pluralize("words"))
	fmt.Println(Pluralize("CamelOctopus"))
	// Output:
	// posts
	// octopi
	// sheep
	// words
	// CamelOctopi
}

func ExampleSingularize() {
	fmt.Println(Singularize("posts"))
	fmt.Println(Singularize("octopi"))
	fmt.Println(Singularize("sheep"))
	fmt.Println(Singularize("word"))
	fmt.Println(Singularize("CamelOctopi"))
	// Output:
	// post
	// octopus
	// sheep
	// word
	// CamelOctopus
}

func ExampleCamelize() {
	fmt.Println(Camelize("my_account"))
	fmt.Println(Camelize("user-profile"))
	fmt.Println(Camelize("ssl_error"))
	fmt.Println(Camelize("http_connection_timeout"))
	fmt.Println(Camelize("restful_controller"))
	fmt.Println(Camelize("multiple_http_calls"))
	// Output:
	// MyAccount
	// UserProfile
	// SSLError
	// HTTPConnectionTimeout
	// RESTfulController
	// MultipleHTTPCalls
}

func ExampleUnderscorize() {
	fmt.Println(Underscorize("MyAccount"))
	fmt.Println(Underscorize("user-profile"))
	fmt.Println(Underscorize("SSLError"))
	fmt.Println(Underscorize("HTTPConnectionTimeout"))
	fmt.Println(Underscorize("RESTfulController"))
	fmt.Println(Underscorize("MultipleHTTPCalls"))
	// Output:
	// my_account
	// user_profile
	// ssl_error
	// http_connection_timeout
	// restful_controller
	// multiple_http_calls
}

func ExampleDasherize() {
	fmt.Println(Dasherize("MyAccount"))
	fmt.Println(Dasherize("user_profile"))
	// Output:
	// my-account
	// user-profile
}

func ExampleTableize() {
	fmt.Println(Tableize("RawScaledScorer"))
	fmt.Println(Tableize("ham_and_egg"))
	fmt.Println(Tableize("fancyCategory"))
	// Output:
	// raw_scaled_scorers
	// ham_and_eggs
	// fancy_categories
}

func ExampleForeignKey() {
	fmt.Println(ForeignKey("Message"))
	fmt.Println(ForeignKey("AdminPost"))
	// Output:
	// message_id
	// admin_post_id
}

func ExampleOrdinal() {
	fmt.Println(Ordinal(1))
	fmt.Println(Ordinal(2))
	fmt.Println(Ordinal(14))
	fmt.Println(Ordinal(1002))
	fmt.Println(Ordinal(1003))
	fmt.Println(Ordinal(-11))
	fmt.Println(Ordinal(-1021))
	// Output:
	// st
	// nd
	// th
	// nd
	// rd
	// th
	// st
}

func ExampleOrdinalize() {
	fmt.Println(Ordinalize(1))
	fmt.Println(Ordinalize(2))
	fmt.Println(Ordinalize(14))
	fmt.Println(Ordinalize(1002))
	fmt.Println(Ordinalize(1003))
	fmt.Println(Ordinalize(-11))
	fmt.Println(Ordinalize(-1021))
	// Output:
	// 1st
	// 2nd
	// 14th
	// 1002nd
	// 1003rd
	// -11th
	// -1021st
}
