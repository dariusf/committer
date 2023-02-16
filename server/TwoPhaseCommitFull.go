package server

import (
	// "encoding/json"
	"fmt"
	"math"
	"path"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

// TLA expressions
type TLA interface {
	String() string
}

type Seq struct {
	elements []TLA
}

func (s Seq) String() string {
	ss := []string{}
	for _, v := range s.elements {
		ss = append(ss, v.String())
	}
	return fmt.Sprintf("<<%s>>", strings.Join(ss, ", "))
}

type Record struct {
	elements map[string]TLA
}

func (s Record) String() string {
	keys := []string{}
	for k := range s.elements {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	ss := []string{}
	for _, k := range keys {
		v := s.elements[k]
		ss = append(ss, fmt.Sprintf("%s |-> %s", k, v.String()))
	}
	return fmt.Sprintf("[%s]", strings.Join(ss, ", "))
}

type Set struct {
	elements map[string]TLA
}

func (s Set) String() string {
	keys := []string{}
	for k := range s.elements {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	ss := []string{}
	for _, k := range keys {
		v := s.elements[k]
		ss = append(ss, v.String())
	}
	return fmt.Sprintf("{%s}", strings.Join(ss, ", "))
}

type Int struct {
	value int
}

func (s Int) String() string {
	return fmt.Sprintf("%d", s.value)
}

type Bool struct {
	value bool
}

func (b Bool) String() string {
	return strconv.FormatBool(b.value)
}

type String struct {
	value string
}

func (s String) String() string {
	return s.value
}

// smart constructors

func boolean(b bool) Bool {
	return Bool{value: b}
}

func integer(n int) Int {
	return Int{value: n}
}

func str(s string) String {
	return String{value: s}
}

func set(elts ...TLA) Set {
	// avoid nil slice vs empty slice shenanigans
	if len(elts) == 0 {
		elts = []TLA{}
	}
	res := map[string]TLA{}
	for _, v := range elts {
		res[hash(v)] = v
	}
	return Set{elements: res}
}

func record(kvs ...TLA) Record {
	// avoid nil slice vs empty slice shenanigans
	if len(kvs) == 0 {
		kvs = []TLA{}
	}
	res := map[string]TLA{}
	for i := 0; i < len(kvs); i += 2 {
		res[kvs[i].(String).value] = kvs[i+1]
	}
	return Record{elements: res}
}

func seq(elts ...TLA) Seq {
	// avoid nil slice vs empty slice shenanigans
	if len(elts) == 0 {
		elts = []TLA{}
	}
	return Seq{elements: elts}
}

// library

func Some(a TLA) Seq {
	return seq(a)
}

func None() Seq {
	return seq()
}

func Append(a Seq, b TLA) Seq {
	return Seq{elements: append(a.elements, b)}
}

func AppendSeqs(a Seq, b Seq) Seq {
	return Seq{elements: append(a.elements, b.elements...)}
}

func Len(a Seq) Int {
	return integer(len(a.elements))
}

func Cardinality(a Set) Int {
	return integer(len(a.elements))
}

func SetUnion(a Set, b Set) Set {
	res := map[string]TLA{}
	for k, v := range a.elements {
		res[k] = v
	}
	for k, v := range b.elements {
		res[k] = v
	}
	return Set{elements: res}
}

func SetIn(a TLA, b Set) Bool {
	_, ok := b.elements[hash(a)]
	return boolean(ok)
}

func SetNotIn(a TLA, b Set) Bool {
	return boolean(!SetIn(a, b).value)
}

func RecordIndex(a Record, b String) TLA {
	return a.elements[b.value]
}

func TLAIndex(i int) int {
	return i - 1
}

func SeqIndex(a Seq, b Int) TLA {
	return a.elements[TLAIndex(b.value)]
}

func IndexInto(a TLA, b TLA) TLA {
	a1, ok1 := a.(Seq)
	b1, ok2 := b.(Int)
	if ok1 && ok2 {
		return SeqIndex(a1, b1)
	} else {
		return RecordIndex(a.(Record), b.(String))
	}
}

func IntPlus(a Int, b Int) Int {
	return integer(a.value + b.value)
}

func IntMinus(a Int, b Int) Int {
	return integer(a.value - b.value)
}

func IntMul(a Int, b Int) Int {
	return Int{value: a.value * b.value}
}

func IntDiv(a Int, b Int) Int {
	return Int{value: a.value / b.value}
}

func IntLt(a Int, b Int) Bool {
	return boolean(a.value < b.value)
}

func IntLte(a Int, b Int) Bool {
	return boolean(a.value <= b.value)
}

func IntGt(a Int, b Int) Bool {
	return boolean(a.value > b.value)
}

func IntGte(a Int, b Int) Bool {
	return boolean(a.value >= b.value)
}

func Eq(a TLA, b TLA) Bool {
	return boolean(reflect.DeepEqual(a, b))
}

func Not(b Bool) Bool {
	return boolean(!b.value)
}

func Neq(a TLA, b TLA) Bool {
	return Not(Eq(a, b))
}

func And(a Bool, b Bool) Bool {
	return boolean(a.value && b.value)
}

func Or(a Bool, b Bool) Bool {
	return boolean(a.value || b.value)
}

func IsFalse(a TLA) bool {
	return !a.(Bool).value
}

func IsTrue(a TLA) bool {
	return !IsFalse(a)
}

func ToSet(s Seq) Set {
	res := map[string]TLA{}
	for _, v := range s.elements {
		res[hash(v)] = v
	}
	return Set{elements: res}
}

func Except(r Record, k String, v TLA) Record {
	res := map[string]TLA{}
	for k1, v1 := range r.elements {
		res[k1] = v1
	}
	res[k.value] = v
	return Record{elements: res}
}

func BoundedForall(set Set, f func(TLA) Bool) Bool {
	res := true
	for _, v := range set.elements {
		res = res && IsTrue(f(v))
	}
	return Bool{value: res}
}

func BoundedExists(set Set, f func(TLA) Bool) Bool {
	res := false
	for _, v := range set.elements {
		res = res || IsTrue(f(v))
	}
	return Bool{value: res}
}

func FnConstruct(set Set, f func(TLA) TLA) Record {
	res := map[string]TLA{}
	for _, v := range set.elements {
		res[v.(String).value] = f(v)
	}
	return Record{elements: res}
}

func FoldSeq(f func(TLA, TLA) TLA, base TLA, seq Seq) TLA {
	res := base
	for _, v := range seq.elements {
		res = f(v, res)
	}
	return res
}

func Remove(s Seq, e TLA) Seq {
	res := []TLA{}
	for _, v := range s.elements {
		if IsFalse(Eq(v, e)) {
			res = append(res, v)
		}
	}
	return seq(res...)
}

func RemoveAt(s Seq, i Int) Seq {
	res := []TLA{}
	for j, v := range s.elements {
		if TLAIndex(i.value) != j {
			res = append(res, v)
		}
	}
	return seq(res...)
}

func SetToSeq(set Set) Seq {
	res := []TLA{}
	for _, v := range set.elements {
		res = append(res, v)
	}
	return seq(res...)
}

func Min(set Set) Int {
	res := math.MaxInt
	for _, v := range set.elements {
		if v.(Int).value < res {
			res = v.(Int).value
		}
	}
	return integer(res)
}

func Max(set Set) Int {
	res := math.MinInt
	for _, v := range set.elements {
		if v.(Int).value > res {
			res = v.(Int).value
		}
	}
	return integer(res)
}

func IsPrefix(prefix Seq, seq Seq) Bool {
	for i := 0; i < len(prefix.elements); i++ {
		if IsFalse(Eq(prefix.elements[i], seq.elements[i])) {
			return boolean(false)
		}
	}
	return boolean(true)
}

func SelectSeq(s Seq, f func(TLA) TLA) Seq {
	res := []TLA{}
	for _, v := range s.elements {
		if IsTrue(f(v)) {
			res = append(res, v)
		}
	}
	return seq(res...)
}

func RangeIncl(lower Int, upper Int) Set {
	res := []TLA{}
	for i := lower.value; i <= upper.value; i++ {
		res = append(res, integer(i))
	}
	return set(res...)
}

func d(a TLA) Bool {
	// fmt.Printf("%#v = %+v\n", a, a)
	fmt.Printf("%+v\n", a)
	return a.(Bool)
}

func dAnd(a Bool, b Bool) Bool {
	return d(And(a, b))
}

// panic instead of returning error
var crash = true

func hash(a TLA) string {
	return a.String()
}

// this does not guarantee keys are sorted, which is a problem since map traversal is nondet
// func hash(a any) string {
// 	return fmt.Sprintf("%+v", a)
// }

// this doesn't work for maps with non-string keys
// func hash(a any) string {
// 	s, _ := json.Marshal(a)
// 	return string(s)
// }

func thisFile() string {
	_, file, _, ok := runtime.Caller(1)
	if ok {
		return file
	}
	panic("could not get this file")
}

func getFileLine() (string, int) {
	for i := 1; i < 10; i++ {
		_, f, l, _ := runtime.Caller(i)
		if !strings.Contains(f, thisFile()) {
			return f, l
		}
	}
	panic("could not get file and line")
}

type State struct {
	who         TLA
	actions     TLA
	tmCommitted TLA
	outbox      TLA
	inflight    TLA
	tmPrepared  TLA
	tmDecision  TLA
	tmAborted   TLA
	rmState     TLA
	inbox       TLA
}

type EventType int

const (
	Initial = iota // special
	CSendPrepare
	PHandlePrepare
	CReceivePrepare
	CReceiveAbort
	CSendCommit
	CSendAbort
	PHandleCommit
	CReceiveCommit
	PHandleAbort
	NetworkTakeMessage
	NetworkDeliverMessage
)

func (e EventType) String() string {
	switch e {
	case Initial:
		return "Initial"
	case CSendPrepare:
		return "CSendPrepare"
	case PHandlePrepare:
		return "PHandlePrepare"
	case CReceivePrepare:
		return "CReceivePrepare"
	case CReceiveAbort:
		return "CReceiveAbort"
	case CSendCommit:
		return "CSendCommit"
	case CSendAbort:
		return "CSendAbort"
	case PHandleCommit:
		return "PHandleCommit"
	case CReceiveCommit:
		return "CReceiveCommit"
	case PHandleAbort:
		return "PHandleAbort"
	case NetworkTakeMessage:
		return "NetworkTakeMessage"
	case NetworkDeliverMessage:
		return "NetworkDeliverMessage"
	default:
		panic(fmt.Sprintf("invalid %d", e))
	}
}

type Event struct {
	typ    EventType
	params []TLA
	state  State
	file   string
	line   int
}

func printParams(ps []TLA) string {
	res := []string{}
	for _, v := range ps {
		res = append(res, fmt.Sprintf("%+v", v))
	}
	return strings.Join(res, ", ")
}

func (e Event) String() string {
	return fmt.Sprintf("%s(%s);%s:%d;%+v",
		e.typ, printParams(e.params), path.Base(e.file), e.line, e.state)
}

/*
type Constants struct {
    %s
}
*/

type Monitor struct {
	// the goal of extra is just to remove maintaining our own aux state,
	// which is annoying and error-prone as it may have to be passed across several functions
	extra  []Event
	events []Event
	//constants Constants
}

func NewMonitor( /* constants Constants */ ) *Monitor {
	return &Monitor{
		extra:  []Event{},
		events: []Event{},
		//constants: constants,
	}
}

// TODO check initial

func (m *Monitor) CheckTrace() error {
	var prev Event
	for i, this := range m.events {
		if i == 0 {
			prev = this
		}
		switch this.typ {
		case Initial:
			if err := m.CheckInitial(i, Event{}, this); err != nil {
				return err
			}
		case CSendPrepare:
			if err := m.CheckCSendPrepare(this.params[0], i, prev, this); err != nil {
				return err
			}
		case PHandlePrepare:
			if err := m.CheckPHandlePrepare(this.params[0], i, prev, this); err != nil {
				return err
			}
		case CReceivePrepare:
			if err := m.CheckCReceivePrepare(this.params[0], i, prev, this); err != nil {
				return err
			}
		case CReceiveAbort:
			if err := m.CheckCReceiveAbort(this.params[0], i, prev, this); err != nil {
				return err
			}
		case CSendCommit:
			if err := m.CheckCSendCommit(this.params[0], i, prev, this); err != nil {
				return err
			}
		case CSendAbort:
			if err := m.CheckCSendAbort(this.params[0], i, prev, this); err != nil {
				return err
			}
		case PHandleCommit:
			if err := m.CheckPHandleCommit(this.params[0], i, prev, this); err != nil {
				return err
			}
		case CReceiveCommit:
			if err := m.CheckCReceiveCommit(this.params[0], i, prev, this); err != nil {
				return err
			}
		case PHandleAbort:
			if err := m.CheckPHandleAbort(this.params[0], i, prev, this); err != nil {
				return err
			}
		case NetworkTakeMessage:
			if err := m.CheckNetworkTakeMessage(this.params[0], i, prev, this); err != nil {
				return err
			}
		case NetworkDeliverMessage:
			if err := m.CheckNetworkDeliverMessage(this.params[0], i, prev, this); err != nil {
				return err
			}
		}
		prev = this
	}
	return nil
}

func (m *Monitor) ShowTrace() {
	for i, v := range m.events {
		fmt.Printf("%d;%+v\n", i, v)
	}
}

func fail(format string, a ...any) error {
	if crash {
		panic(fmt.Sprintf(format, a...))
	}
	return fmt.Errorf(format, a...)
}

func (m *Monitor) CheckInitial(trace_i int, prev Event, this Event) error {

	// tmAborted = <<>>
	if IsFalse(Eq(this.state.tmAborted, seq())) {
		return fail("precondition failed in initial at %d; tmAborted = <<>>\n\nthis.state.tmAborted = %+v\n\nseq() = %+v", trace_i, this.state.tmAborted, seq())
	}
	// tmCommitted = <<>>
	if IsFalse(Eq(this.state.tmCommitted, seq())) {
		return fail("precondition failed in initial at %d; tmCommitted = <<>>\n\nthis.state.tmCommitted = %+v\n\nseq() = %+v", trace_i, this.state.tmCommitted, seq())
	}
	// tmPrepared = <<>>
	if IsFalse(Eq(this.state.tmPrepared, seq())) {
		return fail("precondition failed in initial at %d; tmPrepared = <<>>\n\nthis.state.tmPrepared = %+v\n\nseq() = %+v", trace_i, this.state.tmPrepared, seq())
	}
	// tmDecision = "none"
	if IsFalse(Eq(this.state.tmDecision, str("none"))) {
		return fail("precondition failed in initial at %d; tmDecision = \"none\"\n\nthis.state.tmDecision = %+v\n\nstr(\"none\") = %+v", trace_i, this.state.tmDecision, str("none"))
	}
	// rmState = [r1 |-> "working", r2 |-> "working"]
	if IsFalse(Eq(this.state.rmState, record(str("r1"), str("working"), str("r2"), str("working")))) {
		return fail("precondition failed in initial at %d; rmState = [r1 |-> \"working\", r2 |-> \"working\"]\n\nthis.state.rmState = %+v\n\nrecord(str(\"r1\"), str(\"working\"), str(\"r2\"), str(\"working\")) = %+v", trace_i, this.state.rmState, record(str("r1"), str("working"), str("r2"), str("working")))
	}
	// actions = <<>>
	if IsFalse(Eq(this.state.actions, seq())) {
		return fail("precondition failed in initial at %d; actions = <<>>\n\nthis.state.actions = %+v\n\nseq() = %+v", trace_i, this.state.actions, seq())
	}
	// outbox = [r1 |-> <<>>, r2 |-> <<>>, coordinator |-> <<>>]
	if IsFalse(Eq(this.state.outbox, record(str("r1"), seq(), str("r2"), seq(), str("coordinator"), seq()))) {
		return fail("precondition failed in initial at %d; outbox = [r1 |-> <<>>, r2 |-> <<>>, coordinator |-> <<>>]\n\nthis.state.outbox = %+v\n\nrecord(str(\"r1\"), seq(), str(\"r2\"), seq(), str(\"coordinator\"), seq()) = %+v", trace_i, this.state.outbox, record(str("r1"), seq(), str("r2"), seq(), str("coordinator"), seq()))
	}
	// inbox = [r1 |-> <<>>, r2 |-> <<>>, coordinator |-> <<>>]
	if IsFalse(Eq(this.state.inbox, record(str("r1"), seq(), str("r2"), seq(), str("coordinator"), seq()))) {
		return fail("precondition failed in initial at %d; inbox = [r1 |-> <<>>, r2 |-> <<>>, coordinator |-> <<>>]\n\nthis.state.inbox = %+v\n\nrecord(str(\"r1\"), seq(), str(\"r2\"), seq(), str(\"coordinator\"), seq()) = %+v", trace_i, this.state.inbox, record(str("r1"), seq(), str("r2"), seq(), str("coordinator"), seq()))
	}
	// who = "none"
	if IsFalse(Eq(this.state.who, str("none"))) {
		return fail("precondition failed in initial at %d; who = \"none\"\n\nthis.state.who = %+v\n\nstr(\"none\") = %+v", trace_i, this.state.who, str("none"))
	}
	// inflight = <<>>
	if IsFalse(Eq(this.state.inflight, seq())) {
		return fail("precondition failed in initial at %d; inflight = <<>>\n\nthis.state.inflight = %+v\n\nseq() = %+v", trace_i, this.state.inflight, seq())
	}
	return nil
}

func (monitor *Monitor) CheckCSendPrepare(r TLA, trace_i int, prev Event, this Event) error {

	// tmDecision = "none"
	if IsFalse(Eq(prev.state.tmDecision, str("none"))) {
		return fail("precondition failed in CSendPrepare at %d; tmDecision = \"none\"\n\nprev.state.tmDecision = %+v\n\nstr(\"none\") = %+v", trace_i, prev.state.tmDecision, str("none"))
	}

	// ToSet(tmPrepared) /= RM
	if IsFalse(Neq(ToSet(any(prev.state.tmPrepared).(Seq)), set(str("r1"), str("r2")))) {
		return fail("precondition failed in CSendPrepare at %d; ToSet(tmPrepared) /= RM\n\nToSet(any(prev.state.tmPrepared).(Seq)) = %+v\n\nset(str(\"r1\"), str(\"r2\")) = %+v", trace_i, ToSet(any(prev.state.tmPrepared).(Seq)), set(str("r1"), str("r2")))
	}

	// Send(["type" |-> "Prepare", "mdest" |-> r, "msource" |-> "coordinator"])
	if IsFalse(And(And(any(Eq(this.state.outbox, Except(any(prev.state.outbox).(Record), any(IndexInto(record(str("type"), str("Prepare"), str("mdest"), r, str("msource"), str("coordinator")), str("msource"))).(String), Append(any(IndexInto(prev.state.outbox, IndexInto(record(str("type"), str("Prepare"), str("mdest"), r, str("msource"), str("coordinator")), str("msource")))).(Seq), record(str("type"), str("Prepare"), str("mdest"), r, str("msource"), str("coordinator")))))).(Bool), Eq(this.state.inflight, prev.state.inflight)), Eq(this.state.inbox, prev.state.inbox))) {
		return fail("precondition failed in CSendPrepare at %d; Send([\"type\" |-> \"Prepare\", \"mdest\" |-> r, \"msource\" |-> \"coordinator\"])\n\nAnd(any(Eq(this.state.outbox, Except(any(prev.state.outbox).(Record), any(IndexInto(record(str(\"type\"), str(\"Prepare\"), str(\"mdest\"), r, str(\"msource\"), str(\"coordinator\")), str(\"msource\"))).(String), Append(any(IndexInto(prev.state.outbox, IndexInto(record(str(\"type\"), str(\"Prepare\"), str(\"mdest\"), r, str(\"msource\"), str(\"coordinator\")), str(\"msource\")))).(Seq), record(str(\"type\"), str(\"Prepare\"), str(\"mdest\"), r, str(\"msource\"), str(\"coordinator\")))))).(Bool), Eq(this.state.inflight, prev.state.inflight)) = %+v\n\nEq(this.state.inbox, prev.state.inbox) = %+v", trace_i, And(any(Eq(this.state.outbox, Except(any(prev.state.outbox).(Record), any(IndexInto(record(str("type"), str("Prepare"), str("mdest"), r, str("msource"), str("coordinator")), str("msource"))).(String), Append(any(IndexInto(prev.state.outbox, IndexInto(record(str("type"), str("Prepare"), str("mdest"), r, str("msource"), str("coordinator")), str("msource")))).(Seq), record(str("type"), str("Prepare"), str("mdest"), r, str("msource"), str("coordinator")))))).(Bool), Eq(this.state.inflight, prev.state.inflight)), Eq(this.state.inbox, prev.state.inbox))
	}

	// UNCHANGED(<<rmState>>)
	if IsFalse(Eq(this.state.rmState, prev.state.rmState)) {
		return fail("precondition failed in CSendPrepare at %d; UNCHANGED(<<rmState>>)\n\nthis.state.rmState = %+v\n\nprev.state.rmState = %+v", trace_i, this.state.rmState, prev.state.rmState)
	}

	// UNCHANGED(<<tmPrepared, tmCommitted, tmAborted, tmDecision>>)
	if IsFalse(And(And(And(Eq(this.state.tmPrepared, prev.state.tmPrepared), Eq(this.state.tmCommitted, prev.state.tmCommitted)), Eq(this.state.tmAborted, prev.state.tmAborted)), Eq(this.state.tmDecision, prev.state.tmDecision))) {
		return fail("precondition failed in CSendPrepare at %d; UNCHANGED(<<tmPrepared, tmCommitted, tmAborted, tmDecision>>)\n\nAnd(And(Eq(this.state.tmPrepared, prev.state.tmPrepared), Eq(this.state.tmCommitted, prev.state.tmCommitted)), Eq(this.state.tmAborted, prev.state.tmAborted)) = %+v\n\nEq(this.state.tmDecision, prev.state.tmDecision) = %+v", trace_i, And(And(Eq(this.state.tmPrepared, prev.state.tmPrepared), Eq(this.state.tmCommitted, prev.state.tmCommitted)), Eq(this.state.tmAborted, prev.state.tmAborted)), Eq(this.state.tmDecision, prev.state.tmDecision))
	}

	// LogAction(<<"CSendPrepare", r>>)
	if IsFalse(Eq(this.state.actions, Append(any(prev.state.actions).(Seq), seq(str("CSendPrepare"), r)))) {
		return fail("precondition failed in CSendPrepare at %d; LogAction(<<\"CSendPrepare\", r>>)\n\nthis.state.actions = %+v\n\nAppend(any(prev.state.actions).(Seq), seq(str(\"CSendPrepare\"), r)) = %+v", trace_i, this.state.actions, Append(any(prev.state.actions).(Seq), seq(str("CSendPrepare"), r)))
	}

	// LogActor("coordinator")
	if IsFalse(Eq(this.state.who, str("coordinator"))) {
		return fail("precondition failed in CSendPrepare at %d; LogActor(\"coordinator\")\n\nthis.state.who = %+v\n\nstr(\"coordinator\") = %+v", trace_i, this.state.who, str("coordinator"))
	}
	return nil
}

func (monitor *Monitor) CheckPHandlePrepare(r TLA, trace_i int, prev Event, this Event) error {

	// rmState[r] = "working"
	if IsFalse(Eq(IndexInto(prev.state.rmState, r), str("working"))) {
		return fail("precondition failed in PHandlePrepare at %d; rmState[r] = \"working\"\n\nIndexInto(prev.state.rmState, r) = %+v\n\nstr(\"working\") = %+v", trace_i, IndexInto(prev.state.rmState, r), str("working"))
	}

	if IsFalse(And(Eq(this.state.rmState, Except(any(prev.state.rmState).(Record), any(r).(String), str("prepared"))), any(And(And(any(And(SetIn(record(str("type"), str("Prepare"), str("mdest"), r, str("msource"), str("coordinator")), ToSet(any(IndexInto(prev.state.inbox, IndexInto(record(str("type"), str("Prepare"), str("mdest"), r, str("msource"), str("coordinator")), str("mdest")))).(Seq))), Eq(this.state.inbox, Except(any(prev.state.inbox).(Record), any(IndexInto(record(str("type"), str("Prepare"), str("mdest"), r, str("msource"), str("coordinator")), str("mdest"))).(String), Remove(any(IndexInto(prev.state.inbox, IndexInto(record(str("type"), str("Prepare"), str("mdest"), r, str("msource"), str("coordinator")), str("mdest")))).(Seq), record(str("type"), str("Prepare"), str("mdest"), r, str("msource"), str("coordinator"))))))).(Bool), any(Eq(this.state.outbox, Except(any(prev.state.outbox).(Record), any(IndexInto(record(str("type"), str("Prepared"), str("msource"), r, str("mdest"), str("coordinator")), str("msource"))).(String), Append(any(IndexInto(prev.state.outbox, IndexInto(record(str("type"), str("Prepared"), str("msource"), r, str("mdest"), str("coordinator")), str("msource")))).(Seq), record(str("type"), str("Prepared"), str("msource"), r, str("mdest"), str("coordinator")))))).(Bool)), Eq(this.state.inflight, prev.state.inflight))).(Bool))) {

		// ((rmState' = [rmState EXCEPT ![r] = "prepared"] /\ Reply(["type" |-> "Prepared", "msource" |-> r, "mdest" |-> "coordinator"], ["type" |-> "Prepare", "mdest" |-> r, "msource" |-> "coordinator"])) \/ (rmState' = [rmState EXCEPT ![r] = "aborted"] /\ Reply(["type" |-> "Aborted", "msource" |-> r, "mdest" |-> "coordinator"], ["type" |-> "Prepare", "mdest" |-> r, "msource" |-> "coordinator"])))
		if IsFalse(And(Eq(this.state.rmState, Except(any(prev.state.rmState).(Record), any(r).(String), str("aborted"))), any(And(And(any(And(SetIn(record(str("type"), str("Prepare"), str("mdest"), r, str("msource"), str("coordinator")), ToSet(any(IndexInto(prev.state.inbox, IndexInto(record(str("type"), str("Prepare"), str("mdest"), r, str("msource"), str("coordinator")), str("mdest")))).(Seq))), Eq(this.state.inbox, Except(any(prev.state.inbox).(Record), any(IndexInto(record(str("type"), str("Prepare"), str("mdest"), r, str("msource"), str("coordinator")), str("mdest"))).(String), Remove(any(IndexInto(prev.state.inbox, IndexInto(record(str("type"), str("Prepare"), str("mdest"), r, str("msource"), str("coordinator")), str("mdest")))).(Seq), record(str("type"), str("Prepare"), str("mdest"), r, str("msource"), str("coordinator"))))))).(Bool), any(Eq(this.state.outbox, Except(any(prev.state.outbox).(Record), any(IndexInto(record(str("type"), str("Aborted"), str("msource"), r, str("mdest"), str("coordinator")), str("msource"))).(String), Append(any(IndexInto(prev.state.outbox, IndexInto(record(str("type"), str("Aborted"), str("msource"), r, str("mdest"), str("coordinator")), str("msource")))).(Seq), record(str("type"), str("Aborted"), str("msource"), r, str("mdest"), str("coordinator")))))).(Bool)), Eq(this.state.inflight, prev.state.inflight))).(Bool))) {
			return fail("precondition failed in PHandlePrepare at %d; ((rmState' = [rmState EXCEPT ![r] = \"prepared\"] /\\ Reply([\"type\" |-> \"Prepared\", \"msource\" |-> r, \"mdest\" |-> \"coordinator\"], [\"type\" |-> \"Prepare\", \"mdest\" |-> r, \"msource\" |-> \"coordinator\"])) \\/ (rmState' = [rmState EXCEPT ![r] = \"aborted\"] /\\ Reply([\"type\" |-> \"Aborted\", \"msource\" |-> r, \"mdest\" |-> \"coordinator\"], [\"type\" |-> \"Prepare\", \"mdest\" |-> r, \"msource\" |-> \"coordinator\"])))\n\nEq(this.state.rmState, Except(any(prev.state.rmState).(Record), any(r).(String), str(\"aborted\"))) = %+v\n\nany(And(And(any(And(SetIn(record(str(\"type\"), str(\"Prepare\"), str(\"mdest\"), r, str(\"msource\"), str(\"coordinator\")), ToSet(any(IndexInto(prev.state.inbox, IndexInto(record(str(\"type\"), str(\"Prepare\"), str(\"mdest\"), r, str(\"msource\"), str(\"coordinator\")), str(\"mdest\")))).(Seq))), Eq(this.state.inbox, Except(any(prev.state.inbox).(Record), any(IndexInto(record(str(\"type\"), str(\"Prepare\"), str(\"mdest\"), r, str(\"msource\"), str(\"coordinator\")), str(\"mdest\"))).(String), Remove(any(IndexInto(prev.state.inbox, IndexInto(record(str(\"type\"), str(\"Prepare\"), str(\"mdest\"), r, str(\"msource\"), str(\"coordinator\")), str(\"mdest\")))).(Seq), record(str(\"type\"), str(\"Prepare\"), str(\"mdest\"), r, str(\"msource\"), str(\"coordinator\"))))))).(Bool), any(Eq(this.state.outbox, Except(any(prev.state.outbox).(Record), any(IndexInto(record(str(\"type\"), str(\"Aborted\"), str(\"msource\"), r, str(\"mdest\"), str(\"coordinator\")), str(\"msource\"))).(String), Append(any(IndexInto(prev.state.outbox, IndexInto(record(str(\"type\"), str(\"Aborted\"), str(\"msource\"), r, str(\"mdest\"), str(\"coordinator\")), str(\"msource\")))).(Seq), record(str(\"type\"), str(\"Aborted\"), str(\"msource\"), r, str(\"mdest\"), str(\"coordinator\")))))).(Bool)), Eq(this.state.inflight, prev.state.inflight))).(Bool) = %+v", trace_i, Eq(this.state.rmState, Except(any(prev.state.rmState).(Record), any(r).(String), str("aborted"))), any(And(And(any(And(SetIn(record(str("type"), str("Prepare"), str("mdest"), r, str("msource"), str("coordinator")), ToSet(any(IndexInto(prev.state.inbox, IndexInto(record(str("type"), str("Prepare"), str("mdest"), r, str("msource"), str("coordinator")), str("mdest")))).(Seq))), Eq(this.state.inbox, Except(any(prev.state.inbox).(Record), any(IndexInto(record(str("type"), str("Prepare"), str("mdest"), r, str("msource"), str("coordinator")), str("mdest"))).(String), Remove(any(IndexInto(prev.state.inbox, IndexInto(record(str("type"), str("Prepare"), str("mdest"), r, str("msource"), str("coordinator")), str("mdest")))).(Seq), record(str("type"), str("Prepare"), str("mdest"), r, str("msource"), str("coordinator"))))))).(Bool), any(Eq(this.state.outbox, Except(any(prev.state.outbox).(Record), any(IndexInto(record(str("type"), str("Aborted"), str("msource"), r, str("mdest"), str("coordinator")), str("msource"))).(String), Append(any(IndexInto(prev.state.outbox, IndexInto(record(str("type"), str("Aborted"), str("msource"), r, str("mdest"), str("coordinator")), str("msource")))).(Seq), record(str("type"), str("Aborted"), str("msource"), r, str("mdest"), str("coordinator")))))).(Bool)), Eq(this.state.inflight, prev.state.inflight))).(Bool))
		}
	}

	// UNCHANGED(<<tmPrepared, tmCommitted, tmAborted, tmDecision>>)
	if IsFalse(And(And(And(Eq(this.state.tmPrepared, prev.state.tmPrepared), Eq(this.state.tmCommitted, prev.state.tmCommitted)), Eq(this.state.tmAborted, prev.state.tmAborted)), Eq(this.state.tmDecision, prev.state.tmDecision))) {
		return fail("precondition failed in PHandlePrepare at %d; UNCHANGED(<<tmPrepared, tmCommitted, tmAborted, tmDecision>>)\n\nAnd(And(Eq(this.state.tmPrepared, prev.state.tmPrepared), Eq(this.state.tmCommitted, prev.state.tmCommitted)), Eq(this.state.tmAborted, prev.state.tmAborted)) = %+v\n\nEq(this.state.tmDecision, prev.state.tmDecision) = %+v", trace_i, And(And(Eq(this.state.tmPrepared, prev.state.tmPrepared), Eq(this.state.tmCommitted, prev.state.tmCommitted)), Eq(this.state.tmAborted, prev.state.tmAborted)), Eq(this.state.tmDecision, prev.state.tmDecision))
	}

	// LogAction(<<"PHandlePrepare", r, rmState'[r]>>)
	if IsFalse(Eq(this.state.actions, Append(any(prev.state.actions).(Seq), seq(str("PHandlePrepare"), r, IndexInto(this.state.rmState, r))))) {
		return fail("precondition failed in PHandlePrepare at %d; LogAction(<<\"PHandlePrepare\", r, rmState'[r]>>)\n\nthis.state.actions = %+v\n\nAppend(any(prev.state.actions).(Seq), seq(str(\"PHandlePrepare\"), r, IndexInto(this.state.rmState, r))) = %+v", trace_i, this.state.actions, Append(any(prev.state.actions).(Seq), seq(str("PHandlePrepare"), r, IndexInto(this.state.rmState, r))))
	}

	// LogActor(r)
	if IsFalse(Eq(this.state.who, r)) {
		return fail("precondition failed in PHandlePrepare at %d; LogActor(r)\n\nthis.state.who = %+v\n\nr = %+v", trace_i, this.state.who, r)
	}
	return nil
}

func (monitor *Monitor) CheckCReceivePrepare(r TLA, trace_i int, prev Event, this Event) error {

	// r \notin ToSet(tmPrepared)
	if IsFalse(SetNotIn(r, ToSet(any(prev.state.tmPrepared).(Seq)))) {
		return fail("precondition failed in CReceivePrepare at %d; r \\notin ToSet(tmPrepared)\n\nr = %+v\n\nToSet(any(prev.state.tmPrepared).(Seq)) = %+v", trace_i, r, ToSet(any(prev.state.tmPrepared).(Seq)))
	}

	// Receive(["type" |-> "Prepared", "mdest" |-> "coordinator", "msource" |-> r])
	if IsFalse(And(And(any(And(SetIn(record(str("type"), str("Prepared"), str("mdest"), str("coordinator"), str("msource"), r), ToSet(any(IndexInto(prev.state.inbox, IndexInto(record(str("type"), str("Prepared"), str("mdest"), str("coordinator"), str("msource"), r), str("mdest")))).(Seq))), Eq(this.state.inbox, Except(any(prev.state.inbox).(Record), any(IndexInto(record(str("type"), str("Prepared"), str("mdest"), str("coordinator"), str("msource"), r), str("mdest"))).(String), Remove(any(IndexInto(prev.state.inbox, IndexInto(record(str("type"), str("Prepared"), str("mdest"), str("coordinator"), str("msource"), r), str("mdest")))).(Seq), record(str("type"), str("Prepared"), str("mdest"), str("coordinator"), str("msource"), r)))))).(Bool), Eq(this.state.outbox, prev.state.outbox)), Eq(this.state.inflight, prev.state.inflight))) {
		return fail("precondition failed in CReceivePrepare at %d; Receive([\"type\" |-> \"Prepared\", \"mdest\" |-> \"coordinator\", \"msource\" |-> r])\n\nAnd(any(And(SetIn(record(str(\"type\"), str(\"Prepared\"), str(\"mdest\"), str(\"coordinator\"), str(\"msource\"), r), ToSet(any(IndexInto(prev.state.inbox, IndexInto(record(str(\"type\"), str(\"Prepared\"), str(\"mdest\"), str(\"coordinator\"), str(\"msource\"), r), str(\"mdest\")))).(Seq))), Eq(this.state.inbox, Except(any(prev.state.inbox).(Record), any(IndexInto(record(str(\"type\"), str(\"Prepared\"), str(\"mdest\"), str(\"coordinator\"), str(\"msource\"), r), str(\"mdest\"))).(String), Remove(any(IndexInto(prev.state.inbox, IndexInto(record(str(\"type\"), str(\"Prepared\"), str(\"mdest\"), str(\"coordinator\"), str(\"msource\"), r), str(\"mdest\")))).(Seq), record(str(\"type\"), str(\"Prepared\"), str(\"mdest\"), str(\"coordinator\"), str(\"msource\"), r)))))).(Bool), Eq(this.state.outbox, prev.state.outbox)) = %+v\n\nEq(this.state.inflight, prev.state.inflight) = %+v", trace_i, And(any(And(SetIn(record(str("type"), str("Prepared"), str("mdest"), str("coordinator"), str("msource"), r), ToSet(any(IndexInto(prev.state.inbox, IndexInto(record(str("type"), str("Prepared"), str("mdest"), str("coordinator"), str("msource"), r), str("mdest")))).(Seq))), Eq(this.state.inbox, Except(any(prev.state.inbox).(Record), any(IndexInto(record(str("type"), str("Prepared"), str("mdest"), str("coordinator"), str("msource"), r), str("mdest"))).(String), Remove(any(IndexInto(prev.state.inbox, IndexInto(record(str("type"), str("Prepared"), str("mdest"), str("coordinator"), str("msource"), r), str("mdest")))).(Seq), record(str("type"), str("Prepared"), str("mdest"), str("coordinator"), str("msource"), r)))))).(Bool), Eq(this.state.outbox, prev.state.outbox)), Eq(this.state.inflight, prev.state.inflight))
	}

	// tmPrepared' = Append(tmPrepared, r)
	if IsFalse(Eq(this.state.tmPrepared, Append(any(prev.state.tmPrepared).(Seq), r))) {
		return fail("postcondition failed in CReceivePrepare at %d; tmPrepared' = Append(tmPrepared, r)\n\nthis.state.tmPrepared = %+v\n\nAppend(any(prev.state.tmPrepared).(Seq), r) = %+v", trace_i, this.state.tmPrepared, Append(any(prev.state.tmPrepared).(Seq), r))
	}

	// UNCHANGED(<<rmState>>)
	if IsFalse(Eq(this.state.rmState, prev.state.rmState)) {
		return fail("precondition failed in CReceivePrepare at %d; UNCHANGED(<<rmState>>)\n\nthis.state.rmState = %+v\n\nprev.state.rmState = %+v", trace_i, this.state.rmState, prev.state.rmState)
	}

	// UNCHANGED(<<tmCommitted, tmAborted, tmDecision>>)
	if IsFalse(And(And(Eq(this.state.tmCommitted, prev.state.tmCommitted), Eq(this.state.tmAborted, prev.state.tmAborted)), Eq(this.state.tmDecision, prev.state.tmDecision))) {
		return fail("precondition failed in CReceivePrepare at %d; UNCHANGED(<<tmCommitted, tmAborted, tmDecision>>)\n\nAnd(Eq(this.state.tmCommitted, prev.state.tmCommitted), Eq(this.state.tmAborted, prev.state.tmAborted)) = %+v\n\nEq(this.state.tmDecision, prev.state.tmDecision) = %+v", trace_i, And(Eq(this.state.tmCommitted, prev.state.tmCommitted), Eq(this.state.tmAborted, prev.state.tmAborted)), Eq(this.state.tmDecision, prev.state.tmDecision))
	}

	// LogAction(<<"CReceivePrepare", r>>)
	if IsFalse(Eq(this.state.actions, Append(any(prev.state.actions).(Seq), seq(str("CReceivePrepare"), r)))) {
		return fail("precondition failed in CReceivePrepare at %d; LogAction(<<\"CReceivePrepare\", r>>)\n\nthis.state.actions = %+v\n\nAppend(any(prev.state.actions).(Seq), seq(str(\"CReceivePrepare\"), r)) = %+v", trace_i, this.state.actions, Append(any(prev.state.actions).(Seq), seq(str("CReceivePrepare"), r)))
	}

	// LogActor("coordinator")
	if IsFalse(Eq(this.state.who, str("coordinator"))) {
		return fail("precondition failed in CReceivePrepare at %d; LogActor(\"coordinator\")\n\nthis.state.who = %+v\n\nstr(\"coordinator\") = %+v", trace_i, this.state.who, str("coordinator"))
	}
	return nil
}

func (monitor *Monitor) CheckCReceiveAbort(r TLA, trace_i int, prev Event, this Event) error {

	// Receive(["type" |-> "Aborted", "msource" |-> r, "mdest" |-> "coordinator"])
	if IsFalse(And(And(any(And(SetIn(record(str("type"), str("Aborted"), str("msource"), r, str("mdest"), str("coordinator")), ToSet(any(IndexInto(prev.state.inbox, IndexInto(record(str("type"), str("Aborted"), str("msource"), r, str("mdest"), str("coordinator")), str("mdest")))).(Seq))), Eq(this.state.inbox, Except(any(prev.state.inbox).(Record), any(IndexInto(record(str("type"), str("Aborted"), str("msource"), r, str("mdest"), str("coordinator")), str("mdest"))).(String), Remove(any(IndexInto(prev.state.inbox, IndexInto(record(str("type"), str("Aborted"), str("msource"), r, str("mdest"), str("coordinator")), str("mdest")))).(Seq), record(str("type"), str("Aborted"), str("msource"), r, str("mdest"), str("coordinator"))))))).(Bool), Eq(this.state.outbox, prev.state.outbox)), Eq(this.state.inflight, prev.state.inflight))) {
		return fail("precondition failed in CReceiveAbort at %d; Receive([\"type\" |-> \"Aborted\", \"msource\" |-> r, \"mdest\" |-> \"coordinator\"])\n\nAnd(any(And(SetIn(record(str(\"type\"), str(\"Aborted\"), str(\"msource\"), r, str(\"mdest\"), str(\"coordinator\")), ToSet(any(IndexInto(prev.state.inbox, IndexInto(record(str(\"type\"), str(\"Aborted\"), str(\"msource\"), r, str(\"mdest\"), str(\"coordinator\")), str(\"mdest\")))).(Seq))), Eq(this.state.inbox, Except(any(prev.state.inbox).(Record), any(IndexInto(record(str(\"type\"), str(\"Aborted\"), str(\"msource\"), r, str(\"mdest\"), str(\"coordinator\")), str(\"mdest\"))).(String), Remove(any(IndexInto(prev.state.inbox, IndexInto(record(str(\"type\"), str(\"Aborted\"), str(\"msource\"), r, str(\"mdest\"), str(\"coordinator\")), str(\"mdest\")))).(Seq), record(str(\"type\"), str(\"Aborted\"), str(\"msource\"), r, str(\"mdest\"), str(\"coordinator\"))))))).(Bool), Eq(this.state.outbox, prev.state.outbox)) = %+v\n\nEq(this.state.inflight, prev.state.inflight) = %+v", trace_i, And(any(And(SetIn(record(str("type"), str("Aborted"), str("msource"), r, str("mdest"), str("coordinator")), ToSet(any(IndexInto(prev.state.inbox, IndexInto(record(str("type"), str("Aborted"), str("msource"), r, str("mdest"), str("coordinator")), str("mdest")))).(Seq))), Eq(this.state.inbox, Except(any(prev.state.inbox).(Record), any(IndexInto(record(str("type"), str("Aborted"), str("msource"), r, str("mdest"), str("coordinator")), str("mdest"))).(String), Remove(any(IndexInto(prev.state.inbox, IndexInto(record(str("type"), str("Aborted"), str("msource"), r, str("mdest"), str("coordinator")), str("mdest")))).(Seq), record(str("type"), str("Aborted"), str("msource"), r, str("mdest"), str("coordinator"))))))).(Bool), Eq(this.state.outbox, prev.state.outbox)), Eq(this.state.inflight, prev.state.inflight))
	}

	// r \notin ToSet(tmAborted)
	if IsFalse(SetNotIn(r, ToSet(any(prev.state.tmAborted).(Seq)))) {
		return fail("precondition failed in CReceiveAbort at %d; r \\notin ToSet(tmAborted)\n\nr = %+v\n\nToSet(any(prev.state.tmAborted).(Seq)) = %+v", trace_i, r, ToSet(any(prev.state.tmAborted).(Seq)))
	}

	// tmAborted' = Append(tmAborted, r)
	if IsFalse(Eq(this.state.tmAborted, Append(any(prev.state.tmAborted).(Seq), r))) {
		return fail("postcondition failed in CReceiveAbort at %d; tmAborted' = Append(tmAborted, r)\n\nthis.state.tmAborted = %+v\n\nAppend(any(prev.state.tmAborted).(Seq), r) = %+v", trace_i, this.state.tmAborted, Append(any(prev.state.tmAborted).(Seq), r))
	}

	// UNCHANGED(<<rmState>>)
	if IsFalse(Eq(this.state.rmState, prev.state.rmState)) {
		return fail("precondition failed in CReceiveAbort at %d; UNCHANGED(<<rmState>>)\n\nthis.state.rmState = %+v\n\nprev.state.rmState = %+v", trace_i, this.state.rmState, prev.state.rmState)
	}

	// UNCHANGED(<<tmPrepared, tmCommitted, tmDecision>>)
	if IsFalse(And(And(Eq(this.state.tmPrepared, prev.state.tmPrepared), Eq(this.state.tmCommitted, prev.state.tmCommitted)), Eq(this.state.tmDecision, prev.state.tmDecision))) {
		return fail("precondition failed in CReceiveAbort at %d; UNCHANGED(<<tmPrepared, tmCommitted, tmDecision>>)\n\nAnd(Eq(this.state.tmPrepared, prev.state.tmPrepared), Eq(this.state.tmCommitted, prev.state.tmCommitted)) = %+v\n\nEq(this.state.tmDecision, prev.state.tmDecision) = %+v", trace_i, And(Eq(this.state.tmPrepared, prev.state.tmPrepared), Eq(this.state.tmCommitted, prev.state.tmCommitted)), Eq(this.state.tmDecision, prev.state.tmDecision))
	}

	// LogAction(<<"CReceiveAbort", r>>)
	if IsFalse(Eq(this.state.actions, Append(any(prev.state.actions).(Seq), seq(str("CReceiveAbort"), r)))) {
		return fail("precondition failed in CReceiveAbort at %d; LogAction(<<\"CReceiveAbort\", r>>)\n\nthis.state.actions = %+v\n\nAppend(any(prev.state.actions).(Seq), seq(str(\"CReceiveAbort\"), r)) = %+v", trace_i, this.state.actions, Append(any(prev.state.actions).(Seq), seq(str("CReceiveAbort"), r)))
	}

	// LogActor("coordinator")
	if IsFalse(Eq(this.state.who, str("coordinator"))) {
		return fail("precondition failed in CReceiveAbort at %d; LogActor(\"coordinator\")\n\nthis.state.who = %+v\n\nstr(\"coordinator\") = %+v", trace_i, this.state.who, str("coordinator"))
	}
	return nil
}

func (monitor *Monitor) CheckCSendCommit(r TLA, trace_i int, prev Event, this Event) error {

	// ToSet(tmPrepared) = RM
	if IsFalse(Eq(ToSet(any(prev.state.tmPrepared).(Seq)), set(str("r1"), str("r2")))) {
		return fail("precondition failed in CSendCommit at %d; ToSet(tmPrepared) = RM\n\nToSet(any(prev.state.tmPrepared).(Seq)) = %+v\n\nset(str(\"r1\"), str(\"r2\")) = %+v", trace_i, ToSet(any(prev.state.tmPrepared).(Seq)), set(str("r1"), str("r2")))
	}

	// tmDecision /= "abort"
	if IsFalse(Neq(prev.state.tmDecision, str("abort"))) {
		return fail("precondition failed in CSendCommit at %d; tmDecision /= \"abort\"\n\nprev.state.tmDecision = %+v\n\nstr(\"abort\") = %+v", trace_i, prev.state.tmDecision, str("abort"))
	}

	// tmDecision' = "commit"
	if IsFalse(Eq(this.state.tmDecision, str("commit"))) {
		return fail("postcondition failed in CSendCommit at %d; tmDecision' = \"commit\"\n\nthis.state.tmDecision = %+v\n\nstr(\"commit\") = %+v", trace_i, this.state.tmDecision, str("commit"))
	}

	// Send(["type" |-> "Commit", "mdest" |-> r, "msource" |-> "coordinator"])
	if IsFalse(And(And(any(Eq(this.state.outbox, Except(any(prev.state.outbox).(Record), any(IndexInto(record(str("type"), str("Commit"), str("mdest"), r, str("msource"), str("coordinator")), str("msource"))).(String), Append(any(IndexInto(prev.state.outbox, IndexInto(record(str("type"), str("Commit"), str("mdest"), r, str("msource"), str("coordinator")), str("msource")))).(Seq), record(str("type"), str("Commit"), str("mdest"), r, str("msource"), str("coordinator")))))).(Bool), Eq(this.state.inflight, prev.state.inflight)), Eq(this.state.inbox, prev.state.inbox))) {
		return fail("precondition failed in CSendCommit at %d; Send([\"type\" |-> \"Commit\", \"mdest\" |-> r, \"msource\" |-> \"coordinator\"])\n\nAnd(any(Eq(this.state.outbox, Except(any(prev.state.outbox).(Record), any(IndexInto(record(str(\"type\"), str(\"Commit\"), str(\"mdest\"), r, str(\"msource\"), str(\"coordinator\")), str(\"msource\"))).(String), Append(any(IndexInto(prev.state.outbox, IndexInto(record(str(\"type\"), str(\"Commit\"), str(\"mdest\"), r, str(\"msource\"), str(\"coordinator\")), str(\"msource\")))).(Seq), record(str(\"type\"), str(\"Commit\"), str(\"mdest\"), r, str(\"msource\"), str(\"coordinator\")))))).(Bool), Eq(this.state.inflight, prev.state.inflight)) = %+v\n\nEq(this.state.inbox, prev.state.inbox) = %+v", trace_i, And(any(Eq(this.state.outbox, Except(any(prev.state.outbox).(Record), any(IndexInto(record(str("type"), str("Commit"), str("mdest"), r, str("msource"), str("coordinator")), str("msource"))).(String), Append(any(IndexInto(prev.state.outbox, IndexInto(record(str("type"), str("Commit"), str("mdest"), r, str("msource"), str("coordinator")), str("msource")))).(Seq), record(str("type"), str("Commit"), str("mdest"), r, str("msource"), str("coordinator")))))).(Bool), Eq(this.state.inflight, prev.state.inflight)), Eq(this.state.inbox, prev.state.inbox))
	}

	// UNCHANGED(<<rmState>>)
	if IsFalse(Eq(this.state.rmState, prev.state.rmState)) {
		return fail("precondition failed in CSendCommit at %d; UNCHANGED(<<rmState>>)\n\nthis.state.rmState = %+v\n\nprev.state.rmState = %+v", trace_i, this.state.rmState, prev.state.rmState)
	}

	// UNCHANGED(<<tmPrepared, tmCommitted, tmAborted>>)
	if IsFalse(And(And(Eq(this.state.tmPrepared, prev.state.tmPrepared), Eq(this.state.tmCommitted, prev.state.tmCommitted)), Eq(this.state.tmAborted, prev.state.tmAborted))) {
		return fail("precondition failed in CSendCommit at %d; UNCHANGED(<<tmPrepared, tmCommitted, tmAborted>>)\n\nAnd(Eq(this.state.tmPrepared, prev.state.tmPrepared), Eq(this.state.tmCommitted, prev.state.tmCommitted)) = %+v\n\nEq(this.state.tmAborted, prev.state.tmAborted) = %+v", trace_i, And(Eq(this.state.tmPrepared, prev.state.tmPrepared), Eq(this.state.tmCommitted, prev.state.tmCommitted)), Eq(this.state.tmAborted, prev.state.tmAborted))
	}

	// LogAction(<<"CSendCommit", r>>)
	if IsFalse(Eq(this.state.actions, Append(any(prev.state.actions).(Seq), seq(str("CSendCommit"), r)))) {
		return fail("precondition failed in CSendCommit at %d; LogAction(<<\"CSendCommit\", r>>)\n\nthis.state.actions = %+v\n\nAppend(any(prev.state.actions).(Seq), seq(str(\"CSendCommit\"), r)) = %+v", trace_i, this.state.actions, Append(any(prev.state.actions).(Seq), seq(str("CSendCommit"), r)))
	}

	// LogActor("coordinator")
	if IsFalse(Eq(this.state.who, str("coordinator"))) {
		return fail("precondition failed in CSendCommit at %d; LogActor(\"coordinator\")\n\nthis.state.who = %+v\n\nstr(\"coordinator\") = %+v", trace_i, this.state.who, str("coordinator"))
	}
	return nil
}

func (monitor *Monitor) CheckCSendAbort(r TLA, trace_i int, prev Event, this Event) error {

	// Len(tmAborted) > 0
	if IsFalse(IntGt(Len(any(prev.state.tmAborted).(Seq)), integer(0))) {
		return fail("precondition failed in CSendAbort at %d; Len(tmAborted) > 0\n\nLen(any(prev.state.tmAborted).(Seq)) = %+v\n\ninteger(0) = %+v", trace_i, Len(any(prev.state.tmAborted).(Seq)), integer(0))
	}

	// tmDecision /= "commit"
	if IsFalse(Neq(prev.state.tmDecision, str("commit"))) {
		return fail("precondition failed in CSendAbort at %d; tmDecision /= \"commit\"\n\nprev.state.tmDecision = %+v\n\nstr(\"commit\") = %+v", trace_i, prev.state.tmDecision, str("commit"))
	}

	// tmDecision' = "abort"
	if IsFalse(Eq(this.state.tmDecision, str("abort"))) {
		return fail("postcondition failed in CSendAbort at %d; tmDecision' = \"abort\"\n\nthis.state.tmDecision = %+v\n\nstr(\"abort\") = %+v", trace_i, this.state.tmDecision, str("abort"))
	}

	// r \notin ToSet(tmAborted)
	if IsFalse(SetNotIn(r, ToSet(any(prev.state.tmAborted).(Seq)))) {
		return fail("precondition failed in CSendAbort at %d; r \\notin ToSet(tmAborted)\n\nr = %+v\n\nToSet(any(prev.state.tmAborted).(Seq)) = %+v", trace_i, r, ToSet(any(prev.state.tmAborted).(Seq)))
	}

	// Send(["type" |-> "Abort", "mdest" |-> r, "msource" |-> "coordinator"])
	if IsFalse(And(And(any(Eq(this.state.outbox, Except(any(prev.state.outbox).(Record), any(IndexInto(record(str("type"), str("Abort"), str("mdest"), r, str("msource"), str("coordinator")), str("msource"))).(String), Append(any(IndexInto(prev.state.outbox, IndexInto(record(str("type"), str("Abort"), str("mdest"), r, str("msource"), str("coordinator")), str("msource")))).(Seq), record(str("type"), str("Abort"), str("mdest"), r, str("msource"), str("coordinator")))))).(Bool), Eq(this.state.inflight, prev.state.inflight)), Eq(this.state.inbox, prev.state.inbox))) {
		return fail("precondition failed in CSendAbort at %d; Send([\"type\" |-> \"Abort\", \"mdest\" |-> r, \"msource\" |-> \"coordinator\"])\n\nAnd(any(Eq(this.state.outbox, Except(any(prev.state.outbox).(Record), any(IndexInto(record(str(\"type\"), str(\"Abort\"), str(\"mdest\"), r, str(\"msource\"), str(\"coordinator\")), str(\"msource\"))).(String), Append(any(IndexInto(prev.state.outbox, IndexInto(record(str(\"type\"), str(\"Abort\"), str(\"mdest\"), r, str(\"msource\"), str(\"coordinator\")), str(\"msource\")))).(Seq), record(str(\"type\"), str(\"Abort\"), str(\"mdest\"), r, str(\"msource\"), str(\"coordinator\")))))).(Bool), Eq(this.state.inflight, prev.state.inflight)) = %+v\n\nEq(this.state.inbox, prev.state.inbox) = %+v", trace_i, And(any(Eq(this.state.outbox, Except(any(prev.state.outbox).(Record), any(IndexInto(record(str("type"), str("Abort"), str("mdest"), r, str("msource"), str("coordinator")), str("msource"))).(String), Append(any(IndexInto(prev.state.outbox, IndexInto(record(str("type"), str("Abort"), str("mdest"), r, str("msource"), str("coordinator")), str("msource")))).(Seq), record(str("type"), str("Abort"), str("mdest"), r, str("msource"), str("coordinator")))))).(Bool), Eq(this.state.inflight, prev.state.inflight)), Eq(this.state.inbox, prev.state.inbox))
	}

	// UNCHANGED(<<rmState>>)
	if IsFalse(Eq(this.state.rmState, prev.state.rmState)) {
		return fail("precondition failed in CSendAbort at %d; UNCHANGED(<<rmState>>)\n\nthis.state.rmState = %+v\n\nprev.state.rmState = %+v", trace_i, this.state.rmState, prev.state.rmState)
	}

	// UNCHANGED(<<tmPrepared, tmCommitted, tmAborted>>)
	if IsFalse(And(And(Eq(this.state.tmPrepared, prev.state.tmPrepared), Eq(this.state.tmCommitted, prev.state.tmCommitted)), Eq(this.state.tmAborted, prev.state.tmAborted))) {
		return fail("precondition failed in CSendAbort at %d; UNCHANGED(<<tmPrepared, tmCommitted, tmAborted>>)\n\nAnd(Eq(this.state.tmPrepared, prev.state.tmPrepared), Eq(this.state.tmCommitted, prev.state.tmCommitted)) = %+v\n\nEq(this.state.tmAborted, prev.state.tmAborted) = %+v", trace_i, And(Eq(this.state.tmPrepared, prev.state.tmPrepared), Eq(this.state.tmCommitted, prev.state.tmCommitted)), Eq(this.state.tmAborted, prev.state.tmAborted))
	}

	// LogAction(<<"CSendAbort", r>>)
	if IsFalse(Eq(this.state.actions, Append(any(prev.state.actions).(Seq), seq(str("CSendAbort"), r)))) {
		return fail("precondition failed in CSendAbort at %d; LogAction(<<\"CSendAbort\", r>>)\n\nthis.state.actions = %+v\n\nAppend(any(prev.state.actions).(Seq), seq(str(\"CSendAbort\"), r)) = %+v", trace_i, this.state.actions, Append(any(prev.state.actions).(Seq), seq(str("CSendAbort"), r)))
	}

	// LogActor("coordinator")
	if IsFalse(Eq(this.state.who, str("coordinator"))) {
		return fail("precondition failed in CSendAbort at %d; LogActor(\"coordinator\")\n\nthis.state.who = %+v\n\nstr(\"coordinator\") = %+v", trace_i, this.state.who, str("coordinator"))
	}
	return nil
}

func (monitor *Monitor) CheckPHandleCommit(r TLA, trace_i int, prev Event, this Event) error {

	// rmState[r] = "prepared"
	if IsFalse(Eq(IndexInto(prev.state.rmState, r), str("prepared"))) {
		return fail("precondition failed in PHandleCommit at %d; rmState[r] = \"prepared\"\n\nIndexInto(prev.state.rmState, r) = %+v\n\nstr(\"prepared\") = %+v", trace_i, IndexInto(prev.state.rmState, r), str("prepared"))
	}

	// rmState' = [rmState EXCEPT ![r] = "committed"]
	if IsFalse(Eq(this.state.rmState, Except(any(prev.state.rmState).(Record), any(r).(String), str("committed")))) {
		return fail("postcondition failed in PHandleCommit at %d; rmState' = [rmState EXCEPT ![r] = \"committed\"]\n\nthis.state.rmState = %+v\n\nExcept(any(prev.state.rmState).(Record), any(r).(String), str(\"committed\")) = %+v", trace_i, this.state.rmState, Except(any(prev.state.rmState).(Record), any(r).(String), str("committed")))
	}

	// Reply(["type" |-> "Committed", "msource" |-> r, "mdest" |-> "coordinator"], ["type" |-> "Commit", "mdest" |-> r, "msource" |-> "coordinator"])
	if IsFalse(And(And(any(And(SetIn(record(str("type"), str("Commit"), str("mdest"), r, str("msource"), str("coordinator")), ToSet(any(IndexInto(prev.state.inbox, IndexInto(record(str("type"), str("Commit"), str("mdest"), r, str("msource"), str("coordinator")), str("mdest")))).(Seq))), Eq(this.state.inbox, Except(any(prev.state.inbox).(Record), any(IndexInto(record(str("type"), str("Commit"), str("mdest"), r, str("msource"), str("coordinator")), str("mdest"))).(String), Remove(any(IndexInto(prev.state.inbox, IndexInto(record(str("type"), str("Commit"), str("mdest"), r, str("msource"), str("coordinator")), str("mdest")))).(Seq), record(str("type"), str("Commit"), str("mdest"), r, str("msource"), str("coordinator"))))))).(Bool), any(Eq(this.state.outbox, Except(any(prev.state.outbox).(Record), any(IndexInto(record(str("type"), str("Committed"), str("msource"), r, str("mdest"), str("coordinator")), str("msource"))).(String), Append(any(IndexInto(prev.state.outbox, IndexInto(record(str("type"), str("Committed"), str("msource"), r, str("mdest"), str("coordinator")), str("msource")))).(Seq), record(str("type"), str("Committed"), str("msource"), r, str("mdest"), str("coordinator")))))).(Bool)), Eq(this.state.inflight, prev.state.inflight))) {
		return fail("precondition failed in PHandleCommit at %d; Reply([\"type\" |-> \"Committed\", \"msource\" |-> r, \"mdest\" |-> \"coordinator\"], [\"type\" |-> \"Commit\", \"mdest\" |-> r, \"msource\" |-> \"coordinator\"])\n\nAnd(any(And(SetIn(record(str(\"type\"), str(\"Commit\"), str(\"mdest\"), r, str(\"msource\"), str(\"coordinator\")), ToSet(any(IndexInto(prev.state.inbox, IndexInto(record(str(\"type\"), str(\"Commit\"), str(\"mdest\"), r, str(\"msource\"), str(\"coordinator\")), str(\"mdest\")))).(Seq))), Eq(this.state.inbox, Except(any(prev.state.inbox).(Record), any(IndexInto(record(str(\"type\"), str(\"Commit\"), str(\"mdest\"), r, str(\"msource\"), str(\"coordinator\")), str(\"mdest\"))).(String), Remove(any(IndexInto(prev.state.inbox, IndexInto(record(str(\"type\"), str(\"Commit\"), str(\"mdest\"), r, str(\"msource\"), str(\"coordinator\")), str(\"mdest\")))).(Seq), record(str(\"type\"), str(\"Commit\"), str(\"mdest\"), r, str(\"msource\"), str(\"coordinator\"))))))).(Bool), any(Eq(this.state.outbox, Except(any(prev.state.outbox).(Record), any(IndexInto(record(str(\"type\"), str(\"Committed\"), str(\"msource\"), r, str(\"mdest\"), str(\"coordinator\")), str(\"msource\"))).(String), Append(any(IndexInto(prev.state.outbox, IndexInto(record(str(\"type\"), str(\"Committed\"), str(\"msource\"), r, str(\"mdest\"), str(\"coordinator\")), str(\"msource\")))).(Seq), record(str(\"type\"), str(\"Committed\"), str(\"msource\"), r, str(\"mdest\"), str(\"coordinator\")))))).(Bool)) = %+v\n\nEq(this.state.inflight, prev.state.inflight) = %+v", trace_i, And(any(And(SetIn(record(str("type"), str("Commit"), str("mdest"), r, str("msource"), str("coordinator")), ToSet(any(IndexInto(prev.state.inbox, IndexInto(record(str("type"), str("Commit"), str("mdest"), r, str("msource"), str("coordinator")), str("mdest")))).(Seq))), Eq(this.state.inbox, Except(any(prev.state.inbox).(Record), any(IndexInto(record(str("type"), str("Commit"), str("mdest"), r, str("msource"), str("coordinator")), str("mdest"))).(String), Remove(any(IndexInto(prev.state.inbox, IndexInto(record(str("type"), str("Commit"), str("mdest"), r, str("msource"), str("coordinator")), str("mdest")))).(Seq), record(str("type"), str("Commit"), str("mdest"), r, str("msource"), str("coordinator"))))))).(Bool), any(Eq(this.state.outbox, Except(any(prev.state.outbox).(Record), any(IndexInto(record(str("type"), str("Committed"), str("msource"), r, str("mdest"), str("coordinator")), str("msource"))).(String), Append(any(IndexInto(prev.state.outbox, IndexInto(record(str("type"), str("Committed"), str("msource"), r, str("mdest"), str("coordinator")), str("msource")))).(Seq), record(str("type"), str("Committed"), str("msource"), r, str("mdest"), str("coordinator")))))).(Bool)), Eq(this.state.inflight, prev.state.inflight))
	}

	// UNCHANGED(<<tmPrepared, tmCommitted, tmAborted, tmDecision>>)
	if IsFalse(And(And(And(Eq(this.state.tmPrepared, prev.state.tmPrepared), Eq(this.state.tmCommitted, prev.state.tmCommitted)), Eq(this.state.tmAborted, prev.state.tmAborted)), Eq(this.state.tmDecision, prev.state.tmDecision))) {
		return fail("precondition failed in PHandleCommit at %d; UNCHANGED(<<tmPrepared, tmCommitted, tmAborted, tmDecision>>)\n\nAnd(And(Eq(this.state.tmPrepared, prev.state.tmPrepared), Eq(this.state.tmCommitted, prev.state.tmCommitted)), Eq(this.state.tmAborted, prev.state.tmAborted)) = %+v\n\nEq(this.state.tmDecision, prev.state.tmDecision) = %+v", trace_i, And(And(Eq(this.state.tmPrepared, prev.state.tmPrepared), Eq(this.state.tmCommitted, prev.state.tmCommitted)), Eq(this.state.tmAborted, prev.state.tmAborted)), Eq(this.state.tmDecision, prev.state.tmDecision))
	}

	// LogAction(<<"PHandleCommit", r>>)
	if IsFalse(Eq(this.state.actions, Append(any(prev.state.actions).(Seq), seq(str("PHandleCommit"), r)))) {
		return fail("precondition failed in PHandleCommit at %d; LogAction(<<\"PHandleCommit\", r>>)\n\nthis.state.actions = %+v\n\nAppend(any(prev.state.actions).(Seq), seq(str(\"PHandleCommit\"), r)) = %+v", trace_i, this.state.actions, Append(any(prev.state.actions).(Seq), seq(str("PHandleCommit"), r)))
	}

	// LogActor(r)
	if IsFalse(Eq(this.state.who, r)) {
		return fail("precondition failed in PHandleCommit at %d; LogActor(r)\n\nthis.state.who = %+v\n\nr = %+v", trace_i, this.state.who, r)
	}
	return nil
}

func (monitor *Monitor) CheckCReceiveCommit(r TLA, trace_i int, prev Event, this Event) error {

	// Receive(["type" |-> "Committed", "msource" |-> r, "mdest" |-> "coordinator"])
	if IsFalse(And(And(any(And(SetIn(record(str("type"), str("Committed"), str("msource"), r, str("mdest"), str("coordinator")), ToSet(any(IndexInto(prev.state.inbox, IndexInto(record(str("type"), str("Committed"), str("msource"), r, str("mdest"), str("coordinator")), str("mdest")))).(Seq))), Eq(this.state.inbox, Except(any(prev.state.inbox).(Record), any(IndexInto(record(str("type"), str("Committed"), str("msource"), r, str("mdest"), str("coordinator")), str("mdest"))).(String), Remove(any(IndexInto(prev.state.inbox, IndexInto(record(str("type"), str("Committed"), str("msource"), r, str("mdest"), str("coordinator")), str("mdest")))).(Seq), record(str("type"), str("Committed"), str("msource"), r, str("mdest"), str("coordinator"))))))).(Bool), Eq(this.state.outbox, prev.state.outbox)), Eq(this.state.inflight, prev.state.inflight))) {
		return fail("precondition failed in CReceiveCommit at %d; Receive([\"type\" |-> \"Committed\", \"msource\" |-> r, \"mdest\" |-> \"coordinator\"])\n\nAnd(any(And(SetIn(record(str(\"type\"), str(\"Committed\"), str(\"msource\"), r, str(\"mdest\"), str(\"coordinator\")), ToSet(any(IndexInto(prev.state.inbox, IndexInto(record(str(\"type\"), str(\"Committed\"), str(\"msource\"), r, str(\"mdest\"), str(\"coordinator\")), str(\"mdest\")))).(Seq))), Eq(this.state.inbox, Except(any(prev.state.inbox).(Record), any(IndexInto(record(str(\"type\"), str(\"Committed\"), str(\"msource\"), r, str(\"mdest\"), str(\"coordinator\")), str(\"mdest\"))).(String), Remove(any(IndexInto(prev.state.inbox, IndexInto(record(str(\"type\"), str(\"Committed\"), str(\"msource\"), r, str(\"mdest\"), str(\"coordinator\")), str(\"mdest\")))).(Seq), record(str(\"type\"), str(\"Committed\"), str(\"msource\"), r, str(\"mdest\"), str(\"coordinator\"))))))).(Bool), Eq(this.state.outbox, prev.state.outbox)) = %+v\n\nEq(this.state.inflight, prev.state.inflight) = %+v", trace_i, And(any(And(SetIn(record(str("type"), str("Committed"), str("msource"), r, str("mdest"), str("coordinator")), ToSet(any(IndexInto(prev.state.inbox, IndexInto(record(str("type"), str("Committed"), str("msource"), r, str("mdest"), str("coordinator")), str("mdest")))).(Seq))), Eq(this.state.inbox, Except(any(prev.state.inbox).(Record), any(IndexInto(record(str("type"), str("Committed"), str("msource"), r, str("mdest"), str("coordinator")), str("mdest"))).(String), Remove(any(IndexInto(prev.state.inbox, IndexInto(record(str("type"), str("Committed"), str("msource"), r, str("mdest"), str("coordinator")), str("mdest")))).(Seq), record(str("type"), str("Committed"), str("msource"), r, str("mdest"), str("coordinator"))))))).(Bool), Eq(this.state.outbox, prev.state.outbox)), Eq(this.state.inflight, prev.state.inflight))
	}

	// r \notin ToSet(tmCommitted)
	if IsFalse(SetNotIn(r, ToSet(any(prev.state.tmCommitted).(Seq)))) {
		return fail("precondition failed in CReceiveCommit at %d; r \\notin ToSet(tmCommitted)\n\nr = %+v\n\nToSet(any(prev.state.tmCommitted).(Seq)) = %+v", trace_i, r, ToSet(any(prev.state.tmCommitted).(Seq)))
	}

	// tmCommitted' = Append(tmCommitted, r)
	if IsFalse(Eq(this.state.tmCommitted, Append(any(prev.state.tmCommitted).(Seq), r))) {
		return fail("postcondition failed in CReceiveCommit at %d; tmCommitted' = Append(tmCommitted, r)\n\nthis.state.tmCommitted = %+v\n\nAppend(any(prev.state.tmCommitted).(Seq), r) = %+v", trace_i, this.state.tmCommitted, Append(any(prev.state.tmCommitted).(Seq), r))
	}

	// UNCHANGED(<<rmState>>)
	if IsFalse(Eq(this.state.rmState, prev.state.rmState)) {
		return fail("precondition failed in CReceiveCommit at %d; UNCHANGED(<<rmState>>)\n\nthis.state.rmState = %+v\n\nprev.state.rmState = %+v", trace_i, this.state.rmState, prev.state.rmState)
	}

	// UNCHANGED(<<tmPrepared, tmAborted, tmDecision>>)
	if IsFalse(And(And(Eq(this.state.tmPrepared, prev.state.tmPrepared), Eq(this.state.tmAborted, prev.state.tmAborted)), Eq(this.state.tmDecision, prev.state.tmDecision))) {
		return fail("precondition failed in CReceiveCommit at %d; UNCHANGED(<<tmPrepared, tmAborted, tmDecision>>)\n\nAnd(Eq(this.state.tmPrepared, prev.state.tmPrepared), Eq(this.state.tmAborted, prev.state.tmAborted)) = %+v\n\nEq(this.state.tmDecision, prev.state.tmDecision) = %+v", trace_i, And(Eq(this.state.tmPrepared, prev.state.tmPrepared), Eq(this.state.tmAborted, prev.state.tmAborted)), Eq(this.state.tmDecision, prev.state.tmDecision))
	}

	// LogAction(<<"CReceiveCommit", r>>)
	if IsFalse(Eq(this.state.actions, Append(any(prev.state.actions).(Seq), seq(str("CReceiveCommit"), r)))) {
		return fail("precondition failed in CReceiveCommit at %d; LogAction(<<\"CReceiveCommit\", r>>)\n\nthis.state.actions = %+v\n\nAppend(any(prev.state.actions).(Seq), seq(str(\"CReceiveCommit\"), r)) = %+v", trace_i, this.state.actions, Append(any(prev.state.actions).(Seq), seq(str("CReceiveCommit"), r)))
	}

	// LogActor("coordinator")
	if IsFalse(Eq(this.state.who, str("coordinator"))) {
		return fail("precondition failed in CReceiveCommit at %d; LogActor(\"coordinator\")\n\nthis.state.who = %+v\n\nstr(\"coordinator\") = %+v", trace_i, this.state.who, str("coordinator"))
	}
	return nil
}

func (monitor *Monitor) CheckPHandleAbort(r TLA, trace_i int, prev Event, this Event) error {

	// rmState[r] \in {"working", "prepared"}
	if IsFalse(SetIn(IndexInto(prev.state.rmState, r), set(str("working"), str("prepared")))) {
		return fail("precondition failed in PHandleAbort at %d; rmState[r] \\in {\"working\", \"prepared\"}\n\nIndexInto(prev.state.rmState, r) = %+v\n\nset(str(\"working\"), str(\"prepared\")) = %+v", trace_i, IndexInto(prev.state.rmState, r), set(str("working"), str("prepared")))
	}

	// rmState' = [rmState EXCEPT ![r] = "aborted"]
	if IsFalse(Eq(this.state.rmState, Except(any(prev.state.rmState).(Record), any(r).(String), str("aborted")))) {
		return fail("postcondition failed in PHandleAbort at %d; rmState' = [rmState EXCEPT ![r] = \"aborted\"]\n\nthis.state.rmState = %+v\n\nExcept(any(prev.state.rmState).(Record), any(r).(String), str(\"aborted\")) = %+v", trace_i, this.state.rmState, Except(any(prev.state.rmState).(Record), any(r).(String), str("aborted")))
	}

	// Reply(["type" |-> "Aborted", "msource" |-> r, "mdest" |-> "coordinator"], ["type" |-> "Abort", "mdest" |-> r, "msource" |-> "coordinator"])
	if IsFalse(And(And(any(And(SetIn(record(str("type"), str("Abort"), str("mdest"), r, str("msource"), str("coordinator")), ToSet(any(IndexInto(prev.state.inbox, IndexInto(record(str("type"), str("Abort"), str("mdest"), r, str("msource"), str("coordinator")), str("mdest")))).(Seq))), Eq(this.state.inbox, Except(any(prev.state.inbox).(Record), any(IndexInto(record(str("type"), str("Abort"), str("mdest"), r, str("msource"), str("coordinator")), str("mdest"))).(String), Remove(any(IndexInto(prev.state.inbox, IndexInto(record(str("type"), str("Abort"), str("mdest"), r, str("msource"), str("coordinator")), str("mdest")))).(Seq), record(str("type"), str("Abort"), str("mdest"), r, str("msource"), str("coordinator"))))))).(Bool), any(Eq(this.state.outbox, Except(any(prev.state.outbox).(Record), any(IndexInto(record(str("type"), str("Aborted"), str("msource"), r, str("mdest"), str("coordinator")), str("msource"))).(String), Append(any(IndexInto(prev.state.outbox, IndexInto(record(str("type"), str("Aborted"), str("msource"), r, str("mdest"), str("coordinator")), str("msource")))).(Seq), record(str("type"), str("Aborted"), str("msource"), r, str("mdest"), str("coordinator")))))).(Bool)), Eq(this.state.inflight, prev.state.inflight))) {
		return fail("precondition failed in PHandleAbort at %d; Reply([\"type\" |-> \"Aborted\", \"msource\" |-> r, \"mdest\" |-> \"coordinator\"], [\"type\" |-> \"Abort\", \"mdest\" |-> r, \"msource\" |-> \"coordinator\"])\n\nAnd(any(And(SetIn(record(str(\"type\"), str(\"Abort\"), str(\"mdest\"), r, str(\"msource\"), str(\"coordinator\")), ToSet(any(IndexInto(prev.state.inbox, IndexInto(record(str(\"type\"), str(\"Abort\"), str(\"mdest\"), r, str(\"msource\"), str(\"coordinator\")), str(\"mdest\")))).(Seq))), Eq(this.state.inbox, Except(any(prev.state.inbox).(Record), any(IndexInto(record(str(\"type\"), str(\"Abort\"), str(\"mdest\"), r, str(\"msource\"), str(\"coordinator\")), str(\"mdest\"))).(String), Remove(any(IndexInto(prev.state.inbox, IndexInto(record(str(\"type\"), str(\"Abort\"), str(\"mdest\"), r, str(\"msource\"), str(\"coordinator\")), str(\"mdest\")))).(Seq), record(str(\"type\"), str(\"Abort\"), str(\"mdest\"), r, str(\"msource\"), str(\"coordinator\"))))))).(Bool), any(Eq(this.state.outbox, Except(any(prev.state.outbox).(Record), any(IndexInto(record(str(\"type\"), str(\"Aborted\"), str(\"msource\"), r, str(\"mdest\"), str(\"coordinator\")), str(\"msource\"))).(String), Append(any(IndexInto(prev.state.outbox, IndexInto(record(str(\"type\"), str(\"Aborted\"), str(\"msource\"), r, str(\"mdest\"), str(\"coordinator\")), str(\"msource\")))).(Seq), record(str(\"type\"), str(\"Aborted\"), str(\"msource\"), r, str(\"mdest\"), str(\"coordinator\")))))).(Bool)) = %+v\n\nEq(this.state.inflight, prev.state.inflight) = %+v", trace_i, And(any(And(SetIn(record(str("type"), str("Abort"), str("mdest"), r, str("msource"), str("coordinator")), ToSet(any(IndexInto(prev.state.inbox, IndexInto(record(str("type"), str("Abort"), str("mdest"), r, str("msource"), str("coordinator")), str("mdest")))).(Seq))), Eq(this.state.inbox, Except(any(prev.state.inbox).(Record), any(IndexInto(record(str("type"), str("Abort"), str("mdest"), r, str("msource"), str("coordinator")), str("mdest"))).(String), Remove(any(IndexInto(prev.state.inbox, IndexInto(record(str("type"), str("Abort"), str("mdest"), r, str("msource"), str("coordinator")), str("mdest")))).(Seq), record(str("type"), str("Abort"), str("mdest"), r, str("msource"), str("coordinator"))))))).(Bool), any(Eq(this.state.outbox, Except(any(prev.state.outbox).(Record), any(IndexInto(record(str("type"), str("Aborted"), str("msource"), r, str("mdest"), str("coordinator")), str("msource"))).(String), Append(any(IndexInto(prev.state.outbox, IndexInto(record(str("type"), str("Aborted"), str("msource"), r, str("mdest"), str("coordinator")), str("msource")))).(Seq), record(str("type"), str("Aborted"), str("msource"), r, str("mdest"), str("coordinator")))))).(Bool)), Eq(this.state.inflight, prev.state.inflight))
	}

	// UNCHANGED(<<tmPrepared, tmCommitted, tmAborted, tmDecision>>)
	if IsFalse(And(And(And(Eq(this.state.tmPrepared, prev.state.tmPrepared), Eq(this.state.tmCommitted, prev.state.tmCommitted)), Eq(this.state.tmAborted, prev.state.tmAborted)), Eq(this.state.tmDecision, prev.state.tmDecision))) {
		return fail("precondition failed in PHandleAbort at %d; UNCHANGED(<<tmPrepared, tmCommitted, tmAborted, tmDecision>>)\n\nAnd(And(Eq(this.state.tmPrepared, prev.state.tmPrepared), Eq(this.state.tmCommitted, prev.state.tmCommitted)), Eq(this.state.tmAborted, prev.state.tmAborted)) = %+v\n\nEq(this.state.tmDecision, prev.state.tmDecision) = %+v", trace_i, And(And(Eq(this.state.tmPrepared, prev.state.tmPrepared), Eq(this.state.tmCommitted, prev.state.tmCommitted)), Eq(this.state.tmAborted, prev.state.tmAborted)), Eq(this.state.tmDecision, prev.state.tmDecision))
	}

	// LogAction(<<"PHandleAbort", r>>)
	if IsFalse(Eq(this.state.actions, Append(any(prev.state.actions).(Seq), seq(str("PHandleAbort"), r)))) {
		return fail("precondition failed in PHandleAbort at %d; LogAction(<<\"PHandleAbort\", r>>)\n\nthis.state.actions = %+v\n\nAppend(any(prev.state.actions).(Seq), seq(str(\"PHandleAbort\"), r)) = %+v", trace_i, this.state.actions, Append(any(prev.state.actions).(Seq), seq(str("PHandleAbort"), r)))
	}

	// LogActor(r)
	if IsFalse(Eq(this.state.who, r)) {
		return fail("precondition failed in PHandleAbort at %d; LogActor(r)\n\nthis.state.who = %+v\n\nr = %+v", trace_i, this.state.who, r)
	}
	return nil
}

func (monitor *Monitor) CheckNetworkTakeMessage(msg TLA, trace_i int, prev Event, this Event) error {

	// msg \in ToSet(outbox[msg.msource])
	if IsFalse(SetIn(msg, ToSet(any(IndexInto(prev.state.outbox, IndexInto(msg, str("msource")))).(Seq)))) {
		return fail("precondition failed in NetworkTakeMessage at %d; msg \\in ToSet(outbox[msg.msource])\n\nmsg = %+v\n\nToSet(any(IndexInto(prev.state.outbox, IndexInto(msg, str(\"msource\")))).(Seq)) = %+v", trace_i, msg, ToSet(any(IndexInto(prev.state.outbox, IndexInto(msg, str("msource")))).(Seq)))
	}

	// UNCHANGED(inflight)
	if IsFalse(Eq(this.state.inflight, prev.state.inflight)) {
		return fail("precondition failed in NetworkTakeMessage at %d; UNCHANGED(inflight)\n\nthis.state.inflight = %+v\n\nprev.state.inflight = %+v", trace_i, this.state.inflight, prev.state.inflight)
	}

	// outbox' = [outbox EXCEPT ![msg.msource] = Remove(outbox[msg.msource], msg)]
	if IsFalse(Eq(this.state.outbox, Except(any(prev.state.outbox).(Record), any(IndexInto(msg, str("msource"))).(String), Remove(any(IndexInto(prev.state.outbox, IndexInto(msg, str("msource")))).(Seq), msg)))) {
		return fail("postcondition failed in NetworkTakeMessage at %d; outbox' = [outbox EXCEPT ![msg.msource] = Remove(outbox[msg.msource], msg)]\n\nthis.state.outbox = %+v\n\nExcept(any(prev.state.outbox).(Record), any(IndexInto(msg, str(\"msource\"))).(String), Remove(any(IndexInto(prev.state.outbox, IndexInto(msg, str(\"msource\")))).(Seq), msg)) = %+v", trace_i, this.state.outbox, Except(any(prev.state.outbox).(Record), any(IndexInto(msg, str("msource"))).(String), Remove(any(IndexInto(prev.state.outbox, IndexInto(msg, str("msource")))).(Seq), msg)))
	}

	// LogAction(<<"NetworkTakeMessage", msg>>)
	if IsFalse(Eq(this.state.actions, Append(any(prev.state.actions).(Seq), seq(str("NetworkTakeMessage"), msg)))) {
		return fail("precondition failed in NetworkTakeMessage at %d; LogAction(<<\"NetworkTakeMessage\", msg>>)\n\nthis.state.actions = %+v\n\nAppend(any(prev.state.actions).(Seq), seq(str(\"NetworkTakeMessage\"), msg)) = %+v", trace_i, this.state.actions, Append(any(prev.state.actions).(Seq), seq(str("NetworkTakeMessage"), msg)))
	}

	// LogActor("Network")
	if IsFalse(Eq(this.state.who, str("Network"))) {
		return fail("precondition failed in NetworkTakeMessage at %d; LogActor(\"Network\")\n\nthis.state.who = %+v\n\nstr(\"Network\") = %+v", trace_i, this.state.who, str("Network"))
	}

	// UNCHANGED(inbox)
	if IsFalse(Eq(this.state.inbox, prev.state.inbox)) {
		return fail("precondition failed in NetworkTakeMessage at %d; UNCHANGED(inbox)\n\nthis.state.inbox = %+v\n\nprev.state.inbox = %+v", trace_i, this.state.inbox, prev.state.inbox)
	}
	return nil
}

func (monitor *Monitor) CheckNetworkDeliverMessage(msg TLA, trace_i int, prev Event, this Event) error {

	// UNCHANGED(inflight)
	if IsFalse(Eq(this.state.inflight, prev.state.inflight)) {
		return fail("precondition failed in NetworkDeliverMessage at %d; UNCHANGED(inflight)\n\nthis.state.inflight = %+v\n\nprev.state.inflight = %+v", trace_i, this.state.inflight, prev.state.inflight)
	}

	// inbox' = [inbox EXCEPT ![msg.mdest] = Append(inbox[msg.mdest], msg)]
	if IsFalse(Eq(this.state.inbox, Except(any(prev.state.inbox).(Record), any(IndexInto(msg, str("mdest"))).(String), Append(any(IndexInto(prev.state.inbox, IndexInto(msg, str("mdest")))).(Seq), msg)))) {
		return fail("postcondition failed in NetworkDeliverMessage at %d; inbox' = [inbox EXCEPT ![msg.mdest] = Append(inbox[msg.mdest], msg)]\n\nthis.state.inbox = %+v\n\nExcept(any(prev.state.inbox).(Record), any(IndexInto(msg, str(\"mdest\"))).(String), Append(any(IndexInto(prev.state.inbox, IndexInto(msg, str(\"mdest\")))).(Seq), msg)) = %+v", trace_i, this.state.inbox, Except(any(prev.state.inbox).(Record), any(IndexInto(msg, str("mdest"))).(String), Append(any(IndexInto(prev.state.inbox, IndexInto(msg, str("mdest")))).(Seq), msg)))
	}

	// LogAction(<<"NetworkDeliverMessage", msg>>)
	if IsFalse(Eq(this.state.actions, Append(any(prev.state.actions).(Seq), seq(str("NetworkDeliverMessage"), msg)))) {
		return fail("precondition failed in NetworkDeliverMessage at %d; LogAction(<<\"NetworkDeliverMessage\", msg>>)\n\nthis.state.actions = %+v\n\nAppend(any(prev.state.actions).(Seq), seq(str(\"NetworkDeliverMessage\"), msg)) = %+v", trace_i, this.state.actions, Append(any(prev.state.actions).(Seq), seq(str("NetworkDeliverMessage"), msg)))
	}

	// LogActor("Network")
	if IsFalse(Eq(this.state.who, str("Network"))) {
		return fail("precondition failed in NetworkDeliverMessage at %d; LogActor(\"Network\")\n\nthis.state.who = %+v\n\nstr(\"Network\") = %+v", trace_i, this.state.who, str("Network"))
	}

	// UNCHANGED(outbox)
	if IsFalse(Eq(this.state.outbox, prev.state.outbox)) {
		return fail("precondition failed in NetworkDeliverMessage at %d; UNCHANGED(outbox)\n\nthis.state.outbox = %+v\n\nprev.state.outbox = %+v", trace_i, this.state.outbox, prev.state.outbox)
	}
	return nil
}

/*
func (m *Monitor) CheckInc(i int, prev Event, this Event) error {

	if prev.state.x.(int) <= 0 {
		return fail("precondition failed at %d; expected x <= 0 but got %s (prev: %+v, this: %+v)", i, prev.state.x, prev, this)
	}
	// check that new values are allowed
	if this.state.x != prev.state.x.(int)+1 { // for each var
		return fail("postcondition violated for x at %d; should be %+v but got %+v (prev: %+v, this: %+v)", i,
			prev.state.x.(int)+1, this.state.x, prev, this)
	}

	// check unchanged
	if this.state.x != prev.state.x { // for each var
		return fail("unchanged violated for x at %d; expected x to remain as %+v but it is %+v (prev: %+v, this: %+v)", i, prev.state.x, this.state.x, prev, this)
	}

	return nil
}
*/

// this state value can have nil fields
func (m *Monitor) CaptureVariable(v State, typ EventType, args ...TLA) error {

	e := Event{
		typ:    typ,
		params: args,
		state:  v,
		// no need to capture file and line here
	}
	m.extra = append(m.extra, e)
	return nil
}

func (m *Monitor) CaptureState(c State, typ EventType, args ...TLA) error {

	// override current values with extras
	// all have to pertain to this action
	for _, v := range m.extra {
		// sanity checks
		if v.typ != typ {
			return fmt.Errorf("type did not match")
		}
		for i, p := range v.params {
			if p != args[i] {
				return fmt.Errorf("arg %d did not match", i)
			}
		}
		// there is no null in TLA+, and also all the struct fields are any, which are reference types

		// for each variable in state
		if v.state.who != nil {
			c.who = v.state.who
		}
		if v.state.actions != nil {
			c.actions = v.state.actions
		}
		if v.state.tmCommitted != nil {
			c.tmCommitted = v.state.tmCommitted
		}
		if v.state.outbox != nil {
			c.outbox = v.state.outbox
		}
		if v.state.inflight != nil {
			c.inflight = v.state.inflight
		}
		if v.state.tmPrepared != nil {
			c.tmPrepared = v.state.tmPrepared
		}
		if v.state.tmDecision != nil {
			c.tmDecision = v.state.tmDecision
		}
		if v.state.tmAborted != nil {
			c.tmAborted = v.state.tmAborted
		}
		if v.state.rmState != nil {
			c.rmState = v.state.rmState
		}
		if v.state.inbox != nil {
			c.inbox = v.state.inbox
		}
	}

	// reset
	m.extra = []Event{}

	// record event
	file, line := getFileLine()
	e := Event{
		typ:    typ,
		params: args,
		state:  c,
		file:   file,
		line:   line,
	}

	m.events = append(m.events, e)

	return nil
}
