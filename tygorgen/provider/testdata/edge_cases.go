package testdata

// AllBasicTypes has all basic type variations
type AllBasicTypes struct {
	Bool    bool    `json:"bool"`
	Int     int     `json:"int"`
	Int8    int8    `json:"int8"`
	Int16   int16   `json:"int16"`
	Int32   int32   `json:"int32"`
	Int64   int64   `json:"int64"`
	Uint    uint    `json:"uint"`
	Uint8   uint8   `json:"uint8"`
	Uint16  uint16  `json:"uint16"`
	Uint32  uint32  `json:"uint32"`
	Uint64  uint64  `json:"uint64"`
	Uintptr uintptr `json:"uintptr"`
	Float32 float32 `json:"float32"`
	Float64 float64 `json:"float64"`
	String  string  `json:"string"`
}

// InvalidMapKeys tests invalid map key types
type InvalidMapKeys struct {
	// These are invalid and should cause errors
	// BoolMap     map[bool]string     `json:"bool_map"`
	// FloatMap    map[float64]string  `json:"float_map"`
	// ComplexMap  map[complex64]string `json:"complex_map"`
}

// EnumFloat is a float-based enum
type EnumFloat float64

const (
	FloatZero EnumFloat = 0.0
	FloatHalf EnumFloat = 0.5
	FloatOne  EnumFloat = 1.0
)

// EnumBool is a bool-based enum
type EnumBool bool

const (
	BoolFalse EnumBool = false
	BoolTrue  EnumBool = true
)
