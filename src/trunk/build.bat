@echo off
setlocal enabledelayedexpansion

set EXE_NAME=PDUCalibrationTool.exe
set VERSION=1.0.0
set ICON=gui/icon/hpx.ico
set BUILD_DATE=%date%
set BUILD_DATE=%BUILD_DATE:~0,-3%
set BUILD_DATE=!BUILD_DATE: =/!
set BUILD_TIME=%time%
set BUILD_TIME=%BUILD_TIME:~0,-3%
set BUILD_TIME=!BUILD_TIME: =0!

if "%1%"=="" (
    echo building...
    rsrc -ico %ICON% -arch amd64 -o rsrc.syso
    go build -buildvcs=false -ldflags "-X main.version=%VERSION% -X main.buildTime=%BUILD_TIME% -X main.buildDate=%BUILD_DATE%" -o "%EXE_NAME%"
    echo build finish
    goto :eof
)

if "%1%"=="clean" (
    echo cleaning...
    del /Q "%EXE_NAME%" 2>nul
    del /Q rsrc.syso 2>nul
    echo clean finish
    goto :eof
)

if "%1%"=="package" (
    set PACKAGE_DIR=publish\!EXE_NAME:.exe=! v%VERSION%

    if not exist "%EXE_NAME%" (
        echo no found %EXE_NAME%
        goto :eof
    )

    md "!PACKAGE_DIR!" 2>nul  :: 2>nul

    echo packaging...
    copy /Y "%EXE_NAME%" "!PACKAGE_DIR!\" >nul 2>nul
    xcopy /E /Y /I "dll" "!PACKAGE_DIR!\dll\" >nul 2>nul
    xcopy /E /Y /I "configs" "!PACKAGE_DIR!\configs\" >nul 2>nul
    xcopy /E /Y /I "gui\icon" "!PACKAGE_DIR!\gui\icon\" >nul 2>nul
    echo package finish
    goto :eof
)

echo unknown cmd %1%

goto :eof