
@REM 配置 windows cmd下 功能


echo "using make.cmd"

@echo off
@REM 如果第一个命令行参数是build
if "%1"=="build" (
    call :build
)else if "%1"=="clean" (
    call :clean
)else if "%1"=="install" (
    call :install   
)
goto :eof

@echo off
:preprocess
    echo Preprocessing...
    @REM 构建构建工具
    go build ./cmd/ybbuilder
    @REM 使用构建工具构建 代码资源
    ybbuilder.exe
    del ybbuilder.exe
    echo Done.
    goto :eof

@echo off
:build
    echo Building...
    @REM 预处理
    call :preprocess
    go build ./cmd/yb
    echo Done.
    goto :eof


@echo off
:clean
    echo Cleaning...
    @REM 删除构建产物 yb.exe
    del yb.exe
    del ybcli.exe
    del ybbuilder.exe
    echo Done.
    goto :eof


@echo off
:install
    echo Installing...
    @REM 预处理
    call :preprocess
    @REM 安装项目
    go install ./cmd/yb
    go install ./cmd/ybcli
    go install ./cmd/ybbuilder
    echo Done.
    goto :eof

@echo off
:install-release
    echo Installing...
    @REM 预处理
    call :preprocess
    @REM 安装项目
    go install -ldflags "-s -w" -tags=release ./cmd/yb
    go install -ldflags "-s -w" -tags=release ./cmd/ybcli
    go install -ldflags "-s -w" -tags=release ./cmd/ybbuilder
    echo Done.
    goto :eof
