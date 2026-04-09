// pkg/wispy/value.go
package grove

import "github.com/wispberry-tech/grove/internal/vm"

// Value is the template runtime value type.
type Value = vm.Value

// StringValue wraps a Go string as a Value.
func StringValue(s string) Value { return vm.StringVal(s) }

// SafeHTMLValue wraps trusted HTML as a Value — auto-escape is skipped on output.
func SafeHTMLValue(s string) Value { return vm.SafeHTMLVal(s) }

// ArgInt reads args[i] as an integer, returning def if i is out of range.
func ArgInt(args []Value, i, def int) int { return vm.ArgInt(args, i, def) }
