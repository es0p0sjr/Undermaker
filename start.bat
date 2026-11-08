@echo off
cd /d "%~dp0"
if exist "dist\uc-print.exe" goto release
if exist "dist\uc-print-fixed.exe" goto fixedrelease
if not exist "data\manifest.json" goto setup
set "GOCACHE=%~dp0.go-cache"
set "GOPROXY=off"
set "GOTOOLCHAIN=local"
go run ./tools/ucdev --open
set "UC_PRINT_RESULT=%ERRORLEVEL%"
goto finish

:fixedrelease
"dist\uc-print-fixed.exe" --open
set "UC_PRINT_RESULT=%ERRORLEVEL%"
goto finish

:release
"dist\uc-print.exe" --open
set "UC_PRINT_RESULT=%ERRORLEVEL%"
goto finish

:setup
echo UC PRINT - FIRST-TIME SETUP
echo.
echo A complete card snapshot has not been installed yet.
echo The importer creates manifest.json after all data and images pass validation.
echo Do not create this file yourself. Running validate will not create it.
echo.
echo The artwork helper runs on the Undercards website in your browser.
echo You do NOT need to start UC Print to use it.
echo Instructions: tools\ucsync\BROWSER.md
echo Helper:       JSON\download-assets.js
echo Save images:  JSON\assets
echo.
echo After the helper says SUCCESS, run this in PowerShell from this folder:
echo go run ./tools/ucsync import ./JSON/AllCards.Json --translations ./JSON/en.json --assets ./JSON/assets
echo.
echo Then run start.bat again.
echo To test printing before setup is complete, run:
echo .\dist\uc-print-demo.exe --open
echo.
pause
exit /b 1

:finish
if not "%UC_PRINT_RESULT%"=="0" pause
exit /b %UC_PRINT_RESULT%
