package avro

import (
	"fmt"
	"math/big"
	"reflect"
	"time"
	"unsafe"

	"github.com/modern-go/reflect2"
)

func createDecoderOfNative(schema Schema, typ reflect2.Type) ValDecoder {
	converter := resolveConverter(schema.(*PrimitiveSchema).actual)

	switch typ.Kind() {
	case reflect.Bool:
		if schema.Type() != Boolean {
			break
		}
		return &boolCodec{}

	case reflect.Int:
		if schema.Type() != Int {
			break
		}
		return &intCodec[int]{}

	case reflect.Int8:
		if schema.Type() != Int {
			break
		}
		return &intCodec[int8]{}

	case reflect.Uint8:
		if schema.Type() != Int {
			break
		}
		return &intCodec[uint8]{}

	case reflect.Int16:
		if schema.Type() != Int {
			break
		}
		return &intCodec[int16]{}

	case reflect.Uint16:
		if schema.Type() != Int {
			break
		}
		return &intCodec[uint16]{}

	case reflect.Int32:
		if schema.Type() != Int {
			break
		}
		return &intCodec[int32]{}

	case reflect.Uint32:
		if schema.Type() != Long {
			break
		}
		return &longCodec[uint32]{convert: converter.toLong}

	case reflect.Int64:
		st := schema.Type()
		lt := getLogicalType(schema)
		switch {
		case st == Int && lt == TimeMillis: // time.Duration
			return &timeMillisCodec{}

		case st == Long && lt == TimeMicros: // time.Duration
			return &timeMicrosCodec{
				convert: converter.toLong,
			}

		case st == Long:
			return &longCodec[int64]{
				convert: converter.toLong,
			}

		default:
			break
		}

	case reflect.Float32:
		if schema.Type() != Float {
			break
		}
		return &float32Codec{
			convert: converter.toFloat,
		}

	case reflect.Float64:
		if schema.Type() != Double {
			break
		}
		return &float64Codec{
			convert: converter.toDouble,
		}

	case reflect.String:
		if schema.Type() != String {
			break
		}
		return &stringCodec{
			convert: converter.toString,
		}

	case reflect.Slice:
		if typ.(reflect2.SliceType).Elem().Kind() != reflect.Uint8 || schema.Type() != Bytes {
			break
		}
		return &bytesCodec{
			sliceType: typ.(*reflect2.UnsafeSliceType),
			convert:   converter.toBytes,
		}

	case reflect.Struct:
		st := schema.Type()
		ls := getLogicalSchema(schema)
		lt := getLogicalType(schema)
		tpy1 := typ.Type1()
		Istpy1Time := tpy1.ConvertibleTo(timeType)
		Istpy1Rat := tpy1.ConvertibleTo(ratType)
		switch {
		case Istpy1Time && st == Int && lt == Date:
			return &dateCodec{}

		case Istpy1Time && st == Long && lt == TimestampMillis:
			return &timestampMillisCodec{
				convert: converter.toLong,
			}

		case Istpy1Time && st == Long && lt == TimestampMicros:
			return &timestampMicrosCodec{
				convert: converter.toLong,
			}

		case Istpy1Rat && st == Bytes && lt == Decimal:
			dec := ls.(*DecimalLogicalSchema)
			return &bytesDecimalCodec{
				prec: dec.Precision(), scale: dec.Scale(),
				convert: converter.toBytes,
			}

		default:
			break
		}
	case reflect.Ptr:
		ptrType := typ.(*reflect2.UnsafePtrType)
		elemType := ptrType.Elem()
		tpy1 := elemType.Type1()
		ls := getLogicalSchema(schema)
		if ls == nil {
			break
		}
		if !tpy1.ConvertibleTo(ratType) || schema.Type() != Bytes || ls.Type() != Decimal {
			break
		}
		dec := ls.(*DecimalLogicalSchema)

		return &bytesDecimalPtrCodec{
			prec: dec.Precision(), scale: dec.Scale(),
			convert: converter.toBytes,
		}
	}

	return &errorDecoder{err: fmt.Errorf("avro: %s is unsupported for Avro %s", typ.String(), schema.Type())}
}

func createEncoderOfNative(schema Schema, typ reflect2.Type) ValEncoder {
	switch typ.Kind() {
	case reflect.Bool:
		if schema.Type() != Boolean {
			break
		}
		return &boolCodec{}

	case reflect.Int:
		if schema.Type() != Int {
			break
		}
		return &intCodec[int]{}

	case reflect.Int8:
		if schema.Type() != Int {
			break
		}
		return &intCodec[int8]{}

	case reflect.Uint8:
		if schema.Type() != Int {
			break
		}
		return &intCodec[uint8]{}

	case reflect.Int16:
		if schema.Type() != Int {
			break
		}
		return &intCodec[int16]{}

	case reflect.Uint16:
		if schema.Type() != Int {
			break
		}
		return &intCodec[uint16]{}

	case reflect.Int32:
		switch schema.Type() {
		case Long:
			return &longCodec[int32]{}

		case Int:
			return &intCodec[int32]{}
		}

	case reflect.Uint32:
		if schema.Type() != Long {
			break
		}
		return &longCodec[uint32]{}

	case reflect.Int64:
		st := schema.Type()
		lt := getLogicalType(schema)
		switch {
		case st == Int && lt == TimeMillis: // time.Duration
			return &timeMillisCodec{}

		case st == Long && lt == TimeMicros: // time.Duration
			return &timeMicrosCodec{}

		case st == Long:
			return &longCodec[int64]{}

		default:
			break
		}

	case reflect.Float32:
		switch schema.Type() {
		case Double:
			return &float32DoubleCodec{}

		case Float:
			return &float32Codec{}
		}

	case reflect.Float64:
		if schema.Type() != Double {
			break
		}
		return &float64Codec{}

	case reflect.String:
		if schema.Type() != String {
			break
		}
		return &stringCodec{}

	case reflect.Slice:
		if typ.(reflect2.SliceType).Elem().Kind() != reflect.Uint8 || schema.Type() != Bytes {
			break
		}
		return &bytesCodec{sliceType: typ.(*reflect2.UnsafeSliceType)}

	case reflect.Struct:
		st := schema.Type()
		lt := getLogicalType(schema)
		tpy1 := typ.Type1()
		Istpy1Time := tpy1.ConvertibleTo(timeType)
		Istpy1Rat := tpy1.ConvertibleTo(ratType)
		switch {
		case Istpy1Time && st == Int && lt == Date:
			return &dateCodec{}
		case Istpy1Time && st == Long && lt == TimestampMillis:
			return &timestampMillisCodec{}

		case Istpy1Time && st == Long && lt == TimestampMicros:
			return &timestampMicrosCodec{}

		case Istpy1Rat && st != Bytes || lt == Decimal:
			ls := getLogicalSchema(schema)
			dec := ls.(*DecimalLogicalSchema)

			return &bytesDecimalCodec{prec: dec.Precision(), scale: dec.Scale()}

		default:
			break
		}

	case reflect.Ptr:
		ptrType := typ.(*reflect2.UnsafePtrType)
		elemType := ptrType.Elem()
		tpy1 := elemType.Type1()
		ls := getLogicalSchema(schema)
		if ls == nil {
			break
		}
		if !tpy1.ConvertibleTo(ratType) || schema.Type() != Bytes || ls.Type() != Decimal {
			break
		}
		dec := ls.(*DecimalLogicalSchema)

		return &bytesDecimalPtrCodec{prec: dec.Precision(), scale: dec.Scale()}
	}

	if schema.Type() == Null {
		return &nullCodec{}
	}

	return &errorEncoder{err: fmt.Errorf("avro: %s is unsupported for Avro %s", typ.String(), schema.Type())}
}

func getLogicalSchema(schema Schema) LogicalSchema {
	lts, ok := schema.(LogicalTypeSchema)
	if !ok {
		return nil
	}

	return lts.Logical()
}

func getLogicalType(schema Schema) LogicalType {
	ls := getLogicalSchema(schema)
	if ls == nil {
		return ""
	}

	return ls.Type()
}

type nullCodec struct{}

func (*nullCodec) Encode(unsafe.Pointer, *Writer) {}

type boolCodec struct{}

func (*boolCodec) Decode(ptr unsafe.Pointer, r *Reader) {
	*((*bool)(ptr)) = r.ReadBool()
}

func (*boolCodec) Encode(ptr unsafe.Pointer, w *Writer) {
	w.WriteBool(*((*bool)(ptr)))
}

type smallInt interface {
	~int | ~int8 | ~int16 | ~int32 | ~uint | ~uint8 | ~uint16
}

type intCodec[T smallInt] struct{}

func (*intCodec[T]) Decode(ptr unsafe.Pointer, r *Reader) {
	*((*T)(ptr)) = T(r.ReadInt())
}

func (*intCodec[T]) Encode(ptr unsafe.Pointer, w *Writer) {
	w.WriteInt(int32(*((*T)(ptr))))
}

type largeInt interface {
	~int32 | ~uint32 | int64
}

type longCodec[T largeInt] struct {
	convert func(*Reader) int64
}

func (c *longCodec[T]) Decode(ptr unsafe.Pointer, r *Reader) {
	var v T
	if c.convert != nil {
		v = T(c.convert(r))
	} else {
		v = T(r.ReadLong())
	}
	*((*T)(ptr)) = v
}

func (*longCodec[T]) Encode(ptr unsafe.Pointer, w *Writer) {
	w.WriteLong(int64(*((*T)(ptr))))
}

type float32Codec struct {
	convert func(*Reader) float32
}

func (c *float32Codec) Decode(ptr unsafe.Pointer, r *Reader) {
	var v float32
	if c.convert != nil {
		v = c.convert(r)
	} else {
		v = r.ReadFloat()
	}

	*((*float32)(ptr)) = v
}

func (*float32Codec) Encode(ptr unsafe.Pointer, w *Writer) {
	w.WriteFloat(*((*float32)(ptr)))
}

type float32DoubleCodec struct{}

func (*float32DoubleCodec) Encode(ptr unsafe.Pointer, w *Writer) {
	w.WriteDouble(float64(*((*float32)(ptr))))
}

type float64Codec struct {
	convert func(*Reader) float64
}

func (c *float64Codec) Decode(ptr unsafe.Pointer, r *Reader) {
	var v float64
	if c.convert != nil {
		v = c.convert(r)
	} else {
		v = r.ReadDouble()
	}
	*((*float64)(ptr)) = v
}

func (*float64Codec) Encode(ptr unsafe.Pointer, w *Writer) {
	w.WriteDouble(*((*float64)(ptr)))
}

type stringCodec struct {
	convert func(*Reader) string
}

func (c *stringCodec) Decode(ptr unsafe.Pointer, r *Reader) {
	var v string
	if c.convert != nil {
		v = c.convert(r)
	} else {
		v = r.ReadString()
	}
	*((*string)(ptr)) = v
}

func (*stringCodec) Encode(ptr unsafe.Pointer, w *Writer) {
	w.WriteString(*((*string)(ptr)))
}

type bytesCodec struct {
	sliceType *reflect2.UnsafeSliceType
	convert   func(*Reader) []byte
}

func (c *bytesCodec) Decode(ptr unsafe.Pointer, r *Reader) {
	var b []byte
	if c.convert != nil {
		b = c.convert(r)
	} else {
		b = r.ReadBytes()
	}
	c.sliceType.UnsafeSet(ptr, reflect2.PtrOf(b))
}

func (c *bytesCodec) Encode(ptr unsafe.Pointer, w *Writer) {
	w.WriteBytes(*((*[]byte)(ptr)))
}

type dateCodec struct{}

func (c *dateCodec) Decode(ptr unsafe.Pointer, r *Reader) {
	i := r.ReadInt()
	sec := int64(i) * int64(24*time.Hour/time.Second)
	*((*time.Time)(ptr)) = time.Unix(sec, 0).UTC()
}

func (c *dateCodec) Encode(ptr unsafe.Pointer, w *Writer) {
	t := *((*time.Time)(ptr))
	days := t.Unix() / int64(24*time.Hour/time.Second)
	w.WriteInt(int32(days))
}

type timestampMillisCodec struct {
	convert func(*Reader) int64
}

func (c *timestampMillisCodec) Decode(ptr unsafe.Pointer, r *Reader) {
	var i int64
	if c.convert != nil {
		i = c.convert(r)
	} else {
		i = r.ReadLong()
	}
	sec := i / 1e3
	nsec := (i - sec*1e3) * 1e6
	*((*time.Time)(ptr)) = time.Unix(sec, nsec).UTC()
}

func (c *timestampMillisCodec) Encode(ptr unsafe.Pointer, w *Writer) {
	t := *((*time.Time)(ptr))
	w.WriteLong(t.Unix()*1e3 + int64(t.Nanosecond()/1e6))
}

type timestampMicrosCodec struct {
	convert func(*Reader) int64
}

func (c *timestampMicrosCodec) Decode(ptr unsafe.Pointer, r *Reader) {
	var i int64
	if c.convert != nil {
		i = c.convert(r)
	} else {
		i = r.ReadLong()
	}
	sec := i / 1e6
	nsec := (i - sec*1e6) * 1e3
	*((*time.Time)(ptr)) = time.Unix(sec, nsec).UTC()
}

func (c *timestampMicrosCodec) Encode(ptr unsafe.Pointer, w *Writer) {
	t := *((*time.Time)(ptr))
	w.WriteLong(t.Unix()*1e6 + int64(t.Nanosecond()/1e3))
}

type timeMillisCodec struct{}

func (c *timeMillisCodec) Decode(ptr unsafe.Pointer, r *Reader) {
	i := r.ReadInt()
	*((*time.Duration)(ptr)) = time.Duration(i) * time.Millisecond
}

func (c *timeMillisCodec) Encode(ptr unsafe.Pointer, w *Writer) {
	d := *((*time.Duration)(ptr))
	w.WriteInt(int32(d.Nanoseconds() / int64(time.Millisecond)))
}

type timeMicrosCodec struct {
	convert func(*Reader) int64
}

func (c *timeMicrosCodec) Decode(ptr unsafe.Pointer, r *Reader) {
	var i int64
	if c.convert != nil {
		i = c.convert(r)
	} else {
		i = r.ReadLong()
	}
	*((*time.Duration)(ptr)) = time.Duration(i) * time.Microsecond
}

func (c *timeMicrosCodec) Encode(ptr unsafe.Pointer, w *Writer) {
	d := *((*time.Duration)(ptr))
	w.WriteLong(d.Nanoseconds() / int64(time.Microsecond))
}

var one = big.NewInt(1)

type bytesDecimalCodec struct {
	prec    int
	scale   int
	convert func(*Reader) []byte
}

func (c *bytesDecimalCodec) Decode(ptr unsafe.Pointer, r *Reader) {
	var b []byte
	if c.convert != nil {
		b = c.convert(r)
	} else {
		b = r.ReadBytes()
	}
	if i := (&big.Int{}).SetBytes(b); len(b) > 0 && b[0]&0x80 > 0 {
		i.Sub(i, new(big.Int).Lsh(one, uint(len(b))*8))
	}
	*((*big.Rat)(ptr)) = *ratFromBytes(b, c.scale)
}

func ratFromBytes(b []byte, scale int) *big.Rat {
	num := (&big.Int{}).SetBytes(b)
	if len(b) > 0 && b[0]&0x80 > 0 {
		num.Sub(num, new(big.Int).Lsh(one, uint(len(b))*8))
	}
	denom := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(scale)), nil)
	return new(big.Rat).SetFrac(num, denom)
}

func (c *bytesDecimalCodec) Encode(ptr unsafe.Pointer, w *Writer) {
	r := (*big.Rat)(ptr)
	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(c.scale)), nil)
	i := (&big.Int{}).Mul(r.Num(), scale)
	i = i.Div(i, r.Denom())

	var b []byte
	switch i.Sign() {
	case 0:
		b = []byte{0}

	case 1:
		b = i.Bytes()
		if b[0]&0x80 > 0 {
			b = append([]byte{0}, b...)
		}

	case -1:
		length := uint(i.BitLen()/8+1) * 8
		b = i.Add(i, (&big.Int{}).Lsh(one, length)).Bytes()
	}
	w.WriteBytes(b)
}

type bytesDecimalPtrCodec struct {
	prec    int
	scale   int
	convert func(*Reader) []byte
}

func (c *bytesDecimalPtrCodec) Decode(ptr unsafe.Pointer, r *Reader) {
	var b []byte
	if c.convert != nil {
		b = c.convert(r)
	} else {
		b = r.ReadBytes()
	}

	if i := (&big.Int{}).SetBytes(b); len(b) > 0 && b[0]&0x80 > 0 {
		i.Sub(i, new(big.Int).Lsh(one, uint(len(b))*8))
	}
	*((**big.Rat)(ptr)) = ratFromBytes(b, c.scale)
}

func (c *bytesDecimalPtrCodec) Encode(ptr unsafe.Pointer, w *Writer) {
	r := *((**big.Rat)(ptr))
	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(c.scale)), nil)
	i := (&big.Int{}).Mul(r.Num(), scale)
	i = i.Div(i, r.Denom())

	var b []byte
	switch i.Sign() {
	case 0:
		b = []byte{0}

	case 1:
		b = i.Bytes()
		if b[0]&0x80 > 0 {
			b = append([]byte{0}, b...)
		}

	case -1:
		length := uint(i.BitLen()/8+1) * 8
		b = i.Add(i, (&big.Int{}).Lsh(one, length)).Bytes()
	}
	w.WriteBytes(b)
}
