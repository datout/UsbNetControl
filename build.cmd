@echo off
setlocal

set VERSION=1.3.1
for /f "delims=" %%i in ('git describe --tags --always 2^>nul') do set VERSION=%%i
if "%VERSION:~0,1%"=="v" set VERSION=%VERSION:~1%

if not exist dist mkdir dist

set GOOS=windows
set CGO_ENABLED=0

set GOARCH=amd64
go build -trimpath -ldflags="-H=windowsgui -s -w -X main.appVersion=%VERSION%" -o dist\UsbNetControl-v%VERSION%-win-x64.exe .
if errorlevel 1 exit /b 1

set GOARCH=arm64
go build -trimpath -ldflags="-H=windowsgui -s -w -X main.appVersion=%VERSION%" -o dist\UsbNetControl-v%VERSION%-win-arm64.exe .
if errorlevel 1 exit /b 1

echo Built:
echo   dist\UsbNetControl-v%VERSION%-win-x64.exe
echo   dist\UsbNetControl-v%VERSION%-win-arm64.exe
