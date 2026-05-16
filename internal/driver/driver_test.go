package driver

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompileExamples(t *testing.T) {
	examples := []string{"hello.lm", "math.lm", "classify.lm", "lists.lm", "objects.lm", "classes.lm", "switch.lm", "for.lm", "let.lm", "match.lm"}
	for _, name := range examples {
		t.Run(name, func(t *testing.T) {
			res, err := CompileFile(filepath.Join("..", "..", "examples", name))
			if err != nil {
				t.Fatalf("CompileFile failed: %v", err)
			}
			if res.GoSource == "" {
				t.Fatal("expected generated Go source")
			}
			if !strings.Contains(res.GoSource, "package main") {
				t.Fatalf("generated source does not look like Go:\n%s", res.GoSource)
			}
		})
	}
}

func TestCompileRebindingUsesGoAssignment(t *testing.T) {
	src := `
fn main()
{
    x = 1
    x = x + 1
    print(x)
}
`
	path := filepath.Join(t.TempDir(), "rebind.lm")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := CompileFile(path)
	if err != nil {
		t.Fatalf("CompileFile failed: %v", err)
	}
	if !strings.Contains(res.GoSource, "x = (x + int64(1))") {
		t.Fatalf("expected rebinding to use assignment, got:\n%s", res.GoSource)
	}
}

func TestCompileLists(t *testing.T) {
	src := `
fn main()
{
    xs: list[int] = [1, 2, 3]
    empty: list[str] = []
    print(xs)
    print(empty)
}
`
	path := filepath.Join(t.TempDir(), "lists.lm")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := CompileFile(path)
	if err != nil {
		t.Fatalf("CompileFile failed: %v", err)
	}
	if !strings.Contains(res.GoSource, "var xs []int64 = []int64{int64(1), int64(2), int64(3)}") {
		t.Fatalf("expected typed int list, got:\n%s", res.GoSource)
	}
	if !strings.Contains(res.GoSource, "var empty []string = nil") {
		t.Fatalf("expected typed empty list, got:\n%s", res.GoSource)
	}
}

func TestCompileObjects(t *testing.T) {
	src := `
fn main()
{
    user: obj = {name: "Jane", age: 30}
    print(user.name)
}
`
	path := filepath.Join(t.TempDir(), "objects.lm")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := CompileFile(path)
	if err != nil {
		t.Fatalf("CompileFile failed: %v", err)
	}
	if !strings.Contains(res.GoSource, `var user map[string]any = map[string]any{"name": "Jane", "age": int64(30)}`) {
		t.Fatalf("expected typed object, got:\n%s", res.GoSource)
	}
	if !strings.Contains(res.GoSource, `fmt.Println(user["name"])`) {
		t.Fatalf("expected field access, got:\n%s", res.GoSource)
	}
}

func TestCompileClasses(t *testing.T) {
	src := `
cl Point
{
    x: int
    y: int
}

fn main()
{
    p = Point(x= 3, y= 4)
    p2 = p.with(y= 10)
    print(p.x)
    print(p2.y)
}
`
	path := filepath.Join(t.TempDir(), "classes.lm")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := CompileFile(path)
	if err != nil {
		t.Fatalf("CompileFile failed: %v\nGenerated:\n%s", err, res.GoSource)
	}
	if !strings.Contains(res.GoSource, "type Point struct") {
		t.Fatalf("expected Go struct, got:\n%s", res.GoSource)
	}
	if !strings.Contains(res.GoSource, "Point{X: int64(3), Y: int64(4)}") {
		t.Fatalf("expected constructor call, got:\n%s", res.GoSource)
	}
	if !strings.Contains(res.GoSource, "Point{X: p.X, Y: int64(10)}") {
		t.Fatalf("expected .with() call, got:\n%s", res.GoSource)
	}
	if !strings.Contains(res.GoSource, "fmt.Println(p.X)") {
		t.Fatalf("expected field access, got:\n%s", res.GoSource)
	}
}

func TestCompileSwitch(t *testing.T) {
	src := `
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

fn main()
{
    print(classify("2"))
}
`
	path := filepath.Join(t.TempDir(), "switch.lm")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := CompileFile(path)
	if err != nil {
		t.Fatalf("CompileFile failed: %v\nGenerated:\n%s", err, res.GoSource)
	}
	if !strings.Contains(res.GoSource, "switch x") {
		t.Fatalf("expected Go switch, got:\n%s", res.GoSource)
	}
	if !strings.Contains(res.GoSource, `case "2":`) {
		t.Fatalf("expected Go case, got:\n%s", res.GoSource)
	}
	if !strings.Contains(res.GoSource, "default:") {
		t.Fatalf("expected Go default, got:\n%s", res.GoSource)
	}
	if !strings.Contains(res.GoSource, `return "two"`) {
		t.Fatalf("expected switch branches to return values, got:\n%s", res.GoSource)
	}
}

func TestCompileForLoops(t *testing.T) {
	src := `
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
`
	path := filepath.Join(t.TempDir(), "for.lm")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := CompileFile(path)
	if err != nil {
		t.Fatalf("CompileFile failed: %v\nGenerated:\n%s", err, res.GoSource)
	}
	if !strings.Contains(res.GoSource, "for _, x := range xs") {
		t.Fatalf("expected Go range loop, got:\n%s", res.GoSource)
	}
	if !strings.Contains(res.GoSource, "for i := int64(0); i < int64(10); i++") {
		t.Fatalf("expected counted range loop, got:\n%s", res.GoSource)
	}
}

func TestCompileLetExpression(t *testing.T) {
	src := `
fn calc() -> int
{
    let(x = 2, y = x * 3){
        y + 1
    }
}

fn main()
{
    print(calc())
}
`
	path := filepath.Join(t.TempDir(), "let.lm")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := CompileFile(path)
	if err != nil {
		t.Fatalf("CompileFile failed: %v\nGenerated:\n%s", err, res.GoSource)
	}
	if !strings.Contains(res.GoSource, "func() int64") {
		t.Fatalf("expected let IIFE with int64 return, got:\n%s", res.GoSource)
	}
	if !strings.Contains(res.GoSource, "return (y + int64(1))") {
		t.Fatalf("expected let body return, got:\n%s", res.GoSource)
	}
}

func TestCompileMatchExpressionAndStatement(t *testing.T) {
	src := `
fn classify(x: int) -> str
{
    match(x){
        case(0){ "zero" }
        case(n){ "value: ${n}" }
    }
}

fn main()
{
    match(classify(0)){
        case("zero"){ print("z") }
        case(_){ print("other") }
    }
}
`
	path := filepath.Join(t.TempDir(), "match.lm")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := CompileFile(path)
	if err != nil {
		t.Fatalf("CompileFile failed: %v\nGenerated:\n%s", err, res.GoSource)
	}
	if !strings.Contains(res.GoSource, "func() string") {
		t.Fatalf("expected match expression IIFE, got:\n%s", res.GoSource)
	}
	if !strings.Contains(res.GoSource, "switch __match") {
		t.Fatalf("expected generated switch, got:\n%s", res.GoSource)
	}
	if !strings.Contains(res.GoSource, "n := __match") {
		t.Fatalf("expected capture binding, got:\n%s", res.GoSource)
	}
}
