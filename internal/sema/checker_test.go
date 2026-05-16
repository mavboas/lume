package sema

import (
	"strings"
	"testing"

	"github.com/mavboas/lume/internal/lexer"
	"github.com/mavboas/lume/internal/parser"
)

func parseAndCheck(t *testing.T, src string) error {
	t.Helper()
	toks, lexErrs := lexer.New(src).Tokenize()
	if len(lexErrs) > 0 {
		t.Fatalf("lex errors: %v", lexErrs)
	}
	tree, parseErrs := parser.New(toks).Parse()
	if len(parseErrs) > 0 {
		t.Fatalf("parse errors: %v", parseErrs)
	}
	return Check(tree)
}

func TestCheckAcceptsCurrentCoreLanguage(t *testing.T) {
	src := `
fn add(a: int, b: int) -> int
{
    a + b
}

fn main()
{
    x = 10
    x = x + 1
    print("x: ${add(x, 31)}")
}
`
	if err := parseAndCheck(t, src); err != nil {
		t.Fatalf("expected program to pass semantic checks: %v", err)
	}
}

func TestCheckRejectsBadArgumentType(t *testing.T) {
	err := parseAndCheck(t, `
fn add(a: int, b: int) -> int
{
    a + b
}

fn main()
{
    add("x", 1)
}
`)
	if err == nil || !strings.Contains(err.Error(), "E2504") {
		t.Fatalf("expected E2504 argument type error, got %v", err)
	}
}

func TestCheckRejectsUnknownIdentifier(t *testing.T) {
	err := parseAndCheck(t, `
fn main()
{
    print(missing)
}
`)
	if err == nil || !strings.Contains(err.Error(), "E2201") {
		t.Fatalf("expected E2201 unknown identifier, got %v", err)
	}
}

func TestCheckRejectsBranchReturnMismatch(t *testing.T) {
	err := parseAndCheck(t, `
fn classify(n: int) -> str
{
    if(n > 0){
        "positive"
    }
    else{
        0
    }
}
`)
	if err == nil || !strings.Contains(err.Error(), "E2102") {
		t.Fatalf("expected E2102 branch mismatch, got %v", err)
	}
}

func TestCheckAcceptsAnnotatedList(t *testing.T) {
	err := parseAndCheck(t, `
fn main()
{
    xs: list[int] = [1, 2, 3]
    empty: list[str] = []
    print(xs)
    print(empty)
}
`)
	if err != nil {
		t.Fatalf("expected annotated lists to pass: %v", err)
	}
}

func TestCheckRejectsMixedListElements(t *testing.T) {
	err := parseAndCheck(t, `
fn main()
{
    xs = [1, "two"]
    print(xs)
}
`)
	if err == nil || !strings.Contains(err.Error(), "E2602") {
		t.Fatalf("expected E2602 mixed list error, got %v", err)
	}
}

func TestCheckRejectsBadBindingAnnotation(t *testing.T) {
	err := parseAndCheck(t, `
fn main()
{
    xs: list[str] = [1, 2]
    print(xs)
}
`)
	if err == nil || !strings.Contains(err.Error(), "E2002") {
		t.Fatalf("expected E2002 binding annotation error, got %v", err)
	}
}

func TestCheckAcceptsObjects(t *testing.T) {
	err := parseAndCheck(t, `
fn main()
{
    user: obj = {name: "Jane", age: 30}
    print(user.name)
}
`)
	if err != nil {
		t.Fatalf("expected object program to pass: %v", err)
	}
}

func TestCheckAcceptsClassDecl(t *testing.T) {
	err := parseAndCheck(t, `
cl Account
{
    id: str
    balance: int
}

fn main()
{
    acc = Account(id= "1", balance= 100)
    print(acc.id)
    print(acc.balance)
}
`)
	if err != nil {
		t.Fatalf("expected class program to pass: %v", err)
	}
}

func TestCheckAcceptsClassWithCall(t *testing.T) {
	err := parseAndCheck(t, `
cl Account
{
    id: str
    balance: int
}

fn main()
{
    acc = Account(id= "1", balance= 100)
    acc2 = acc.with(balance= 50)
    print(acc2.balance)
}
`)
	if err != nil {
		t.Fatalf("expected .with() program to pass: %v", err)
	}
}

func TestCheckAcceptsClassKeys(t *testing.T) {
	err := parseAndCheck(t, `
cl Account
{
    id: str
    balance: int
}

fn main()
{
    acc = Account(id= "1", balance= 100)
    print(acc.keys)
}
`)
	if err != nil {
		t.Fatalf("expected .keys program to pass: %v", err)
	}
}

func TestCheckRejectsUnknownClassField(t *testing.T) {
	err := parseAndCheck(t, `
cl Account
{
    id: str
}

fn main()
{
    acc = Account(id= "1")
    print(acc.missing)
}
`)
	if err == nil || !strings.Contains(err.Error(), "E2703") {
		t.Fatalf("expected E2703 unknown field error, got %v", err)
	}
}

func TestCheckRejectsMissingConstructorField(t *testing.T) {
	err := parseAndCheck(t, `
cl Account
{
    id: str
    balance: int
}

fn main()
{
    acc = Account(id= "1")
    print(acc)
}
`)
	if err == nil || !strings.Contains(err.Error(), "E2506") {
		t.Fatalf("expected E2506 missing field error, got %v", err)
	}
}

func TestCheckRejectsDuplicateConstructorField(t *testing.T) {
	err := parseAndCheck(t, `
cl Account
{
    id: str
}

fn main()
{
    acc = Account(id= "1", id= "2")
    print(acc)
}
`)
	if err == nil || !strings.Contains(err.Error(), "E2507") {
		t.Fatalf("expected E2507 duplicate field error, got %v", err)
	}
}

func TestCheckRejectsUnknownConstructorField(t *testing.T) {
	err := parseAndCheck(t, `
cl Account
{
    id: str
}

fn main()
{
    acc = Account(id= "1", extra= "x")
    print(acc)
}
`)
	if err == nil || !strings.Contains(err.Error(), "E2505") {
		t.Fatalf("expected E2505 unknown field error, got %v", err)
	}
}

func TestCheckRejectsDuplicateClassDecl(t *testing.T) {
	err := parseAndCheck(t, `
cl Account { id: str }
cl Account { balance: int }
fn main() { }
`)
	if err == nil || !strings.Contains(err.Error(), "E1002") {
		t.Fatalf("expected E1002 duplicate class, got %v", err)
	}
}

func TestCheckRejectsDuplicateObjectField(t *testing.T) {
	err := parseAndCheck(t, `
fn main()
{
    user = {name: "Jane", name: "Mary"}
    print(user)
}
`)
	if err == nil || !strings.Contains(err.Error(), "E2701") {
		t.Fatalf("expected E2701 duplicate field error, got %v", err)
	}
}

func TestCheckAcceptsSwitch(t *testing.T) {
	err := parseAndCheck(t, `
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
`)
	if err != nil {
		t.Fatalf("expected switch program to pass: %v", err)
	}
}

func TestCheckRejectsSwitchCaseTypeMismatch(t *testing.T) {
	err := parseAndCheck(t, `
fn main()
{
    x = "1"
    switch(x){
        case(1){
            print("one")
        }
    }
}
`)
	if err == nil || !strings.Contains(err.Error(), "E2104") {
		t.Fatalf("expected E2104 switch case type error, got %v", err)
	}
}

func TestCheckRejectsSwitchNonLiteralCase(t *testing.T) {
	err := parseAndCheck(t, `
fn main()
{
    x = "1"
    y = "2"
    switch(x){
        case(y){
            print("two")
        }
    }
}
`)
	if err == nil || !strings.Contains(err.Error(), "E2103") {
		t.Fatalf("expected E2103 non-literal case error, got %v", err)
	}
}

func TestCheckRejectsSwitchBranchReturnMismatch(t *testing.T) {
	err := parseAndCheck(t, `
fn classify(x: str) -> str
{
    switch(x){
        case("1"){
            "one"
        }
        default(){
            0
        }
    }
}
`)
	if err == nil || !strings.Contains(err.Error(), "E2105") {
		t.Fatalf("expected E2105 branch mismatch, got %v", err)
	}
}

func TestCheckRejectsReturningSwitchWithoutDefault(t *testing.T) {
	err := parseAndCheck(t, `
fn classify(x: str) -> str
{
    switch(x){
        case("1"){
            "one"
        }
    }
}
`)
	if err == nil || !strings.Contains(err.Error(), "E2001") {
		t.Fatalf("expected E2001 missing switch default return error, got %v", err)
	}
}

func TestCheckAcceptsForList(t *testing.T) {
	err := parseAndCheck(t, `
fn main()
{
    xs = [1, 2, 3]
    for(x in xs){
        print(x + 1)
    }
}
`)
	if err != nil {
		t.Fatalf("expected list for loop to pass: %v", err)
	}
}

func TestCheckAcceptsForRange(t *testing.T) {
	err := parseAndCheck(t, `
fn main()
{
    for(i in 0..10){
        print(i)
    }
}
`)
	if err != nil {
		t.Fatalf("expected range for loop to pass: %v", err)
	}
}

func TestCheckRejectsRangeBoundsThatAreNotInt(t *testing.T) {
	err := parseAndCheck(t, `
fn main()
{
    for(i in 0.."10"){
        print(i)
    }
}
`)
	if err == nil || !strings.Contains(err.Error(), "E2107") {
		t.Fatalf("expected E2107 range bounds error, got %v", err)
	}
}

func TestCheckRejectsLoopVariableOutsideBody(t *testing.T) {
	err := parseAndCheck(t, `
fn main()
{
    for(i in 0..3){
        print(i)
    }
    print(i)
}
`)
	if err == nil || !strings.Contains(err.Error(), "E2201") {
		t.Fatalf("expected E2201 unknown loop variable, got %v", err)
	}
}

func TestCheckAcceptsLetSequentialBindings(t *testing.T) {
	err := parseAndCheck(t, `
fn calc() -> int
{
    let(x = 10, y = x + 2){
        y
    }
}
`)
	if err != nil {
		t.Fatalf("expected let expression to pass: %v", err)
	}
}

func TestCheckRejectsLetBindingOutsideBody(t *testing.T) {
	err := parseAndCheck(t, `
fn main()
{
    value = let(x = 1){ x }
    print(x)
}
`)
	if err == nil || !strings.Contains(err.Error(), "E2201") {
		t.Fatalf("expected E2201 for let scoped binding, got %v", err)
	}
}

func TestCheckRejectsLetReturnMismatch(t *testing.T) {
	err := parseAndCheck(t, `
fn calc() -> int
{
    let(x = 1){
        "x"
    }
}
`)
	if err == nil || !strings.Contains(err.Error(), "E2001") {
		t.Fatalf("expected E2001 let return mismatch, got %v", err)
	}
}

func TestCheckAcceptsMatchLiteralWildcard(t *testing.T) {
	err := parseAndCheck(t, `
fn classify(x: int) -> str
{
    match(x){
        case(0){ "zero" }
        case(_){ "other" }
    }
}
`)
	if err != nil {
		t.Fatalf("expected match to pass: %v", err)
	}
}

func TestCheckAcceptsMatchCaptureScope(t *testing.T) {
	err := parseAndCheck(t, `
fn classify(x: int) -> int
{
    match(x){
        case(n){ n + 1 }
    }
}
`)
	if err != nil {
		t.Fatalf("expected capture match to pass: %v", err)
	}
	err = parseAndCheck(t, `
fn main()
{
    match(1){ case(n){ print(n) } }
    print(n)
}
`)
	if err == nil || !strings.Contains(err.Error(), "E2201") {
		t.Fatalf("expected E2201 for capture scope, got %v", err)
	}
}

func TestCheckRejectsMatchCaseTypeMismatch(t *testing.T) {
	err := parseAndCheck(t, `
fn main()
{
    match(1){
        case("one"){ print("bad") }
        case(_){ print("ok") }
    }
}
`)
	if err == nil || !strings.Contains(err.Error(), "E2801") {
		t.Fatalf("expected E2801 match pattern type error, got %v", err)
	}
}

func TestCheckRejectsMatchWildcardOrdering(t *testing.T) {
	err := parseAndCheck(t, `
fn main()
{
    match(1){
        case(_){ print("all") }
        case(1){ print("one") }
    }
}
`)
	if err == nil || !strings.Contains(err.Error(), "E2803") {
		t.Fatalf("expected E2803 wildcard ordering error, got %v", err)
	}
}

func TestCheckRejectsNonExhaustiveMatchExpression(t *testing.T) {
	err := parseAndCheck(t, `
fn classify(x: int) -> str
{
    match(x){
        case(0){ "zero" }
    }
}
`)
	if err == nil || !strings.Contains(err.Error(), "E2805") {
		t.Fatalf("expected E2805 non-exhaustive match, got %v", err)
	}
}

func TestCheckAcceptsExhaustiveBoolMatch(t *testing.T) {
	err := parseAndCheck(t, `
fn yesno(x: bool) -> str
{
    match(x){
        case(true){ "yes" }
        case(false){ "no" }
    }
}
`)
	if err != nil {
		t.Fatalf("expected exhaustive bool match to pass: %v", err)
	}
}
