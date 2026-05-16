package sema

// DiagnosticInfo is stable, AI-readable metadata for a semantic diagnostic.
type DiagnosticInfo struct {
	Code    string `json:"code"`
	Feature string `json:"feature"`
	Message string `json:"message"`
	FixHint string `json:"fix_hint"`
}

// DiagnosticCatalog lists semantic diagnostics with short repair hints.
func DiagnosticCatalog() []DiagnosticInfo {
	return []DiagnosticInfo{
		{Code: "E1001", Feature: "functions", Message: "function is already declared", FixHint: "rename one declaration or merge the duplicated function"},
		{Code: "E1002", Feature: "class", Message: "class is already declared", FixHint: "rename one declaration or keep a single class definition"},
		{Code: "E1101", Feature: "types", Message: "unknown type", FixHint: "use a builtin type or declare the class before referencing it"},
		{Code: "E1102", Feature: "types", Message: "type list expects exactly one type argument", FixHint: "write list[T] with one element type, for example list[int]"},
		{Code: "E2001", Feature: "functions", Message: "function return type mismatch", FixHint: "return a value assignable to the declared function return type"},
		{Code: "E2002", Feature: "let", Message: "binding type mismatch", FixHint: "change the annotation or assign a value of the annotated type"},
		{Code: "E2101", Feature: "if", Message: "condition must be bool", FixHint: "use a boolean expression in if or elsif conditions"},
		{Code: "E2102", Feature: "if", Message: "if branches return incompatible types", FixHint: "make all value-producing branches return the same type"},
		{Code: "E2103", Feature: "switch", Message: "switch case value must be a literal", FixHint: "replace the case expression with an int, dec, str, bool, or none literal"},
		{Code: "E2104", Feature: "switch", Message: "switch case type mismatch", FixHint: "make every case literal match the switch value type"},
		{Code: "E2105", Feature: "switch", Message: "switch branches return incompatible types", FixHint: "make all value-producing switch branches return the same type"},
		{Code: "E2106", Feature: "for", Message: "for target must be list or int range", FixHint: "iterate over a list or a half-open integer range like 0..10"},
		{Code: "E2107", Feature: "for", Message: "range bounds must be int", FixHint: "use int expressions on both sides of the range operator"},
		{Code: "E2201", Feature: "scopes", Message: "unknown identifier", FixHint: "declare the name before use or check the spelling and local scope"},
		{Code: "E2301", Feature: "operators", Message: "operator - expects int or dec", FixHint: "apply unary minus only to numeric values"},
		{Code: "E2302", Feature: "operators", Message: "operator ! expects bool", FixHint: "apply logical not only to boolean values"},
		{Code: "E2401", Feature: "operators", Message: "arithmetic operator type mismatch", FixHint: "use matching numeric operands, or strings for + concatenation"},
		{Code: "E2402", Feature: "operators", Message: "comparison requires matching types", FixHint: "compare values with compatible types"},
		{Code: "E2403", Feature: "operators", Message: "ordered comparison requires matching numeric types", FixHint: "use < <= > >= only with matching numeric operands"},
		{Code: "E2404", Feature: "operators", Message: "logical operator expects bool operands", FixHint: "use && and || only with boolean operands"},
		{Code: "E2501", Feature: "calls", Message: "unsupported call target or method", FixHint: "call a named function, class constructor, or the supported .with() method"},
		{Code: "E2502", Feature: "calls", Message: "unknown function", FixHint: "declare the function before calling it or use an existing builtin"},
		{Code: "E2503", Feature: "calls", Message: "function argument count mismatch", FixHint: "pass exactly the parameters declared by the function signature"},
		{Code: "E2504", Feature: "calls", Message: "argument or field type mismatch", FixHint: "pass a value assignable to the expected parameter or field type"},
		{Code: "E2505", Feature: "class", Message: "invalid named argument or missing class field", FixHint: "use named arguments only for class fields and constructors"},
		{Code: "E2506", Feature: "class", Message: "constructor is missing a field", FixHint: "provide every class field in the constructor call"},
		{Code: "E2507", Feature: "class", Message: "duplicate constructor or .with field", FixHint: "specify each named field at most once"},
		{Code: "E2601", Feature: "lists", Message: "empty list literal needs an explicit type annotation", FixHint: "annotate the binding, for example xs: list[int] = []"},
		{Code: "E2602", Feature: "lists", Message: "list elements must have one type", FixHint: "make all list elements assignable to the same element type"},
		{Code: "E2701", Feature: "objects", Message: "object field is already defined", FixHint: "remove or rename the duplicate object field"},
		{Code: "E2702", Feature: "objects", Message: "field access expects obj or class instance", FixHint: "access fields only on obj values or class instances"},
		{Code: "E2703", Feature: "class", Message: "class has no field", FixHint: "use an existing class field or add the field to the class declaration"},
		{Code: "E2801", Feature: "match", Message: "match pattern type mismatch", FixHint: "make literal patterns match the input value type"},
		{Code: "E2802", Feature: "match", Message: "unsupported match pattern", FixHint: "use a literal, wildcard _, or identifier capture pattern"},
		{Code: "E2803", Feature: "match", Message: "unreachable match case", FixHint: "put wildcard or capture patterns last"},
		{Code: "E2804", Feature: "match", Message: "match branches return incompatible types", FixHint: "make every value-producing match branch return the same type"},
		{Code: "E2805", Feature: "match", Message: "match expression is not exhaustive", FixHint: "add case(_) or cover true/false for bool"},
	}
}
