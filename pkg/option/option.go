package option

import (
	"bytes"
	"errors"
	"fmt"

	json "github.com/json-iterator/go"
)

// ErrNoneValueTaken 当获取 None 值时引发的错误。
var ErrNoneValueTaken = errors.New("none value taken")

// Option 是一个数据类型，它必须是 Some (即有值) 或 None (即没有值)。
// 此类型实现了 database/sql/driver.Valuer 和 database/sql.Scanner 接口。
type Option[T any] []T

const (
	value = iota
)

// Some 是一个函数，用于创建一个包含实际值的 Option 类型值。
func Some[T any](v T) Option[T] {
	return Option[T]{
		value: v,
	}
}

// None 是一个函数，用于创建一个不包含值的 Option 类型值。
func None[T any]() Option[T] {
	return nil
}

// FromNillable 是一个函数，用于从一个可为 nil 的值创建一个 Option 类型值，并进行值解引用。
// 如果给定值不为 nil，则返回 Some[T] 类型的值。反之，如果值为 nil，则返回 None[T]。
// 此函数在将值包装到 Option 值中时会对值进行"解引用"。如果不需要此行为，请考虑使用 PtrFromNillable()。
func FromNillable[T any](v *T) Option[T] {
	if v == nil {
		return None[T]()
	}
	return Some(*v)
}

// PtrFromNillable 是一个函数，用于从一个可为 nil 的值创建一个 Option 类型值，但不进行值解引用。
// 如果给定值不为 nil，则返回 Some[*T] 类型的值。反之，如果值为 nil，则返回 None[*T]。
// 此函数在将值包装到 Option 值中时不会对值进行"解引用"；换句话说，它将按原样将指针值放入 Option 封套中。
// 此行为与 FromNillable() 函数的行为相反。
func PtrFromNillable[T any](v *T) Option[*T] {
	if v == nil {
		return None[*T]()
	}
	return Some(v)
}

// IsNone 返回 Option 是否 *不* 包含值。
func (o Option[T]) IsNone() bool {
	return o == nil
}

// IsSome 返回 Option 是否包含值。
func (o Option[T]) IsSome() bool {
	return o != nil
}

// Unwrap 返回值，无论 Option 的状态是 Some 还是 None。
// 如果 Option 的值是 Some，此方法返回实际值。
// 反之，如果 Option 的值是 None，此方法返回该类型的 *默认* 值。
func (o Option[T]) Unwrap() T {
	if o.IsNone() {
		var defaultValue T
		return defaultValue
	}
	return o[value]
}

// UnwrapAsPtr 以指针形式返回接收者 Option 中包含的值。
// 这类似于 `Unwrap()` 方法，但区别在于此方法返回指针值而不是实际值。
// 如果接收者 Option 的值是 None，此方法返回 nil。
func (o Option[T]) UnwrapAsPtr() *T {
	if o.IsNone() {
		return nil
	}
	return &o[value]
}

// Take 获取 Option 中包含的值。
// 如果 Option 的值是 Some，则返回 Option 中包含的值。
// 反之，则返回 ErrNoneValueTaken 作为第二个返回值。
func (o Option[T]) Take() (T, error) {
	if o.IsNone() {
		var defaultValue T
		return defaultValue, ErrNoneValueTaken
	}
	return o[value], nil
}

// TakeOr 如果 Option 有值，则返回实际值。
// 反之，返回 fallbackValue。
func (o Option[T]) TakeOr(fallbackValue T) T {
	if o.IsNone() {
		return fallbackValue
	}
	return o[value]
}

// TakeOrElse 如果 Option 有值，则返回实际值。
// 反之，执行 fallbackFunc 并返回该函数的结果值。
func (o Option[T]) TakeOrElse(fallbackFunc func() T) T {
	if o.IsNone() {
		return fallbackFunc()
	}
	return o[value]
}

// Or 根据实际值是否存在返回 Option 值。
// 如果接收者的 Option 值是 Some，此函数将其直接返回。否则，此函数返回 `fallbackOptionValue`。
func (o Option[T]) Or(fallbackOptionValue Option[T]) Option[T] {
	if o.IsNone() {
		return fallbackOptionValue
	}
	return o
}

// OrElse 根据实际值是否存在返回 Option 值。
// 如果接收者的 Option 值是 Some，此函数将其直接返回。否则，此函数执行 `fallbackOptionFunc` 并返回该函数的结果值。
func (o Option[T]) OrElse(fallbackOptionFunc func() Option[T]) Option[T] {
	if o.IsNone() {
		return fallbackOptionFunc()
	}
	return o
}

// Filter 如果 Option 有值并且该值满足断言函数的条件，则返回自身。
// 在其他情况下 (即，它与断言不匹配或 Option 为 None)，则返回 None 值。
func (o Option[T]) Filter(predicate func(v T) bool) Option[T] {
	if o.IsNone() || !predicate(o[value]) {
		return None[T]()
	}
	return o
}

// IfSome 如果接收者值为 Some，则使用 Option 的值调用给定函数。
func (o Option[T]) IfSome(f func(v T)) {
	if o.IsNone() {
		return
	}
	f(o[value])
}

// IfSomeWithError 如果接收者值为 Some，则使用 Option 的值调用给定函数。
// 此方法传播给定函数的错误，如果接收者值为 None，则返回 nil 错误。
func (o Option[T]) IfSomeWithError(f func(v T) error) error {
	if o.IsNone() {
		return nil
	}
	return f(o[value])
}

// IfNone 如果接收者值为 None，则调用给定函数。
func (o Option[T]) IfNone(f func()) {
	if o.IsSome() {
		return
	}
	f()
}

// IfNoneWithError 如果接收者值为 None，则调用给定函数。
// 此方法传播给定函数的错误，如果接收者值为 Some，则返回 nil 错误。
func (o Option[T]) IfNoneWithError(f func() error) error {
	if o.IsSome() {
		return nil
	}
	return f()
}

func (o Option[T]) String() string {
	if o.IsNone() {
		return "None[]"
	}

	v := o.Unwrap()
	if stringer, ok := any(v).(fmt.Stringer); ok {
		return fmt.Sprintf("Some[%s]", stringer)
	}
	return fmt.Sprintf("Some[%v]", v)
}

// Map 根据映射函数将给定的 Option 值转换为另一个 Option 值。
// 如果给定的 Option 值为 None，则此函数也返回 None。
func Map[T, U any](option Option[T], mapper func(v T) U) Option[U] {
	if option.IsNone() {
		return None[U]()
	}

	return Some(mapper(option[value]))
}

// MapOr 根据映射函数将给定的 Option 值转换为另一个 *实际* 值。
// 如果给定的 Option 值为 None，则此函数返回 fallbackValue。
func MapOr[T, U any](option Option[T], fallbackValue U, mapper func(v T) U) U {
	if option.IsNone() {
		return fallbackValue
	}
	return mapper(option[value])
}

// MapWithError 根据能够返回带错误的值的映射函数，将给定的 Option 值转换为另一个 Option 值。
// 如果给定的 Option 值为 None，则返回 (None, nil)。如果映射函数返回错误，则返回 (None, error)。
// 否则，即给定的 Option 值为 Some 且映射函数未返回错误，则返回 (Some[U], nil)。
func MapWithError[T, U any](option Option[T], mapper func(v T) (U, error)) (Option[U], error) {
	if option.IsNone() {
		return None[U](), nil
	}

	u, err := mapper(option[value])
	if err != nil {
		return None[U](), err
	}
	return Some(u), nil
}

// MapOrWithError 根据能够返回带错误的值的映射函数，将给定的 Option 值转换为另一个 *实际* 值。
// 如果给定的 Option 值为 None，则返回 (fallbackValue, nil)。如果映射函数返回错误，则返回 (_, error)。
// 否则，即给定的 Option 值为 Some 且映射函数未返回错误，则返回 (U, nil)。
func MapOrWithError[T, U any](option Option[T], fallbackValue U, mapper func(v T) (U, error)) (U, error) {
	if option.IsNone() {
		return fallbackValue, nil
	}
	return mapper(option[value])
}

// FlatMap 根据映射函数将给定的 Option 值转换为另一个 Option 值。
// 与 Map 的区别在于，映射函数返回一个 Option 值而不是裸值。
// 如果给定的 Option 值为 None，则此函数也返回 None。
func FlatMap[T, U any](option Option[T], mapper func(v T) Option[U]) Option[U] {
	if option.IsNone() {
		return None[U]()
	}

	return mapper(option[value])
}

// FlatMapOr 根据映射函数将给定的 Option 值转换为另一个 *实际* 值。
// 与 MapOr 的区别在于，映射函数返回一个 Option 值而不是裸值。
// 如果给定的 Option 值为 None 或映射函数返回 None，则此函数返回 fallbackValue。
func FlatMapOr[T, U any](option Option[T], fallbackValue U, mapper func(v T) Option[U]) U {
	if option.IsNone() {
		return fallbackValue
	}

	return (mapper(option[value])).TakeOr(fallbackValue)
}

// FlatMapWithError 根据能够返回带错误的值的映射函数，将给定的 Option 值转换为另一个 Option 值。
// 与 MapWithError 的区别在于，映射函数返回一个 Option 值而不是裸值。
// 如果给定的 Option 值为 None，则返回 (None, nil)。如果映射函数返回错误，则返回 (None, error)。
// 否则，即给定的 Option 值为 Some 且映射函数未返回错误，则返回 (Some[U], nil)。
func FlatMapWithError[T, U any](option Option[T], mapper func(v T) (Option[U], error)) (Option[U], error) {
	if option.IsNone() {
		return None[U](), nil
	}

	mapped, err := mapper(option[value])
	if err != nil {
		return None[U](), err
	}
	return mapped, nil
}

// FlatMapOrWithError 根据能够返回带错误的值的映射函数，将给定的 Option 值转换为另一个 *实际* 值。
// 与 MapOrWithError 的区别在于，映射函数返回一个 Option 值而不是裸值。
// 如果给定的 Option 值为 None，则返回 (fallbackValue, nil)。如果映射函数返回错误，则返回 ($类型的零值, error)。
// 否则，即给定的 Option 值为 Some 且映射函数未返回错误，则返回 (U, nil)。
func FlatMapOrWithError[T, U any](option Option[T], fallbackValue U, mapper func(v T) (Option[U], error)) (U, error) {
	if option.IsNone() {
		return fallbackValue, nil
	}

	maybe, err := mapper(option[value])
	if err != nil {
		var zeroValue U
		return zeroValue, err
	}

	return maybe.TakeOr(fallbackValue), nil
}

// Pair 是一个表示包含两个元素的元组的数据类型。
type Pair[T, U any] struct {
	Value1 T
	Value2 U
}

// Zip 将两个 Option 压缩成一个包含每个 Option 值的 Pair。
// 如果任一 Option 为 None，则此函数也返回 None。
func Zip[T, U any](opt1 Option[T], opt2 Option[U]) Option[Pair[T, U]] {
	if opt1.IsSome() && opt2.IsSome() {
		return Some(Pair[T, U]{
			Value1: opt1[value],
			Value2: opt2[value],
		})
	}

	return None[Pair[T, U]]()
}

// ZipWith 根据压缩函数将两个 Option 压缩成一个类型化的值。
// 如果任一 Option 为 None，则此函数也返回 None。
func ZipWith[T, U, V any](opt1 Option[T], opt2 Option[U], zipper func(opt1 T, opt2 U) V) Option[V] {
	if opt1.IsSome() && opt2.IsSome() {
		return Some(zipper(opt1[value], opt2[value]))
	}
	return None[V]()
}

// Unzip 从 Pair 中提取值，并将它们分别包装到 Option 值中。
// 如果给定的压缩值为 None，则此函数对所有返回值都返回 None。
func Unzip[T, U any](zipped Option[Pair[T, U]]) (Option[T], Option[U]) {
	if zipped.IsNone() {
		return None[T](), None[U]()
	}

	pair := zipped[value]
	return Some(pair.Value1), Some(pair.Value2)
}

// UnzipWith 根据解压缩函数从给定值中提取值，并将它们分别包装到 Option 值中。
// 如果给定的压缩值为 None，则此函数对所有返回值都返回 None。
func UnzipWith[T, U, V any](zipped Option[V], unzipper func(zipped V) (T, U)) (Option[T], Option[U]) {
	if zipped.IsNone() {
		return None[T](), None[U]()
	}

	v1, v2 := unzipper(zipped[value])
	return Some(v1), Some(v2)
}

var jsonNull = []byte("null")

// MarshalJSON 将 Option 值序列化为 JSON 字节切片。
func (o Option[T]) MarshalJSON() ([]byte, error) {
	if o.IsNone() {
		return jsonNull, nil
	}

	marshal, err := json.Marshal(o.Unwrap())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal option: %w", err)
	}
	return marshal, nil
}

// UnmarshalJSON 将 JSON 字节切片反序列化为 Option 值。
func (o *Option[T]) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || bytes.Equal(data, jsonNull) {
		*o = None[T]()
		return nil
	}

	var v T
	err := json.Unmarshal(data, &v)
	if err != nil {
		return fmt.Errorf("failed to unmarshal option: %w", err)
	}
	*o = Some(v)

	return nil
}
