[golang常用库：gorilla/mux-http路由库使用](https://www.cnblogs.com/jiujuan/p/12768907.html)
[golang常用库：配置文件解析库/管理工具-viper使用](https://www.cnblogs.com/jiujuan/p/13799976.html)
[golang常用库：操作数据库的orm框架-gorm基本使用](https://www.cnblogs.com/jiujuan/p/12676195.html)
[golang常用库：字段参数验证库-validator使用](https://www.cnblogs.com/jiujuan/p/13823864.html)

> 说明：此文发布于 2020.01.12，此文中使用的 gorm 版本和你使用的 gorm 版本可能不同。所以 gorm 的最新使用请查看官方文档 https://gorm.io/zh_CN/docs/

## 一：字段映射-模型定义



gorm中通常用struct来映射字段. [gorm教程](https://gorm.io/zh_CN/docs/models.html)中叫`模型定义`

比如我们定义一个模型Model：

```go
type User struct {
	gorm.Model
	UserId      int64 `gorm:"index"` //设置一个普通的索引，没有设置索引名，gorm会自动命名
	Birtheday   time.Time
        Age         int           `gorm:"column:age"`//column:一个tag，可以设置列名称
        Name        string        `gorm:"size:255;index:idx_name_add_id"`//size:设置长度大小，index:设置索引，这个就取了一个索引名
	Num         int           `gorm:"AUTO_INCREMENT"`
        Email       string        `gorm:"type:varchar(100);unique_index"`//type:定义字段类型和大小
	AddressID   sql.NullInt64 `gorm:"index:idx_name_add_id"`
	IgnoreMe    int           `gorm:"_"`
	Description string        `gorm:"size:2019;comment:'用户描述字段'"`//comment：字段注释
	Status      string        `gorm:"type:enum('published', 'pending', 'deleted');default:'pending'"`
}
```

上面的gorm.Model 定义如下：

```go
type Model struct {
  ID        uint `gorm:"primary_key"`//primary_key:设置主键
  CreatedAt time.Time
  UpdatedAt time.Time
  DeletedAt *time.Time
}
```
当然我们也可以不用gorm.Model，自己定义一个差不多的类型

如果你用ID，系统会自动设为表的主键，当然我们可以自己定义主键：
比如：

```go
// 使用`AnimalID`作为主键
type Animal struct {
  AnimalID int64 `gorm:"primary_key"`
  Name     string
  Age      int64
}
```


> 参考：[https://gorm.io/zh_CN/docs/conventions.html](https://gorm.io/zh_CN/docs/conventions.html)


## 二：创建表

直接看下面的例子：createtable.go

```go
package main

import (
	"database/sql"
	"fmt"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"time"
)

type User struct {
	gorm.Model
	UserId      int64 `gorm:"index"`
	Birtheday   time.Time
	Age         int           `gorm:"column:age"`
	Name        string        `gorm:"size:255;index:idx_name_add_id"`
	Num         int           `gorm:"AUTO_INCREMENT"`
	Email       string        `gorm:"type:varchar(100);unique_index"`
	AddressID   sql.NullInt64 `gorm:"index:idx_name_add_id"`
	IgnoreMe    int           `gorm:"_"`
	Description string        `gorm:"size:2019;comment:'用户描述字段'"`
	Status      string        `gorm:"type:enum('published', 'pending', 'deleted');default:'pending'"`
}

//设置表名，默认是结构体的名的复数形式
func (User) TableName() string {
	return "VIP_USER"
}

func main() {
	db, err := gorm.Open("mysql", "root:root@/gormdemo?charset=utf8&parseTime=True&loc=Local")
	if err != nil {
		fmt.Println("connect db err: ", err)
	}
	defer db.Close()

	if db.HasTable(&User{}) { //判断表是否存在
		db.AutoMigrate(&User{}) //存在就自动适配表，也就说原先没字段的就增加字段
	} else {
		db.CreateTable(&User{}) //不存在就创建新表
	}
}

```


上面的gorm.Open()操作，如果想指定主机话，就需要加上括号 `()`

例如：

`user:password@(localhost)/dbname?charset=utf8&parseTime=True&loc=Local`

上面的程序中，先新建了一个数据库名叫 `gormdemo`，然后运行：`go run createtable.go` , 成功运行后，数据库就会出现一张名为 `vip_user` 的表。

## 三：增删改查

新建一个gormdemo的数据库，然后执行下面的sql语句，就会建立一个animals的表，里面还有一些测试数据

```sql
CREATE TABLE `animals` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `name` varchar(255) DEFAULT 'galeone',
  `age` int(10) unsigned DEFAULT '0',
  PRIMARY KEY (`id`)
) ENGINE=MyISAM AUTO_INCREMENT=8 DEFAULT CHARSET=utf8;

-- ----------------------------
-- Records of animals
-- ----------------------------
INSERT INTO `animals` VALUES ('1', 'demo-test', '20');
INSERT INTO `animals` VALUES ('2', 'galeone', '30');
INSERT INTO `animals` VALUES ('3', 'demotest', '30');
INSERT INTO `animals` VALUES ('4', 'jim', '90');
INSERT INTO `animals` VALUES ('5', 'jimmy', '10');
INSERT INTO `animals` VALUES ('6', 'jim', '23');
INSERT INTO `animals` VALUES ('7', 'test3', '27');
```


### 增加

例子：create.go

```go
package main

import (
	"fmt"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
)

type Animal struct {
	ID   int64
	Name string
	Age  int64
}

func main() {
	db, err := gorm.Open("mysql", "root:root@/gormdemo?charset=utf8&parseTime=true&loc=Local")
	if err != nil {
		fmt.Println("connect db error: ", err)
	}
	defer db.Close()

	animal := Animal{Name: "demo-test", Age: 20}
	db.Create(&animal)
}
```
说明：上面的这个例子，自己在mysql中创建一个animals的数据表，字段为id，name，age<br />

### 查找

select.go

```go
package main

import (
	"fmt"

	"github.com/jinzhu/gorm"

	_ "github.com/jinzhu/gorm/dialects/mysql"
)

type Animal struct {
	ID   int64
	Name string
	Age  int64
}

//https://gorm.io/zh_CN/docs/query.html
func main() {
	db, err := gorm.Open("mysql", "root:root@/gormdemo?charset=utf8&parseTime=true&loc=Local")
	if err != nil {
		fmt.Println("connect db error: ", err)
	}
	defer db.Close()

	//根据逐渐查询第一条记录
	var animal Animal
	db.First(&animal)
	fmt.Println(animal)

	//根据逐渐查询最后一条记录
	var animal2 Animal
	db.Last(&animal2)
	fmt.Println(animal2)

	//指定某条记录（仅当主键为整型时可用）
	var animal3 Animal
	db.First(&animal3, 2)
	fmt.Println(animal3)

	//where条件

	//符合条件的第一条记录
	var animal4 Animal
	db.Where("name = ?", "demotest2").First(&animal4)
	fmt.Println("where : ", animal4, animal4.ID, animal4.Name, animal4.Age)

	//符合条件的所有记录
	var animals5 []Animal
	db.Where("name = ?", "galeone").Find(&animals5)
	fmt.Println(animals5)
	for k, v := range animals5 {
		fmt.Println("k:", k, "ID:", v.ID, "Name:", v.Name, "Age:", v.Age)
	}

	//IN
	var animals6 []Animal
	db.Where("name IN (?)", []string{"demo-test", "demotest2"}).Find(&animals6)
	fmt.Println(animals6)

	//LIKE
	var animals7 []Animal
	db.Where("name like ?", "%jim%").Find(&animals7)
	fmt.Println(animals7)

	//AND
	var animals8 []Animal
	db.Where("name = ? AND age >= ?", "jim", "24").Find(&animals8)
	fmt.Println(animals8)

	//总数
	var count int
	var animals9 []Animal
	db.Where("name = ?", "galeone").Or("name = ?", "jim").Find(&animals9).Count(&count)
	fmt.Println(animals9)
	fmt.Println(count)

	//Scan, 原生查询
	var animals10 []Animal
	db.Raw("SELECT id, name, age From Animals WHERE name = ? AND age = ? ", "galeone", "30").Scan(&animals10)
	fmt.Println("Scan: ", animals10)

	//原生查询，select all
	var animals11 []Animal
	rows, _ := db.Raw("SELECT id,name FROM Animals").Rows()
	//注意：上面的 select id,name 后面不能写成 * 代替，不然出来的结果都是默认0值
	//像这样结果： ALL:  [{0  0} {0  0} {0  0} {0  0} {0  0} {0  0} {0  0}]
	//Scan 后面是什么字段，select 后面就紧跟什么字段
	for rows.Next() {
		var result Animal
		rows.Scan(&result.ID, &result.Name)
		animals11 = append(animals11, result)
	}
	fmt.Println("ALL: ", animals11)
	//output:ALL:  [{1 demo-test 0} {2 galeone 0} {3 demotest2 0} {4 galeone 0} {5 galeone 0} {6 jim 0} {7 jimmy 0}]

	//select 查询
	var animal12 Animal
	db.Select("name,age").Find(&animal12) //只查询name，age字段，相当于select name,age from user
	fmt.Println("select: ", animal12)
	// db.Select([]string{"name", "age"}).Find(&animal12)
	// fmt.Println("select2: ", animal12)
}

```

### 更新

update.go
```go
package main

import (
	"fmt"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
)

type Animal struct {
	ID   int64
	Name string
	Age  int64
}

func main() {
	db, err := gorm.Open("mysql", "root:root@/gormdemo?charset=utf8&parseTime=true&loc=Local")
	if err != nil {
		fmt.Println("connect db error: ", err)
	}
	defer db.Close()

	///根据一个条件更新
	//根据条件更新字段值,
	//后面加Debug()，运行时，可以打印出sql
	db.Debug().Model(&Animal{}).Where("id = ? ", 4).Update("name", "jimupdate")
	//UPDATE `animals` SET `name` = 'jimupdate'  WHERE (id = 4)

	//另外一种写法： 根据条件更新
	var animal Animal
	animal = Animal{ID: 3}
	db.Debug().Model(animal).Update("name", "demotest2update")
	// db.Debug().Model(&animal).Update("name", "demotest2update") // 这种写法也可以
	//UPDATE `animals` SET `name` = 'demotest2update'  WHERE `animals`.`id` = 3

	/// 多个条件更新
	db.Model(&Animal{}).Where("id = ? AND age = ?", 4, 45).Update("name", "jimupdate3")
	//UPDATE `animals` SET `name` = 'jimupdate2'  WHERE (id = 4 AND age = 45)

	/// 更新多个值
	db.Debug().Model(&Animal{}).Where("id = ?", 4).Update(Animal{Name: "jim", Age: 90})
	// UPDATE `animals` SET `age` = 90, `name` = 'jim'  WHERE (id = 4)

	animal2 := Animal{ID: 5}
	db.Debug().Model(&animal2).Update(map[string]interface{}{"name": "jimm", "age": 100})
	//UPDATE `animals` SET `age` = 100, `name` = 'jimm'  WHERE `animals`.`id` = 5
}

```

### 删除

delete.go

```go
package main

import (
	"fmt"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
)

type Animal struct {
	ID   int64
	Name string
	Age  int64
}

func main() {
	db, err := gorm.Open("mysql", "root:root@/gormdemo?charset=utf8&parseTime=true&loc=Local")
	if err != nil {
		fmt.Println("connect db error: ", err)
	}
	defer db.Close()

	db.Debug().Where("id = ?", 13).Delete(&Animal{})
	// DELETE FROM `animals`  WHERE (id = 13)

	db.Debug().Delete(&Animal{}, "id = ? AND age = ?", 14, 10)
	//DELETE FROM `animals`  WHERE (id = 14 AND age = 10)

}
```

## 四：Debug

在db后面直接加上 Debug()， 比如delete.go 里面的例子


## 五：参考

[https://gorm.io/zh_CN/](https://gorm.io/zh_CN/)