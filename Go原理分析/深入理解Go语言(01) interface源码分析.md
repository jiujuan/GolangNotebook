分析接口的赋值，反射，断言的实现原理

> 版本：golang v1.12


interface底层使用2个struct表示的：`eface`和`iface`


## 一：接口类型分为2个



### 1. 空接口


```
//比如
var i interface{}
```



### 2. 带方法的接口


```
//比如
type studenter interface {
    GetName() string
    GetAge()  int
}
```



## 二：eface 空接口定义

空接口通过`eface`结构体定义实现，位于`src/runtime/runtime2.go`

```
type eface struct {
	_type *_type //类型信息
	data  unsafe.Pointer //数据信息，指向数据指针
}
```

可以看到上面eface包含了2个元素，一个是_type，指向对象的类型信息，一个 data，数据指针


## 三：_type 结构体

`_type` 位于 `src/runtime/type.go`



`_type` 是go里面所有类型的一个抽象，里面包含GC，反射，大小等需要的细节，它也决定了data如何解释和操作。

里面包含了非常多信息 类型的大小、哈希、对齐以及种类等自动。

所以不论是空`eface`和非空`iface`都包含 `_type` 数据类型

```go
type _type struct {
	size       uintptr //数据类型共占用的空间大小
	ptrdata    uintptr //含有所有指针类型前缀大小
	hash       uint32  //类型hash值；避免在哈希表中计算
	tflag      tflag   //额外类型信息标志
	align      uint8   //该类型变量对齐方式
	fieldalign uint8   //该类型结构字段对齐方式   
	kind       uint8   //类型编号
	alg        *typeAlg //算法表 存储hash和equal两个操作。map key便使用key的_type.alg.hash(k)获取hash值
	// gcdata stores the GC type data for the garbage collector.
	// If the KindGCProg bit is set in kind, gcdata is a GC program.
	// Otherwise it is a ptrmask bitmap. See mbitmap.go for details.
	gcdata    *byte   //gc数据
	str       nameOff        // 类型名字的偏移
	ptrToThis typeOff
}
```



_type 中的一些数据类型如下：

```
// typeAlg is 总是 在 reflect/type.go 中 copy或使用.
// 并保持他们同步.
type typeAlg struct {
	// 算出该类型的Hash
	// (ptr to object, seed) -> hash
	hash func(unsafe.Pointer, uintptr) uintptr
	// 比较该类型对象
	// (ptr to object A, ptr to object B) -> ==?
	equal func(unsafe.Pointer, unsafe.Pointer) bool
}
type nameOff int32
type typeOff int32
```



但是各个类型需要的类型描叙是不一样的，比如chan，除了chan本身外，还需要描述其元素类型，而map则需要key类型信息和value类型信息等:

```
//src/runtime/type.go

type ptrtype struct {
	typ  _type
	elem *_type
}

type chantype struct {
	typ  _type
	elem *_type
	dir  uintptr
}

type maptype struct {
	typ        _type
	key        *_type
	elem       *_type
	bucket     *_type // internal type representing a hash bucket
	keysize    uint8  // size of key slot
	valuesize  uint8  // size of value slot
	bucketsize uint16 // size of bucket
	flags      uint32
}
```



看上面的类型信息，第一个自动都是 `_type`，接下来也定义了一堆类型所需要的信息（如子类信息）,这样在进行类型相关操作时，可通过一个字(typ *_type)即可表述所有类型，然后再通过_type.kind可解析出其具体类型，最后通过地址转换即可得到类型完整的”_type树”，参考reflect.Type.Elem()函数:

```
// reflect/type.go
// reflect.rtype结构体定义和runtime._type一致  type.kind定义也一致(为了分包而重复定义)
// Elem()获取rtype中的元素类型，只针对复合类型(Array, Chan, Map, Ptr, Slice)有效
func (t *rtype) Elem() Type {
	switch t.Kind() {
	case Array:
		tt := (*arrayType)(unsafe.Pointer(t))
		return toType(tt.elem)
	case Chan:
		tt := (*chanType)(unsafe.Pointer(t))
		return toType(tt.elem)
	case Map:
		tt := (*mapType)(unsafe.Pointer(t))
		return toType(tt.elem)
	case Ptr:
		tt := (*ptrType)(unsafe.Pointer(t))
		return toType(tt.elem)
	case Slice:
		tt := (*sliceType)(unsafe.Pointer(t))
		return toType(tt.elem)
	}
	panic("reflect: Elem of invalid type")
}
```



## 四：没有方法的interface赋值后内部结构

对于没有方法的interface赋值后的内部结构是怎样的呢？可以先看段代码：

```
import (
	"fmt"
	"strconv"
)

type Binary uint64

func main() {
	b := Binary(200)
	any := (interface{})(b)
	fmt.Println(any)
}
```



输出200，赋值后的结构图是这样的：

![go-principles-interface-type-data](../images/go-principles-interface-type-data-img.jpeg)

> 图片来自：[https://blog.csdn.net/i6448038/article/details/82916330](https://blog.csdn.net/i6448038/article/details/82916330)

对于将不同类型转化成type万能结构的方法，是运行时的`convT2E`方法，在runtime包中。以上，是对于没有方法的接口说明。

对于包含方法的函数，用到的是另外的一种结构，叫iface



## 五：iface 非空接口

iface结构体表示非空接口:

### iface


```go
// runtime/runtime2.go
// 非空接口
type iface struct {
    tab  *itab
    data unsafe.Pointer //指向原始数据指针
}
```



### itab

itab结构体是iface不同于eface，比较关键的数据结构

```kotlin
// runtime/runtime2.go
// 非空接口的类型信息
type itab struct {
    //inter 和 _type 确定唯一的 _type类型
    inter  *interfacetype    // 接口自身定义的类型信息，用于定位到具体interface类型
    _type  *_type        // 接口实际指向值的类型信息-实际对象类型，用于定义具体interface类型
    hash int32          //_type.hash的拷贝，用于快速查询和判断目标类型和接口中类型是一致
    _     [4]byte
    fun  [1]uintptr //动态数组，接口方法实现列表(方法集)，即函数地址列表，按字典序排序
                    //如果数组中的内容为空表示 _type 没有实现 inter 接口
                    
}
```



属性`interfacetype`类似于_type，其作用就是interface的公共描述，类似的还有`maptype`、`arraytype`、`chantype`…其都是各个结构的公共描述，可以理解为一种外在的表现信息。interfacetype源码如下：

```go
// runtime/type.go
// 非空接口类型，接口定义，包路径等。
type interfacetype struct {
   typ     _type
   pkgpath name
   mhdr    []imethod      // 接口方法声明列表，按字典序排序
}

// 接口的方法声明,一种函数声明的抽象
// 比如：func Print() error
type imethod struct {
   name nameOff          // 方法名
   ityp typeOff                // 描述方法参数返回值等细节
}

type nameOff int32
type typeOff int32
```


> method 存的是func 的声明抽象，而 itab 中的 fun 字段才是存储 func 的真实切片。


非空接口(iface)本身除了可以容纳满足其接口的对象之外，还需要保存其接口的方法，因此除了**data字段，iface通过`tab`字段描述非空接口的细节，包括接口方法定义，接口方法实现地址，接口所指类型**等。iface是非空接口的实现，而不是类型定义，iface的真正类型为`interfacetype`，其第一个字段仍然为描述其自身类型的`_type`字段。


## 六：iface整体结构图

![go-principles-interface-iface-structure](../images/go-principles-interface-iface-structure-img.jpeg)
> 图片来自：[https://blog.csdn.net/i6448038/article/details/82916330](https://blog.csdn.net/i6448038/article/details/82916330)

## 七：含有方法的interface赋值后的内部结构

含有方法的interface赋值后的内部结构是怎样的呢？



```
package main

import (
	"fmt"
	"strconv"
)

type Binary uint64
func (i Binary) String() string {
	return strconv.FormatUint(i.Get(), 10)
}

func (i Binary) Get() uint64 {
	return uint64(i)
}

func main() {
	b := Binary(200)
	any := fmt.Stringer(b)
	fmt.Println(any)
}
```

首先，要知道代码运行结果为:200。

其次，了解到fmt.Stringer是一个包含String方法的接口。

```
type Stringer interface {
	String() string
}
```

最后，赋值后接口Stringer的内部结构为：

![go-principles-interface-stringer-structure](../images/go-principles-interface-stringer-structure-img.jpeg)


## 八：参考：
- [https://wudaijun.com/2018/01/go-interface-implement/](https://wudaijun.com/2018/01/go-interface-implement/)

- [https://blog.csdn.net/i6448038/article/details/82916330#comments](https://blog.csdn.net/i6448038/article/details/82916330#comments)