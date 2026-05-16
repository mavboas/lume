# Lume Language Reference

This document describes the current public language surface and labels future ideas separately. The compiler is experimental and implements only the subset covered by the examples and tests.

## Current Compiler Subset

The v0 compiler currently supports:

- Basic values: `int`, `dec`, `str`, `bool`, `none`, lists, objects, and class instances.
- Inferred and annotated bindings.
- Same-scope rebinding as a new value, not mutation.
- Functions with typed parameters, optional return types, optional doc strings, and final-expression returns.
- Arithmetic, comparison, logical operators, and string interpolation.
- `if/elsif/else` statements.
- `switch/case/default` over literal values.
- `for(name in list)` and `for(i in 0..10)` loops.
- Classes declared with `cl`, named constructors, field access, `.with()`, and `.keys`.
- `let(name = expr, ...){ body }` expressions with sequential local bindings.
- Basic `match(value){ case(pattern){ body } }` with literals, `_`, and capture patterns.
- Calls to declared functions and the builtin `print`.

## Values and Bindings

```lume
fn main()
{
    age = 30
    name = "Jane"
    active = true
    user: obj = {id: 1, name: "Jane"}

    age = age + 1
    print("${name}: ${age}")
}
```

`=` introduces or rebinds a name. Class fields and object fields are not assigned through field mutation in the current subset.

## Functions

```lume
fn add(a: int, b: int) -> int
{
    a + b
}

fn main()
{
    print(add(20, 22))
}
```

Functions return the value of the final expression when a return type is declared.

## Lists and Objects

```lume
fn main()
{
    nums: list[int] = [1, 2, 3]
    empty: list[str] = []
    user: obj = {name: "Jane", age: 30}

    print(nums)
    print(empty)
    print(user.name)
}
```

Empty lists need an explicit type annotation.

## Classes

```lume
cl Point
{
    x: int
    y: int
}

fn main()
{
    p = Point(x= 3, y= 4)
    moved = p.with(y= 10)

    print(p.x)
    print(moved.y)
    print(p.keys)
}
```

Constructors use named fields. `.with()` derives a new class value with selected fields replaced.

## Control Flow

```lume
fn label(x: int) -> str
{
    if(x < 0){
        "negative"
    } elsif(x == 0){
        "zero"
    } else {
        "positive"
    }
}

fn classify(x: str) -> str
{
    switch(x){
        case("1"){
            "one"
        }
        case("2"){
            "two"
        }
        default(){
            "other"
        }
    }
}
```

Use `switch` when comparing one value against literal cases.

## Loops

```lume
fn main()
{
    xs = [1, 2, 3]

    for(x in xs){
        print(x)
    }

    for(i in 0..10){
        print(i)
    }
}
```

Integer ranges are half-open: `0..10` includes `0` and excludes `10`.

## Let Expressions

```lume
fn subtotal(price: int, qty: int) -> int
{
    let(base = price * qty, fee = 2){
        base + fee
    }
}
```

Bindings inside `let` are sequential and scoped to the block.

## Match

```lume
fn classify(n: int) -> str
{
    match(n){
        case(0){
            "zero"
        }
        case(value){
            "other: ${value}"
        }
    }
}
```

The current compiler supports literal patterns, wildcard `_`, and identifier capture patterns. Value-producing match expressions must be exhaustive.

## Planned Language Ideas

These are target-language ideas and are not implemented in the current compiler unless explicitly listed above:

- Full Hindley-Milner inference.
- ADTs and exhaustive matching over custom unions.
- Destructuring in `let` and patterns.
- `Result`, `Option`, and `?` propagation.
- Pipe operator `|>`.
- Modules and imports.
- Lambdas.
- Effect annotations.
- Refinement types.
- Spec blocks.
- A standard library for backend work.
