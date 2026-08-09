## 核心规范

1. 文件命名规范：文件名全部小写，以**`_`**做分词，不要使用拼音和无关单词。
2. 文件夹命名规范：文件夹名全部小写，不要包含下划线和数字，不要使用拼音和无关单词。
3. 变量命名规范：驼峰命名，首字母小写，不要包含下划线和数字，不要使用拼音和无关单词。
4. 常量命名规范：驼峰命名，首字母大写，不要包含下划线和数字，不要使用拼音和无关单词。
5. 函数/结构体命名规范：驼峰命名，可导出首字母大写，不可导出小写，不要包含下划线和数字，不要使用拼音和无关单词。
6. 代码注释规范：注释是必要的，在复杂逻辑或者BUG修改处注释尽量详细，写清楚代码思路和问题点；代码有修
   1.  改时注释要及时更新。
7. 异常处理规范：遵循有错必处理的规则，禁止使用**`_`**忽略遇到的错误。返回错误时要提供清晰的错误信息。
8. 日志记录规范：当遇到错误或者**`recover panic`**的时候应使用日志包记录对应等级的错误。
9. 包命名规范：包命名应该是全部小写，不要使用下划线或混合大小写。

## 建议规范

1. 目录创建规范：建议不要在**`/internal`**以外的目录创建自定义文件和目录。
2. 单元测试规范：复杂逻辑或者头脑风暴产出的方法和函数建议编写单元测试函数并进行测试。
3. 全局变量规范：开发中建议避免使用全局变量。
4. 接口命名规范：接口的命名建议以"er"结尾，例如：`type Reader interface{}`
5. 接收器命名：当为方法定义接收器时不建议使用**`this`****、****`self`**来命名接收器。建议使用接收类型名称首字母小写。例如：`func (s *Struct) methodName()。`
6. 错误变量检查：在处理错误时，建议使用具体的错误变量检查，而不是直接使用字符串比较。例如：
   1.  `if err == ErrRecordNotFound {}`
7. 错误处理返回值：在函数有多个返回值时，建议将错误作为最后一个返回值。例如：
   1.  `func Func() (Type,error)`
8. 空指针检查：在访问指针类型的字段或调用方法之前，建议检查指针是否为nil，以避免空指针异常。
9. Panic使用：建议尽量不要使用panic，除非你知道你在做什么。

## 最佳实践

 https://blog.csdn.net/weixin_43955067/article/details/130657921

 https://blog.csdn.net/ZKQDZKQD/article/details/127230653

####  提交内容  代码提交的时候一定要写清楚自己完成了那些功能、修改了什么BUG等，不要直接写个fix或者add直接提交。

####  推送频率  推送频率建议每天下班之前推送一次自己当天完成的代码，防止出现电脑系统闲时自动更新有可能对硬盘带来的不可逆的损坏，或者是某些场景下对代码造成不可逆的覆盖。

###  命名规范

####   文件命名规范

- 全部小写
- 文件名不要太长

####   变量命名规范

  这里主要针对局部变量，函数的入参、出参的规范

- 首字母小写

  - ```Go
    name := "张三" //正确
    Name := "张三" //错误
    ```

- 驼峰命名

  - ```Go
    updateAt := "张三" //正确
    updateat := "张三" //错误
    ```

- 不要使用拼音，不要使用与变量本身不相关的命名

  - ```Go
    realName := "张三" //正确
    xingming := "张三" //拼音 错误
    userAge := "张三"  //不想关命名 错误
    ```

- 不要包含下划线

  - ```Go
    realName := "张三" //正确
    real_name := "张三" //下划线 错误
    ```

- 不要包含数字

  - ```Go
    realName := "张三"  //正确
    realName1 := "张三" //包含数字 错误
    zhang3 := "张三"    //包含数字 错误
    ```

####   函数命名规范

- 驼峰命名

  - ```Go
    // getUserInfo 驼峰 正确
    func getUserInfo() {
        return
    }
    
    // get_user_info 非驼峰，带下划线 错误
    func get_user_info() {
        return
    }
    ```

- 可导出的首字母大写

- 禁止导出的首字母小写

  - ```Go
    // getUserInfo 不可导出
    func (e *Example) getUserInfo() {
        return
    }
    
    // GetUserInfo 可导出
    func (e *Example) GetUserInfo() {
        return
    }
    ```

####   常量命名规范

- 大写开头驼峰命名

  - ```Go
    const (
        MinPoolSize = 10
        MaxPoolSize = 20
        AvgPoolSize = 15
    )
    ```

###  编码规范

####   代码注释

  在开发时，注释的作用至关重要，它能够提高代码的可读性，也能够帮助其他开发者更好地理解代码的功能和逻辑。

- 注释格式

  - ```JSON
    // 单行注释
    /*
        多行注释
        多行注释
    */
    ```

- 如果是函数或者方法的注释 注释开头应该添加方法或者函数的名称

  - ```JSON
    // BufferSet 单行注释
    func BufferSet() {
    
    }
    
    // BufferSet 注释
    func (e *Example) BufferSet() {
    
    }
    ```

- 注释内容要清晰明了

  -    注释是为了帮助其他开发者更容易理解代码的开发思路和作用，注释的时候应避免使用晦涩难懂的术语，如果使用，请务必写出术语代表的含义。

- 注释位置

  -    注释的位置要在代码的上面或者旁边，如果注释过长，可以放到函数或者方法的头部

- 注释应该和代码同步

  -    当代码产生修改和更新时，应同步修改对应的注释，避免出现代码和注释内容对不上的问题。
  -    在注释中也可以添加修改记录和备注，以便于其他开发者更容易理解代码修改的原因和过程。

- 复杂场景和BUG修改场景必须写注释

  -    复杂场景指的是代码逻辑比较繁琐或者代码完成难度比较高的场景。
  -    BUG修改场景指的是针对已有的代码进行BUG修改时注释必须要写，而且需要加上修改时间

####   日志记录

#####    日志等级

   日志模块有以下几个等级，对应不同的场景，开发中要注意区分使用。

- **`Debug`****（调试）**
  - ​    `Debug`用于记录应用程序在调试模式下的一些状态信息，例如请求处理时间、请求参数等。这些信息主要用于辅助开发者进行调试，并不适合在正式环境中开启。
- **`Info`****（提示）**
  - ​    `Info`用于记录应用程序正常运行时的一些状态信息，例如系统启动完成、请求处理完成等。
- **`Warn`****（警告）**
  - ​    `Warn`用于记录一些异常情况，这些异常情况不会导致应用程序出现错误，但需要引起开发者的注意，例如出现网络连接异常者、传参错误等。
- **`Error`****（错误）**
  - ​    `Error`用于记录已经发生的错误情况，例如文件无法打开、数据无法读取等。这些错误不会导致应用程序崩溃，但需要被开发者及时处理。
- **`Fatal`****（致命）**
  - ​    `Fatal`用于记录导致应用程序无法正常运行的错误，这些常见的错误包括数组越界、空对象等可以被`recover`接收的错误。

#####    日志内容

   日志的内容决定了这条日志是否是有效日志，简单的调试和提示的日志内容只需要包含日志发生事件即可。

   但是`Warn,Error,Fatal`等级别的日志，需要记录相关方法或者函数的入参、调用的堆栈信息等敏感信息。

   参考：[Go 每日一库之 zap 高性能设计与实现](https://mp.weixin.qq.com/s/6dLtjHbtDRekVHC8G_Rv9w)

   参考：[zap-高性能日志库](https://github.com/uber-go/zap)

####   目录创建和命名

#####    知识拓展

   最近Golang官方也出了一版组织Go模块指南，其中推荐把项目核心代码放入`internal`目录，之前Go社区有

   一版比较受欢迎的规范，其中也是推荐把项目核心代码放入`internal`目录，由于社区版规范比较全面，这里对社

   区版规范展开描述。

   参考：[社区目录规范](https://gitcode.com/mirrors/golang-standards/project-layout/overview)

   参考：[官方目录指南](https://go.dev/doc/modules/layout)

######     目录结构图

######     主要目录

- **`/cmd`**
  - ​     该项目的主要应用。
  - ​     每个应用程序的目录名称应与您想要的可执行文件的名称相匹配（例如，`/cmd/myapp`）。
  - ​     不要在应用程序目录中放置大量代码。如果您认为代码可以导入并在其他项目中使用，那么它应该位于该`/pkg`目录中。如果代码不可重用或者您不希望其他人重用它，请将该代码放入目录中`/internal`。你会对别人的做法感到惊讶，所以要明确表达你的意图！
  - ​     通常有一个小函数从和目录`main`导入和调用代码，而不是其他任何东西。`/internal/pkg`
- **`/internal`**
  - ​     私有应用程序和库代码。这是您不希望其他人在其应用程序或库中导入的代码。请注意，这种布局模式是由 Go 编译器本身强制执行的。`release notes`有关更多详细信息，请参阅 Go 1.4 。请注意，您不仅限于顶级`internal`目录。您的项目树的任一级别都可以有多个`internal`目录。
  - ​     您可以选择向内部包添加一些额外的结构，以分隔共享和非共享的内部代码。这不是必需的（特别是对于较小的项目），但最好有视觉线索显示预期的包用途。您的实际应用程序代码可以放在`/internal/app`目录中（例如，`/internal/app/myapp`），而这些应用程序共享的代码可以放在`/internal/pkg`目录中（例如，`/internal/pkg/myprivlib`）。
- **`/pkg`**
  - ​     可以由外部应用程序使用的库代码（例如，`/pkg/mypubliclib`）。其他项目将导入这些库，期望它们能够工作，所以在你在这里放置东西之前请三思而后行:-) 请注意，该`internal`目录是确保你的私有包不可导入的更好方法，因为它是由 Go 强制执行的。该`/pkg`目录仍然是明确传达该目录中的代码可供其他人安全使用的好方法。Travis Jeffery 的博客`I'll take pkg over internal`文章很好地概述了`pkg`和`internal`目录以及何时使用它们可能有意义。
  - ​     当根目录包含大量非 Go 组件和目录时，这也是一种将 Go 代码分组到一个位置的方法，从而更容易运行各种 Go 工具（如这些演讲中提到的：来自 GopherCon EU 2018、GopherCon 2018：Kat `Best Practices for Industrial Programming`Zien [-如何构建您的 Go 应用程序](https://www.youtube.com/watch?v=oL6JBUk6tj0)和[GoLab 2018 - Massimiliano Pippi - Go 中的项目布局模式](https://www.youtube.com/watch?v=3gQa1LWwuzk)）。
  - ​     `/pkg`如果您想了解哪些流行的 Go 存储库使用此项目布局模式，请参阅目录。这是一种常见的布局模式，但并未得到普遍接受，并且 Go 社区中的一些人不推荐它。
  - ​     如果您的应用程序项目非常小并且额外的嵌套级别不会增加太多价值（除非您真的想要:-)），那么不使用它也没关系。当它变得足够大并且您的根目录变得非常繁忙时（特别是如果您有很多非 Go 应用程序组件），请考虑一下。
  - ​     目录`pkg`起源：旧的 Go 源代码用于`pkg`其包，然后社区中的各个 Go 项目开始复制该模式（`this`有关更多上下文，请参阅 Brad Fitzpatrick 的推文）。
- **`/vendor`**
  - ​     应用程序依赖项（手动管理或通过您最喜欢的依赖项管理工具（如新的内置`Go Modules`功能）进行管理）。该命令将为您`go mod vendor`创建目录。请注意，如果您不使用默认启用的 Go 1.14，则可能需要在命令中`/vendor`添加该标志。`-mod=vendorgo build`
  - ​     如果您正在构建库，请不要提交应用程序依赖项。
  - ​     请注意，由于`1.13`Go 还启用了模块代理功能（`https://proxy.golang.org`默认情况下用作其模块代理服务器）。阅读更多相关信息，`here`看看它是否符合您的所有要求和限制。`vendor`如果是这样，那么您根本不需要该目录。

######     API目录

- **`/api`**

​    OpenAPI/Swagger 规范、JSON 模式文件、协议定义文件。

​    请参阅`/api`目录中的示例。

######     通用目录

- **`/configs`**

​    配置文件模板或默认配置。

​    将您的文件`confd`或`consul-template`模板文件放在这里。

- **`/init`**

​    系统 init（systemd、upstart、sysv）和进程管理器/主管（runit、supervisord）配置。

- **`/scripts`**

​    用于执行各种构建、安装、分析等操作的脚本。

​    这些脚本使根级 Makefile 保持小而简单（例如，https://github.com/hashicorp/terraform/blob/main/Makefile）。

- **`/build`**

​    打包和持续集成。CICD目录。

- **`/deployments`**

​    IaaS、PaaS、系统和容器编排部署配置和模板（docker-compose、kubernetes/helm、terraform）。请注意，在某些存储库（尤其是使用 kubernetes 部署的应用程序）中，此目录称为`/deploy`.

- **`/test`**

​    其他外部测试应用程序和测试数据。您可以随意构建`/test`目录。对于较大的项目，有一个数据子目录是有意义的。例如，您可以使用`/test/data`或`/test/testdata`if you need Go 来忽略该目录中的内容。请注意，Go 还会忽略以“.”开头的目录或文件。或“_”，这样您就可以更灵活地命名测试数据目录。

​    请参阅`/test`目录中的示例。

######     其他目录

- **`/docs`**

​    设计和用户文档（除了 godoc 生成的文档之外）。

​    请参阅`/docs`目录中的示例。

- **`/tools`**

​    该项目的支持工具。`/pkg`请注意，这些工具可以从和目录导入代码`/internal`。

​    请参阅`/tools`目录中的示例。

- **`/examples`**

​    您的应用程序和/或公共库的示例。

​    请参阅`/examples`目录中的示例。

- **`/third_party`**

​    外部帮助工具、分叉代码和其他第 3 方实用程序（例如 Swagger UI）。

- **`/assets`**

​    与您的存储库一起使用的其他资产（图像、徽标等）。

- **`/website`**

​    如果您不使用 GitHub 页面，则可以在此处放置项目的网站数据。

​    请参阅`/website`目录中的示例。

#####    **`go-zero`**目录

**`go-zero`**有一套独特的项目结构，一般都是通过**`goctl`**工具生成

######     工程目录`├── consumer ├── go.mod ├── internal │   └── model ├── job ├── pkg ├── restful ├── script └── service`

######      **`consumer`**： 队列消费服务

######      **`internal`**： 工程内部可访问的公共模块

######      **`job`**： 定时任务服务

######      **`pkg`**： 工程外部可访问的公共模块

######      **`restful`**：HTTP 服务目录，下存放以服务为维度的微服务

######      **`script`**：脚本服务目录，下存放以脚本为维度的服务

######      **`service`**：gRPC 服务目录，下存放以服务为维度的微服务

######     服务目录`example ├── etc │   └── example.yaml ├── main.go └── internal    ├── config    │   └── config.go    ├── handler    │   ├── xxxhandler.go    │   └── xxxhandler.go    ├── logic    │   └── xxxlogic.go    ├── svc    │   └── servicecontext.go    └── types        └── types.go`**`example`**：单个服务目录，一般是某微服务名称**`etc`**：静态配置文件目录**`main.go`**：程序启动入口文件**`internal`**：单个服务内部文件，其可见范围仅限当前服务**`config`**：静态配置文件对应的结构体声明目录**`handler`**：handler 目录，可选，一般 http 服务会有这一层做路由管理，**`handler`** 为固定后缀**`logic`**：业务目录，所有业务编码文件都存放在这个目录下面，**`logic`** 为固定后缀**`svc`**：依赖注入目录，所有 logic 层需要用到的依赖都要在这里进行显式注入**`types`**：结构体存放目录

#####    主要规范**`/internal`**     **`/internal`**作为主要的核心目录，包括但不限于**`/model`****、****`/api`****、****`/db`****、****`/tools`****、****`/handler`**等目录，写业务时请不要在**`/internal`**以外的目录创建其他目录或文件**。****命名规则**全部小写除单元测试文件外不要使用下划线 **`_`**和**`-`**可以包含数字但是不要以数字开头不要起与文件功能无关的命名不要使用拼音

####   单元测试

#####    概要

   Go 语言的单元测试默认采用官方自带的测试框架，通过引入 `testing` 包以及 执行 `go test` 命令来实现单元测试功能。

   在源代码包目录内，所有以 `_test.go` 为后缀名的源文件会被 `go test` 认定为单元测试的文件，这些单元测试的文件不会包含在 `go build` 的源代码构建中，而是单独通过 `go test`来编译并执行。

#####    测试文件创建和命名

######     创建

​    测试文件的位置一定要和需要测试的代码文件在同一级别，不要在其他目录创建无关的测试文件。

```JSON
example
├── etc
├── main.go
└── internal
    ├── auth
    │   └── auth.go
    │   └── auth_test.go //正确
    ├── user
    │   └── user.go
    │   └── auth_test.go //错误 无关目录
```

######     命名

​    测试文件的名称是需要测试的代码文件名称加上`_test`，没有加`_test`的文件不会被`go test`命令识别。

```JSON
example
├── etc
├── main.go
└── internal
    ├── auth
    │   └── auth.go
    │   └── auth_test.go //正确
    │   └── auth-test.go //错误
    │   └── authtest.go //错误
```

#####    测试函数的命名和规范

######     逻辑测试     逻辑测试主要是测试代码的逻辑，能不能正常执行，有没有报错`/* TestBufferSet 单元测试 必须以Test开头 入参必须是t *testing.T 不能有返回值 */ func TestBufferSet(t *testing.T) {    e := &Example{}    bs := e.BufferSet(2, 3)    if bs != 6 {       t.Fatal("BufferSet error, except:6, result:", bs)    } }`     命令`//执行单测并打印详情 go test -v //执行以TestBufferSet为开头的测试并打印详情 go test -v -run='TestBufferSet' //执行单测并计算覆盖率（覆盖率：单测所覆盖的代码的比例） go test -v -cover //执行单测并开启代码分析并输出 //-cover 允许代码分析 //-covermode 代码分析模式（set：是否执行；count：执行次数；atomic：次数，并发执行） //-coverprofile 输出结果文件 go test -cover -coverprofile=cover.out -covermode=count //查看单测每个方法的覆盖率。 go tool cover -func=cover.out //-html会默认打开浏览器，将覆盖情况显示到页面中 go tool cover -html=cover.out //测试单个文件时需要执行测试文件和源文件 go test -v auth.go auth_test.go //只测试单个文件的某个方法 go test -v auth.go auth_test.go -run TestAuth //只测试单个文件下包含Auth的方法 go test -v auth.go auth_test.go -run 'Auth'`

######     基准测试（Benchmark）`/* BenchmarkGojaPoolV3PreCompile Goja引擎池预编译V3版本基准测试 基准测试函数必须以Benchmark开头 入参必须是 b *testing.B 不能有返回值 */ func BenchmarkGojaPoolV3PreCompile(b *testing.B) {    //此处可以写一些预处理的东西 不会对基准测试的结果造成影响    //do something    for n := 0; n < b.N; n++ {       test.GojaPoolBenchmarkV3()    } }`     命令`//测试当前目录下所有基准测试用例 go test -bench=. //测试当前目录下包含GojaPool的所有测试用例 go test -bench='GojaPool' //测试当前目录下名叫BenchmarkGojaPool的测试用例 go test -bench=BenchmarkGojaPool  //测试当前目录下名叫BenchmarkGojaPool的测试用例 //-benchtime**=**3s 运行三秒 go test -bench=BenchmarkGojaPool -benchtime**=**3s //测试当前目录下名叫BenchmarkGojaPool的测试用例 //-benchtime**=**3s 运行三秒 //-count=2 运行两轮 go test -bench=BenchmarkGojaPool -benchmem -benchtime**=**3s -count=2 //测试当前目录下名叫BenchmarkGojaPool的测试用例 //-benchtime**=**300x 循环运行300次 go test -bench=BenchmarkGojaPool -benchtime**=**300x //测试当前目录下名叫BenchmarkGojaPool的测试用例 *//-cpu****=****2,4 指定GOMAXPROCS的数量，模拟多核。分别2核和4核运行一次测试* go test -bench=BenchmarkGojaPool -cpu**=**2,4 //测试当前目录下所有基准测试用例 *//-benchmem 显示堆内存分配情况，分配的越多越影响性能* go test -bench= -benchmem``$ go test -bench='GojaPool' . -benchmem -benchtime=0.01s -count=1  goos: darwin goarch: amd64 pkg: service-engine cpu: Intel(R) Core(TM) i5-8257U CPU @ 1.40GHz BenchmarkGojaPool-8                             952        11656 ns/op       4824 B/op    89 allocs/op BenchmarkGojaPoolV2-8                          1015        10640 ns/op       4824 B/op    89 allocs/op BenchmarkGojaPoolV3PreCompile-8                2853         4234 ns/op       1831 B/op    31 allocs/op BenchmarkGojaPoolV4PreCompile-8                2688         3997 ns/op       1831 B/op    31 allocs/op BenchmarkGojaPoolV5PreCompileWeight-8         10000        10209 ns/op      10559 B/op   168 allocs/op BenchmarkGojaPoolV5PreCompileConcurrent-8     18799        18454 ns/op       9150 B/op   164 allocs/op PASS ok      service-engine  3.717s #第一列代表执行的基准测试用例名称 #第二列代表执行时间内循环执行的次数 #第三列代表每次执行所需的时间（单位：纳秒） #第四列代表每次执行分配了多少个字节 #第五列代表每次执行发生了多少个不同的内存分配`

####   函数返回

- 数据量比较大的对象避免非指针返回

  - ```Go
    // bigDataRight 大数据对象正确返回
    func (e *Example) bigDataRight() []*model.User {
        userList := make([]*model.User, 1000)
        for i := 0; i < 1000; i++ {
           user := model.UserInit("Ma", "Pony", 59, map[string]float64{"6530": 200000, "6714": 2000000}, "张小龙", "马云", "强子", "马子", "猴子", "狗子")
           userList = append(userList, user)
        }
        return userList
    }
    // bigDataWrong 大数据对象正确错误返回
    func (e *Example) bigDataWrong() []model.User {
        userList := make([]model.User, 1000)
        for i := 0; i < 1000; i++ {
           user := model.UserInit("Ma", "Pony", 59, map[string]float64{"6530": 200000, "6714": 2000000}, "张小龙", "马云", "强子", "马子", "猴子", "狗子")
           userList = append(userList, *user)
        }
        return userList
    }
    ```

- 出参有`error`的时候必须有正常值返回无`error`，有`error`无正常值返回

  - ```Go
    // hasErrorRight 带error正确返回
    func (e *Example) hasErrorRight() (string, error) {
        if true {
           return "张三", nil
        }
        return "", errors.New("error return")
    }
    
    // hasErrorWrong 带error错误返回
    func (e *Example) hasErrorWrong() (string, error) {
        if true {
           return "张三", errors.New("no error return")
        }
        return "张三", errors.New("error return")
    }
    ```

####   错误处理

- 避免直接使用下划线接收`error`

- 有`error`必须处理

  - ```Go
    func (e *Example) splitUserName() ([]string, error) {
        //禁止下划线接收
        name, _ := e.getUserName()
        //正确接收方式
        name, err := e.getUserName()
        if err != nil {
           return nil, err
        }
        return strings.Split(name, ""), nil
    }
    ```

####   字符串拼接

- 字符串拼接应使用`strings.Builder`或者`bytes.Buffer`不建议使用`fmt.Sprintf`或者`+`

  - ```Go
    // BuilderConcat Builder拼接字符串
    func BuilderConcat(str ...string) string {
        var builder strings.Builder
        for _, s := range str {
           builder.WriteString(s)
        }
        return builder.String()
    }
    
    // BufferConcat Buffer拼接字符串
    func BufferConcat(str ...string) string {
        buffer := new(bytes.Buffer)
        for _, s := range str {
           buffer.WriteString(s)
        }
        return buffer.String()
    }
    ```

####   `make`初始化 

- 使用make初始化切片、`map`、`channel`的时候，如果可以预知最大容量的时候，最好定义一下，可以减少频繁扩容对性能带来的影响。

  - ```Go
    func ConvertSliceToMap(userList []*model.User) map[int]*model.User {
        //定义初始容量
        //避免容量不足出发的扩容
        userMap := make(map[int]*model.User, len(userList))
        for _, user := range userList {
           userMap[user.ID] = user
        }
        return userMap
    }
    ```

####   `defer`使用

  在os包或者使用sync.Mutex等需要关闭的场景中，建议使用defer关闭/释放对应资源，避免出现忘记释放导致的异常。

   **注意：defer是先入后出的，开发的时候注意执行顺序**

```Go
// ExampleMutex 锁例子
func ExampleMutex() {
    //定义锁
    lock := sync.Mutex{}
    //释放锁
    defer lock.Unlock()

    //加锁
    lock.Lock()

    //do something
}
```

####   并发安全

  所谓并发安全就是在Go服务运行过程中出现的多个协程同时操作一个`Map`、`Slice`等非并发安全的类型带来的不确定性。

  `Map`并发操作出现问题:

```Go
func main() {
    //初始化切片
    demo := make(map[int]int, 1000)
    go func() {
       for j := 0; j < 1000; j++ {
          demo[j] = j
       }
    }()

    go func() {
       for j := 0; j < 1000; j++ {
          fmt.Println(demo[j])
       }
    }()

    time.Sleep(time.Second * 1)
}
```

  执行输出：

```Go
$ go run main.go 
fatal error: concurrent map read and map write
```

  建议方式`sync.Map`

```Go
func main() {
    //使用sync.Map
    demo := sync.Map{}
    //使用sync.WaitGroup阻塞主线程
    wg := sync.WaitGroup{}
    wg.Add(2)
    go func() {
       for j := 0; j < 1000; j++ {
          demo.Store(j, j)
       }
       wg.Done()
    }()

    go func() {
       for j := 0; j < 1000; j++ {
          fmt.Println(demo.Load(j))
       }
       wg.Done()
    }()
    
    //等待
    wg.Wait()
}
$ go run main.go
<nil> false
1 true
2 true
3 true

...

997 true
998 true
999 true
```

  建议方式`sync.Mutex`

```Go
func main() {
    //初始化Map
    demo := make(map[int]int, 1000)
    //使用读写锁
    lock := sync.RWMutex{}
    //使用sync.WaitGroup阻塞主线程
    wg := sync.WaitGroup{}
    wg.Add(2)
    go func() {
       for j := 0; j < 1000; j++ {
          lock.Lock()
          demo[j] = j
          lock.Unlock()
       }
       wg.Done()
    }()

    go func() {
       for j := 0; j < 1000; j++ {
          lock.RLock()
          fmt.Println(demo[j])
          lock.RUnlock()
       }
       wg.Done()
    }()
    
    //等待
    wg.Wait()
}
$ go run main.go
0
1
2

...

997
998
999
```

####   接收器

#####    命名

```Go
// 首字母小写命名 正确
func (e *Example) getExample() {
}

// this命名 不建议
func (this *Example) getExample() {
}

// self命名 不建议
func (self *Example) getExample() {
}
```

#####    类型

   编写结构体方法时，接受者的类型（Receiver Type）到底是选择值还是指针通常难以决定。如果你不知道使用什么类型，请参考如下建议：

- 当接受者是map、chan、func，不要使用指针传递，因为他们本身就是引用传递。
- 当函数内部需要修改接受者，必须使用指针传递。
- 当接受者是一个struct，并且包含了sync.Mutex或类似的用于同步的成员。必须使用指针传递，避免成员拷贝。
- 当接受者类型是一个struct并且很庞大，建议使用指针传递来提高性能。
- 当接受者是struct、array、slice，并且其中的元素是指针，并且内部可能修改这些元素，那么使用指针传递是个不错的选择，这能使的函数的语义更加明确。
- 当接受者是小型struct，小array，并且不需要修改里面的元素，里面的元素又是一些基础数据，使用值传递是个不错的选择。

todo 补充：

1. mq的redis名格式，精简防止占用内存
2. model层查单条、多条、错误处理以及返回
3. 常量不要采用硬编码
4. 初始化svc只初始化一次
5. 最小返回值err，优先返回值
6. 报错返回err时，不采用resp，resp改为nil
7. 调用http包外链接需要加超时时间，目前包有默认超时时间
8. 日志采用logz
9. 面条代码不采用，需要经常使用小方法
10. 方法名注释前需要概括一下，小写包内方法不需要概括
11. model查询列表时，数据量不大建议使用[]，数据量非常大的时候使用[]*
12. 参数过长时，可以考虑用struct收纳
13. 错误码需要记录一下