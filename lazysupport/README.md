# lazysupport

## Variables

```golang
var DefaultCache = MemCache{}
```

ShouldCache set if the inflector should (or not) cache the inflections

```golang
var ShouldCache = false
```

## Functions

### func [Cache](/cache.go#L21)

`func Cache(fn func() ([]byte, error), key ...any) ([]byte, error)`

### func [Camelize](/inflector.go#L147)

`func Camelize(term string) string`

Camelize converts strings to UpperCamelCase.

```golang
fmt.Println(Camelize("my_account"))
fmt.Println(Camelize("user-profile"))
fmt.Println(Camelize("ssl_error"))
fmt.Println(Camelize("http_connection_timeout"))
fmt.Println(Camelize("restful_controller"))
fmt.Println(Camelize("multiple_http_calls"))
```

 Output:

```
MyAccount
UserProfile
SSLError
HTTPConnectionTimeout
RESTfulController
MultipleHTTPCalls
```

### func [ClearCache](/inflector.go#L41)

`func ClearCache()`

ClearCache clear the inflection cache. Both for singulars and plurals.

### func [Dasherize](/inflector.go#L173)

`func Dasherize(term string) string`

Dasherize converts strings to dashed, lowercase form.

```golang
fmt.Println(Dasherize("MyAccount"))
fmt.Println(Dasherize("user_profile"))
```

 Output:

```
my-account
user-profile
```

### func [ForeignKey](/inflector.go#L183)

`func ForeignKey(term string) string`

ForeignKey creates a foreign key name from an ORM model name.

```golang
fmt.Println(ForeignKey("Message"))
fmt.Println(ForeignKey("AdminPost"))
```

 Output:

```
message_id
admin_post_id
```

### func [Ordinal](/inflector.go#L189)

`func Ordinal(number int64) string`

Ordinal returns the suffix that should be added to a number to denote the position
in an ordered sequence such as 1st, 2nd, 3rd, 4th.

```golang
fmt.Println(Ordinal(1))
fmt.Println(Ordinal(2))
fmt.Println(Ordinal(14))
fmt.Println(Ordinal(1002))
fmt.Println(Ordinal(1003))
fmt.Println(Ordinal(-11))
fmt.Println(Ordinal(-1021))
```

 Output:

```
st
nd
th
nd
rd
th
st
```

### func [Ordinalize](/inflector.go#L210)

`func Ordinalize(number int64) string`

Ordinalize turns a number into an ordinal string used to denote the position in an
ordered sequence such as 1st, 2nd, 3rd, 4th.

```golang
fmt.Println(Ordinalize(1))
fmt.Println(Ordinalize(2))
fmt.Println(Ordinalize(14))
fmt.Println(Ordinalize(1002))
fmt.Println(Ordinalize(1003))
fmt.Println(Ordinalize(-11))
fmt.Println(Ordinalize(-1021))
```

 Output:

```
1st
2nd
14th
1002nd
1003rd
-11th
-1021st
```

### func [Pluralize](/inflector.go#L131)

`func Pluralize(singular string) string`

Pluralize returns the plural form of the word in the string.

```golang
fmt.Println(Pluralize("post"))
fmt.Println(Pluralize("octopus"))
fmt.Println(Pluralize("sheep"))
fmt.Println(Pluralize("words"))
fmt.Println(Pluralize("CamelOctopus"))
```

 Output:

```
posts
octopi
sheep
words
CamelOctopi
```

### func [Singularize](/inflector.go#L139)

`func Singularize(plural string) string`

Singularize returns the singular form of a word in a string.

```golang
fmt.Println(Singularize("posts"))
fmt.Println(Singularize("octopi"))
fmt.Println(Singularize("sheep"))
fmt.Println(Singularize("word"))
fmt.Println(Singularize("CamelOctopi"))
```

 Output:

```
post
octopus
sheep
word
CamelOctopus
```

### func [Tableize](/inflector.go#L178)

`func Tableize(term string) string`

Tableize creates the name of a table for an ORM model.

```golang
fmt.Println(Tableize("RawScaledScorer"))
fmt.Println(Tableize("ham_and_egg"))
fmt.Println(Tableize("fancyCategory"))
```

 Output:

```
raw_scaled_scorers
ham_and_eggs
fancy_categories
```

### func [ToSentence](/sentence.go#L5)

`func ToSentence(last_join string, parts ...string) string`

### func [ToSnakeCase](/naming.go#L11)

`func ToSnakeCase(str string) string`

### func [Underscorize](/inflector.go#L158)

`func Underscorize(term string) string`

Underscorize converts strings to underscored, lowercase form.

```golang
fmt.Println(Underscorize("MyAccount"))
fmt.Println(Underscorize("user-profile"))
fmt.Println(Underscorize("SSLError"))
fmt.Println(Underscorize("HTTPConnectionTimeout"))
fmt.Println(Underscorize("RESTfulController"))
fmt.Println(Underscorize("MultipleHTTPCalls"))
```

 Output:

```
my_account
user_profile
ssl_error
http_connection_timeout
restful_controller
multiple_http_calls
```

## Types

### type [MemCache](/cache.go#L3)

`type MemCache map[any][]byte`

#### func (MemCache) [Cache](/cache.go#L5)

`func (c MemCache) Cache(fn func() ([]byte, error), key ...any) ([]byte, error)`

### type [Set](/set.go#L5)

`type Set[T comparable] map[T]Void`

#### func [NewSet](/set.go#L7)

`func NewSet[T comparable](values ...T) Set[T]`

#### func (Set[T]) [Has](/set.go#L15)

`func (s Set[T]) Has(item T) bool`

### type [Strings](/strings.go#L5)

`type Strings Set[string]`

#### func [NewStringSet](/strings.go#L30)

`func NewStringSet(s ...string) Strings`

#### func (Strings) [Has](/strings.go#L25)

`func (s Strings) Has(what string) bool`

#### func (Strings) [HasPrefix](/strings.go#L16)

`func (s Strings) HasPrefix(what string) bool`

#### func (Strings) [TrimPrefix](/strings.go#L7)

`func (s Strings) TrimPrefix(what string) (prefix, trimmed string)`

### type [Table](/table.go#L9)

`type Table struct { ... }`

```golang
table := Table{
    Header: []string{"Title", "Age"},
    Values: [][]string{
        {"The film", "1"},
        {"The super super file"},
        {"", ""},
        {"Other Filem", "123123"},
    },
}

fmt.Println(table.String())
```

 Output:

```
Title                Age
The film             1
The super super file

Other Filem          123123
```

#### func (*Table) [String](/table.go#L40)

`func (t *Table) String() string`

### type [Void](/set.go#L3)

`type Void struct{ ... }`

## Sub Packages

* [log](./log)

* [reflect](./reflect)

---
Readme created from Go doc with [goreadme](https://github.com/posener/goreadme)
