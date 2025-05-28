package option

import (
	"errors"

	"github.com/m4n5ter/mindscape/pkg/option"
)

// ErrNoneValueTaken 当获取 None 值时引发的错误。
var ErrNoneValueTaken = errors.New("none value taken")

// Option 是一个数据类型，它必须是 Some (即有值) 或 None (即没有值)。
// 此类型实现了 database/sql/driver.Valuer 和 database/sql.Scanner 接口。
type Option[T any] = option.Option[T]

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
