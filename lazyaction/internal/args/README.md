# args

## Variables

```golang
var (
    ErrNilAction     = fmt.Errorf("action is nil")
    ErrNonFuncAction = fmt.Errorf("action is not a function")
)
```

## Functions

### func [ExtractArgs](/args.go#L13)

`func ExtractArgs(t reflect.Type) (args, rets []string, err error)`

## Types

### type [ErrArgumentNotFound](/args.go#L79)

`type ErrArgumentNotFound string`

#### func (ErrArgumentNotFound) [Error](/args.go#L81)

`func (e ErrArgumentNotFound) Error() string`

### type [Fn](/args.go#L35)

`type Fn struct { ... }`

#### func [NewFn](/args.go#L149)

`func NewFn(v any) *Fn`

#### func (Fn) [Call](/args.go#L85)

`func (f Fn) Call(inputs InputSet) (o []reflect.Value, err error)`

### type [Gen](/args.go#L46)

`type Gen Fn`

#### func [NewGen](/args.go#L63)

`func NewGen(v any) Gen`

#### func (Gen) [Call](/args.go#L48)

`func (g Gen) Call(inputs InputSet) (o []reflect.Value, err error)`

### type [InputSet](/args.go#L41)

`type InputSet struct { ... }`

#### func [OutsToInputs](/args.go#L168)

`func OutsToInputs(outs []reflect.Value) InputSet`

#### func (InputSet) [Merge](/args.go#L179)

`func (is InputSet) Merge(is2 InputSet) InputSet`

### type [OutputSet](/args.go#L144)

`type OutputSet struct { ... }`

---
Readme created from Go doc with [goreadme](https://github.com/posener/goreadme)
