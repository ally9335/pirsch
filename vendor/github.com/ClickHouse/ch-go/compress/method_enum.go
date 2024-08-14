// Code generated by "enumer -transform upper -type Method -output method_enum.go"; DO NOT EDIT.

package compress

import (
	"fmt"
	"strings"
)

const _MethodName = "NONELZ4LZ4HCZSTD"

var _MethodIndex = [...]uint8{0, 4, 7, 12, 16}

const _MethodLowerName = "nonelz4lz4hczstd"

func (i Method) String() string {
	if i >= Method(len(_MethodIndex)-1) {
		return fmt.Sprintf("Method(%d)", i)
	}
	return _MethodName[_MethodIndex[i]:_MethodIndex[i+1]]
}

// An "invalid array index" compiler error signifies that the constant values have changed.
// Re-run the stringer command to generate them again.
func _MethodNoOp() {
	var x [1]struct{}
	_ = x[None-(0)]
	_ = x[LZ4-(1)]
	_ = x[LZ4HC-(2)]
	_ = x[ZSTD-(3)]
}

var _MethodValues = []Method{None, LZ4, LZ4HC, ZSTD}

var _MethodNameToValueMap = map[string]Method{
	_MethodName[0:4]:        None,
	_MethodLowerName[0:4]:   None,
	_MethodName[4:7]:        LZ4,
	_MethodLowerName[4:7]:   LZ4,
	_MethodName[7:12]:       LZ4HC,
	_MethodLowerName[7:12]:  LZ4HC,
	_MethodName[12:16]:      ZSTD,
	_MethodLowerName[12:16]: ZSTD,
}

var _MethodNames = []string{
	_MethodName[0:4],
	_MethodName[4:7],
	_MethodName[7:12],
	_MethodName[12:16],
}

// MethodString retrieves an enum value from the enum constants string name.
// Throws an error if the param is not part of the enum.
func MethodString(s string) (Method, error) {
	if val, ok := _MethodNameToValueMap[s]; ok {
		return val, nil
	}

	if val, ok := _MethodNameToValueMap[strings.ToLower(s)]; ok {
		return val, nil
	}
	return 0, fmt.Errorf("%s does not belong to Method values", s)
}

// MethodValues returns all values of the enum
func MethodValues() []Method {
	return _MethodValues
}

// MethodStrings returns a slice of all String values of the enum
func MethodStrings() []string {
	strs := make([]string, len(_MethodNames))
	copy(strs, _MethodNames)
	return strs
}

// IsAMethod returns "true" if the value is listed in the enum definition. "false" otherwise
func (i Method) IsAMethod() bool {
	for _, v := range _MethodValues {
		if i == v {
			return true
		}
	}
	return false
}
