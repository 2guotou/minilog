# minilog

File Based Log Golang Package, Power by Go Routine 'n' Channel Buffer, 
Rotate daily, Custom level, etc.

## Sample

```golang
package main

import (
    "fmt"
    "github.com/2guotou/minilog"
)

var mylog *minilog.Logger
var hislog *minilog.Logger

func main() {
    mylog = minilog.NewLogger("./", "myerror", 5000) //with 5k buffer
    hislog = minilog.NewLogger("./", "hiserror", 10) //with 10 buffer

    defer mylog.Close(1000)
    defer hislog.Close(1000)

    mylog.WithFileLine("FATL", "DEBG") //keep `Filename:LineNumber` info for FATAL\DEBG level

    //minilog support 5 built in function for 5 level: Info\Access\Error\Debug\Fatal
    //built in level : INFO\ACES\EROR\DEBG\FTAL
    //using `Write` function for custom level
    hislog.Info("I am just a foil~ ^_^")
    mylog.Info("Some Normal Infomartion")
    mylog.Access("Some One Reuqest My Server")
    mylog.Error("Trigger Some Error")
    mylog.Debug("For Debug with Filename and Line Number")
    mylog.Fatal("Wow, Dangerous!, also with Filename and Line Number")
    mylog.Write("SomeLevel", "Custom Level log record", minilog.EmptyIns, func(a, b string) {
        fmt.Printf("Callback, Args: %s, %s\n", a, b)
    })
}
```

./logs/myerror.2017-06-04.log

```log
2017-06-04 16:47:57 [INFO] Some Normal Infomartion
2017-06-04 16:47:57 [ACES] Some One Reuqest My Server
2017-06-04 16:47:57 [ERRO] Trigger Some Error
2017-06-04 16:47:57 [DEBG] For Debug with Filename and Line Number [/a/sample.go:16]
2017-06-04 16:47:57 [FATL] Wow, Dangerous!, also with Filename and Line Number [/a/sample.go:17]
2017-06-04 16:47:58 [SomeLevel] Custom Level log record
```

./logs/hiserror.2017-06-04.log

```log
2017-06-04 16:47:58 [INFO] I am just a foil~ ^_^
```